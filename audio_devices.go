package main

import (
	"fmt"
	"log"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// ── Audio Device Types ─────────────────────────────────────────────────────────

// AudioDevice represents a Windows audio endpoint (render or capture).
type AudioDevice struct {
	ID      string `json:"id"`      // Registry GUID key
	Name    string `json:"name"`    // Friendly name (e.g. "Realtek HD Audio")
	Type    string `json:"type"`    // "render" | "capture"
	HasAPO  bool   `json:"hasAPO"`  // Whether our APO is installed on this endpoint
	State   int    `json:"state"`   // 1=Active, 2=Disabled, 4=NotPresent, 8=Unplugged
	Default bool   `json:"default"` // Whether this is the default endpoint
}

const (
	apoClsid = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}"

	// Device name
	pkeyFriendlyName = "{a45c254e-df1c-4efd-8020-67d146a850e0},2"
	pkeyFXName       = "{b725f130-47ef-101a-a5f1-02608c9eebac},10"

	// Windows 10/11 FX keys
	pkeyFXStreamEffect = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},5" // SFX
	pkeyFXModeEffect   = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},6" // MFX
	pkeyFXEndpoint     = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},7" // EFX

	// Aliases para compatibilidad con el código que los lee
	pkeyFXPreMix  = pkeyFXStreamEffect
	pkeyFXPostMix = pkeyFXModeEffect

	renderBasePath  = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render`
	captureBasePath = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Capture`

	DeviceStateActive     = 1
	DeviceStateDisabled   = 2
	DeviceStateNotPresent = 4
	DeviceStateUnplugged  = 8
)

// ── App Methods (Wails bindings) ───────────────────────────────────────────────

// GetAudioDevices returns all render and capture endpoints with their APO status.
func (a *App) GetAudioDevices() ([]AudioDevice, error) {
	render, err := enumerateDevices("render")
	if err != nil {
		log.Printf("⚠️ Could not enumerate render devices: %v", err)
	}

	capture, err := enumerateDevices("capture")
	if err != nil {
		log.Printf("⚠️ Could not enumerate capture devices: %v", err)
	}

	all := append(render, capture...)
	return all, nil
}

// InstallAPOOnDevice installs our APO on a specific audio endpoint.
// deviceType must be "render" or "capture".
func (a *App) InstallAPOOnDevice(deviceID, deviceType string) error {
	basePath := resolveBasePath(deviceType)
	if basePath == "" {
		return fmt.Errorf("invalid device type: %s", deviceType)
	}

	fxPath := basePath + `\` + deviceID + `\FxProperties`
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("FxProperties access denied (Admin required): %w", err)
	}
	defer k.Close()

	for _, key := range []string{
		pkeyFXPreMix,       // ,0 legacy
		pkeyFXPostMix,      // ,1 legacy
		pkeyFXStreamEffect, // ,5 SFX
		pkeyFXModeEffect,   // ,6 MFX
		pkeyFXEndpoint,     // ,7 EFX
	} {
		if err := k.SetStringValue(key, apoClsid); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	k.SetStringValue(pkeyFXName, "ViPER4Windows APO")

	log.Printf("✓ APO installed on device %s (%s)", deviceID, deviceType)

	dm := &DriverManager{}
	return dm.RestartAudioEngine()
}

// UninstallAPOFromDevice removes our APO from a specific audio endpoint.
func (a *App) UninstallAPOFromDevice(deviceID, deviceType string) error {
	basePath := resolveBasePath(deviceType)
	if basePath == "" {
		return fmt.Errorf("invalid device type: %s", deviceType)
	}

	fxPath := basePath + `\` + deviceID + `\FxProperties`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		// Key doesn't exist — APO already not installed
		return nil
	}
	defer k.Close()

	// Remove pre-mix entry (ignore errors — key may not exist)
	_ = k.DeleteValue(pkeyFXPreMix)
	_ = k.DeleteValue(pkeyFXPostMix)

	log.Printf("✓ APO removed from device %s (%s)", deviceID, deviceType)

	// Restart audio to apply changes
	dm := &DriverManager{}
	if err := dm.RestartAudioEngine(); err != nil {
		log.Printf("⚠️ Audio restart warning: %v", err)
	}

	return nil
}

// InstallAPOOnAllRender installs the APO on every active render endpoint at once.
func (a *App) InstallAPOOnAllRender() error {
	devices, err := enumerateDevices("render")
	if err != nil {
		return fmt.Errorf("failed to enumerate render devices: %w", err)
	}

	var lastErr error
	for _, d := range devices {
		if d.State == DeviceStateActive {
			if err := a.InstallAPOOnDevice(d.ID, "render"); err != nil {
				log.Printf("⚠️ Could not install on %s: %v", d.Name, err)
				lastErr = err
			}
		}
	}
	return lastErr
}

// ── Internal helpers ───────────────────────────────────────────────────────────

func resolveBasePath(deviceType string) string {
	switch deviceType {
	case "render":
		return renderBasePath
	case "capture":
		return captureBasePath
	default:
		return ""
	}
}

// enumerateDevices reads all endpoints of the given type from the registry.
func enumerateDevices(deviceType string) ([]AudioDevice, error) {
	basePath := resolveBasePath(deviceType)
	if basePath == "" {
		return nil, fmt.Errorf("invalid device type: %s", deviceType)
	}

	k, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open registry path %s: %w", basePath, err)
	}
	defer k.Close()

	guids, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("failed to list device GUIDs: %w", err)
	}

	var devices []AudioDevice
	for _, guid := range guids {
		dev, err := readDeviceInfo(basePath, guid, deviceType)
		if err != nil {
			log.Printf("⚠️ Skipping device %s: %v", guid, err)
			continue
		}
		devices = append(devices, dev)
	}

	return devices, nil
}

// readDeviceInfo reads a single audio endpoint from the registry.
func readDeviceInfo(basePath, guid, deviceType string) (AudioDevice, error) {
	dev := AudioDevice{
		ID:   guid,
		Type: deviceType,
		Name: guid, // fallback if friendly name not found
	}

	// ── Read device state ──────────────────────────────────────────────────────
	deviceKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		basePath+`\`+guid, registry.READ)
	if err != nil {
		return dev, fmt.Errorf("failed to open device key: %w", err)
	}
	defer deviceKey.Close()

	state, _, err := deviceKey.GetIntegerValue("DeviceState")
	if err == nil {
		dev.State = int(state)
	}

	// ── Read friendly name ─────────────────────────────────────────────────────
	propsKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		basePath+`\`+guid+`\Properties`, registry.READ)
	if err == nil {
		defer propsKey.Close()
		if name, _, err := propsKey.GetStringValue(pkeyFriendlyName); err == nil {
			dev.Name = name
		}
	}

	// ── Check APO presence ─────────────────────────────────────────────────────
	fxKey, err := registry.OpenKey(registry.LOCAL_MACHINE,
		basePath+`\`+guid+`\FxProperties`, registry.READ)
	if err == nil {
		defer fxKey.Close()
		if val, _, err := fxKey.GetStringValue(pkeyFXPreMix); err == nil {
			dev.HasAPO = strings.EqualFold(val, apoClsid)
		}
	}

	return dev, nil
}
