package windows

import (
	"fmt"
	"strings"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"

	"golang.org/x/sys/windows/registry"
)

// ═══════════════════════════════════════════════════════════════════════════
// Device Management Service
// ═══════════════════════════════════════════════════════════════════════════
// Handles audio device enumeration and APO attachment via registry.
// Ported from original driveManager.go enumerateDevices/readDeviceInfo/AttachToEndpoint.

const (
	OutputBasePath    = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render`
	InputBasePath     = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Capture`
	pkeyFriendlyName  = "{a45c254e-df1c-4efd-8020-67d146a850e0},2"
	pkeyFXName        = "{b725f130-47ef-101a-a5f1-02608c9eebac},10"
	pkeyFXPreMix      = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},5"
	pkeyFXPostMix     = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},6"
	pkeyFXEndpoint    = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},7"
	DeviceStateActive = 1
)

type DeviceService struct {
	logger   ports.Logger
	security ports.SecurityPort
}

// NewDeviceService takes the shared SecurityService the same way
// NewAPOService now does — see the comment there for why this
// replaced a private isElevated/requireAdmin copy.
func NewDeviceService(logger ports.Logger, security ports.SecurityPort) *DeviceService {
	return &DeviceService{
		logger:   logger.WithContext("Devices"),
		security: security,
	}
}

// EnumerateDevices lists all audio endpoints of the given role
func (s *DeviceService) EnumerateDevices(role models.DeviceRole) ([]models.AudioDevice, error) {
	basePath := resolveBasePath(string(role))
	if basePath == "" {
		return nil, fmt.Errorf("invalid device role: %s", role)
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("open registry path %s: %w", basePath, err)
	}
	defer k.Close()

	guids, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("list device GUIDs: %w", err)
	}

	var devices []models.AudioDevice
	for _, guid := range guids {
		dev, err := s.readDeviceInfo(basePath, guid, string(role))
		if err != nil {
			s.logger.Debug("skip device", "guid", guid, "error", err)
			continue
		}
		dev.HasAPO = s.IsAPOAttached(guid)
		if legacy, clsid := s.checkLegacyResidue(guid); legacy {
			dev.LegacyResidue = true
			dev.LegacyCLSID = clsid
		}
		devices = append(devices, dev)
	}

	s.logger.Info("enumerated devices", "count", len(devices), "role", role)
	return devices, nil
}

// GetDefaultDevice returns the system default device for the given role
func (s *DeviceService) GetDefaultDevice(role models.DeviceRole) (*models.AudioDevice, error) {
	// Try to read from Sound Mapper
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Multimedia\Sound Mapper`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		var playback string
		if playback, _, err = k.GetStringValue("Playback"); err == nil && playback != "" {
			basePath := resolveBasePath(string(role))
			dev, err := s.readDeviceInfo(basePath, playback, string(role))
			if err == nil {
				return &dev, nil
			}
		}
	}

	// Fallback: return first active device
	devices, err := s.EnumerateDevices(role)
	if err != nil {
		return nil, err
	}
	for i := range devices {
		if devices[i].State == DeviceStateActive {
			devices[i].IsDefault = true
			return &devices[i], nil
		}
	}

	return nil, fmt.Errorf("no default device found for role %s", role)
}

// AttachAPO attaches the ViPER APO to a specific device via FxProperties.
//
// Before writing ViPER's CLSID over PreMix/PostMix/Endpoint/FXName, this
// reads whatever was there first and hands it to the chain-backup store
// (apo_chain_backup.go) — but only the first time for a given device, so
// a second Attach (e.g. the app re-asserting its own state on startup)
// doesn't overwrite the real backup with ViPER's own values. See
// DetachAPO for the other half of this — it used to just delete these
// three values unconditionally, which silently dropped whatever effects
// chain (if any) was registered before ViPER touched the endpoint.
func (s *DeviceService) AttachAPO(deviceID string) error {
	if err := s.security.RequireAdmin(); err != nil {
		return err
	}

	fxPath := OutputBasePath + `\` + deviceID + `\FxProperties`

	store, err := loadChainBackupStore()
	if err != nil {
		s.logger.Warn("chain backup: load failed, continuing without it", "error", err)
		store = chainBackupStore{Endpoints: map[string]fxChainBackup{}}
	}
	if _, alreadyBackedUp := store.Endpoints[deviceID]; !alreadyBackedUp {
		backup, err := captureCurrentFxChain(fxPath)
		if err != nil {
			return fmt.Errorf("read existing FxProperties before attach: %w", err)
		}
		store.Endpoints[deviceID] = backup
		if err := saveChainBackupStore(store); err != nil {
			s.logger.Warn("chain backup: save failed, detach will not be able to restore this endpoint", "error", err)
		}
	}

	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("FxProperties access denied: %w", err)
	}
	defer k.Close()

	for _, key := range []string{pkeyFXPreMix, pkeyFXPostMix, pkeyFXEndpoint} {
		if err := k.SetStringValue(key, ViPER_CLSID); err != nil {
			return err
		}
	}
	k.SetStringValue(pkeyFXName, "ViPER4Windows APO")

	s.logger.Info("APO attached to device", "device", deviceID)
	return nil
}

// DetachAPO removes the ViPER APO from a specific device, restoring
// whatever FxProperties values AttachAPO backed up — or deleting the
// key if it didn't exist originally — instead of unconditionally
// deleting everything. Falls back to the old delete-only behavior if
// there's no backup on file (installs from before this fix existed),
// so this can't make an existing setup worse, only better going
// forward.
func (s *DeviceService) DetachAPO(deviceID string) error {
	if err := s.security.RequireAdmin(); err != nil {
		return err
	}

	fxPath := OutputBasePath + `\` + deviceID + `\FxProperties`

	store, err := loadChainBackupStore()
	if err != nil {
		s.logger.Warn("chain backup: load failed, falling back to delete-only detach", "error", err)
		return legacyDeleteFxValues(fxPath)
	}

	backup, found := store.Endpoints[deviceID]
	if !found {
		s.logger.Warn("chain backup: no entry for device, falling back to delete-only detach", "device", deviceID)
		return legacyDeleteFxValues(fxPath)
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE)
	if err != nil {
		// Already gone — still drop the now-stale backup entry.
		delete(store.Endpoints, deviceID)
		_ = saveChainBackupStore(store)
		return nil
	}
	defer k.Close()

	restoreFxValue(k, pkeyFXPreMix, backup.PreMix)
	restoreFxValue(k, pkeyFXPostMix, backup.PostMix)
	restoreFxValue(k, pkeyFXEndpoint, backup.Endpoint)
	restoreFxValue(k, pkeyFXName, backup.FXName)

	delete(store.Endpoints, deviceID)
	if err := saveChainBackupStore(store); err != nil {
		s.logger.Warn("chain backup: save-after-restore failed (non-fatal)", "error", err)
	}

	s.logger.Info("APO detached from device", "device", deviceID)
	return nil
}

// IsAPOAttached checks if the ViPER APO is attached to a device
func (s *DeviceService) IsAPOAttached(deviceID string) bool {
	fxPath := OutputBasePath + `\` + deviceID + `\FxProperties`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.READ)
	if err != nil {
		return false
	}
	defer k.Close()

	if val, _, err := k.GetStringValue(pkeyFXPreMix); err == nil {
		return strings.EqualFold(val, ViPER_CLSID)
	}
	return false
}

// checkLegacyResidue reports whether PreMix points at some OTHER CLSID
// than ours — the exact "always shows installed but nothing works"
// symptom an old, separately-uninstalled ViPER4Windows leaves behind:
// its own configurator deregistered its CLSID from
// AudioEngine\AudioProcessingObjects, but never cleaned the per-device
// FxProperties pointer, so Windows still tries (and fails) to
// instantiate a CLSID nothing provides anymore.
func (s *DeviceService) checkLegacyResidue(deviceID string) (bool, string) {
	fxPath := OutputBasePath + `\` + deviceID + `\FxProperties`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.READ)
	if err != nil {
		return false, ""
	}
	defer k.Close()

	val, _, err := k.GetStringValue(pkeyFXPreMix)
	if err != nil || val == "" {
		return false, ""
	}
	if strings.EqualFold(val, ViPER_CLSID) {
		return false, ""
	}
	return true, val
}

// ── Internal helpers ───────────────────────────────────────────────────────────

func (s *DeviceService) readDeviceInfo(basePath, guid, deviceType string) (models.AudioDevice, error) {
	dev := models.AudioDevice{
		ID:         guid,
		DeviceType: deviceType,
		IsDefault:  false,
		Name:       guid, // Fallback
	}

	// Read device state
	deviceKey, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath+`\`+guid, registry.READ)
	if err != nil {
		return dev, fmt.Errorf("open device key: %w", err)
	}
	defer deviceKey.Close()

	if state, _, err := deviceKey.GetIntegerValue("DeviceState"); err == nil {
		dev.State = int(state)
	}

	// Read friendly name
	propsKey, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath+`\`+guid+`\Properties`, registry.READ)
	if err == nil {
		defer propsKey.Close()
		if name, _, err := propsKey.GetStringValue(pkeyFriendlyName); err == nil {
			dev.Name = name
		}
	}

	// Check APO presence
	fxKey, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath+`\`+guid+`\FxProperties`, registry.READ)
	if err == nil {
		defer fxKey.Close()
		if val, _, err := fxKey.GetStringValue(pkeyFXPreMix); err == nil {
			dev.IsEnabled = strings.EqualFold(val, ViPER_CLSID)
		}
	}

	return dev, nil
}

func resolveBasePath(deviceType string) string {
	switch deviceType {
	case "render":
		return OutputBasePath
	case "capture":
		return InputBasePath
	}
	return ""
}
