package windows

import (
	"fmt"
	"os"
	"path/filepath"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"

	"golang.org/x/sys/windows/registry"
)

// ═══════════════════════════════════════════════════════════════════════════
// APO Registration Service
// ═══════════════════════════════════════════════════════════════════════════
// Manages APO registration with Windows registry and DLL installation.
// Ported from original driveManager.go RegisterAPO/UnregisterAPO/CheckInstallation.

const (
	ViPER_CLSID = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}"
	ViPER_IID   = "{FD7F2B29-24D0-4B5C-B177-592C39F9CA10}"
)

type APOService struct {
	logger   ports.Logger
	security ports.SecurityPort
}

// NewAPOService wires in the shared SecurityService instead of
// re-implementing IsElevated/RequireAdmin here — this file,
// DeviceService, and (until this pass) cmd's old main.go each had
// their own copy of the same token-elevation check, and they'd
// already started drifting (this one's copy was missing the
// registry-fallback method SecurityService has). One implementation,
// injected everywhere it's needed.
func NewAPOService(logger ports.Logger, security ports.SecurityPort) *APOService {
	return &APOService{
		logger:   logger.WithContext("APO"),
		security: security,
	}
}

// CheckInstallation verifies if the APO is registered and DLL exists
func (s *APOService) CheckInstallation() bool {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		s.logger.Debug("APO not registered", "error", err)
		return false
	}
	defer k.Close()

	if dllPath, _, err := k.GetStringValue("Library"); err == nil {
		if fileExistsAndNotEmpty(dllPath) {
			s.logger.Debug("APO installed", "dll", dllPath)
			return true
		}
	}
	return false
}

// Install registers the APO with Windows and copies DLL to System32
func (s *APOService) Install(dllPath string) error {
	// Require admin privileges
	if err := s.security.RequireAdmin(); err != nil {
		return err
	}

	if !fileExistsAndNotEmpty(dllPath) {
		return fmt.Errorf("APO DLL invalid or empty: %s", dllPath)
	}

	s.logger.Info("installing APO", "source", dllPath)

	// Copy DLL to System32
	system32Path := filepath.Join(os.Getenv("SystemRoot"), "System32", filepath.Base(dllPath))
	dllBytes, err := os.ReadFile(dllPath)
	if err != nil {
		return fmt.Errorf("read DLL: %w", err)
	}
	if err := os.WriteFile(system32Path, dllBytes, 0644); err != nil {
		return fmt.Errorf("copy DLL to System32: %w", err)
	}
	s.logger.Debug("DLL copied", "dest", system32Path)

	// Create AudioProcessingObjects registry key
	apoPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create APO registry key: %w", err)
	}
	defer k.Close()

	k.SetStringValue("FriendlyName", "ViPER4Windows APO")
	k.SetStringValue("Copyright", "ViPER's Audio")
	k.SetStringValue("Library", system32Path)
	k.SetDWordValue("MajorVersion", 1)
	k.SetDWordValue("MinorVersion", 0)
	k.SetDWordValue("Flags", 0x0000000d)

	// AudioInterface0 subkey with IID
	ik, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath+`\AudioInterface0`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create AudioInterface0: %w", err)
	}
	ik.SetStringValue("IID", ViPER_IID)
	ik.Close()

	// InprocServer32 for COM
	comPath := `SOFTWARE\Classes\CLSID\` + ViPER_CLSID + `\InprocServer32`
	ck, _, err := registry.CreateKey(registry.LOCAL_MACHINE, comPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("create InprocServer32: %w", err)
	}
	ck.SetStringValue("", system32Path)
	ck.SetStringValue("ThreadingModel", "Both")
	ck.Close()

	s.logger.Info("APO registered successfully")
	return nil
}

// Uninstall removes APO registration and deletes DLL from System32
func (s *APOService) Uninstall() error {
	if err := s.security.RequireAdmin(); err != nil {
		return err
	}

	s.logger.Info("uninstalling APO")

	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID + `\AudioInterface0`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID,
		`SOFTWARE\Classes\CLSID\` + ViPER_CLSID + `\InprocServer32`,
		`SOFTWARE\Classes\CLSID\` + ViPER_CLSID,
	}

	for _, p := range paths {
		registry.DeleteKey(registry.LOCAL_MACHINE, p)
		s.logger.Debug("deleted registry key", "path", p)
	}

	// Remove DLL from System32
	dllPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "ViPERDSP.dll")
	_ = os.Remove(dllPath)
	s.logger.Debug("removed DLL", "path", dllPath)

	s.logger.Info("APO unregistered")
	return nil
}

// GetStatus returns the current APO status
func (s *APOService) GetStatus() models.APOStatus {
	status := models.APOStatus{
		IsInstalled: s.CheckInstallation(),
	}

	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return status
	}
	defer k.Close()

	if v, _, err := k.GetIntegerValue("MajorVersion"); err == nil {
		status.Version = fmt.Sprintf("%d.0", v)
	}
	if dllPath, _, err := k.GetStringValue("Library"); err == nil {
		status.DllPath = dllPath
	}
	return status
}

// ── Internal helpers ───────────────────────────────────────────────────────────

func fileExistsAndNotEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Size() > 0
}
