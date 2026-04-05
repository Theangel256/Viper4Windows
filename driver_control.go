package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	// APO Class ID for ViPER4Windows
	ViPER_CLSID = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}"
	// Interface ID for the APO
	ViPER_IID = "{FD7F2B29-24D0-4B5C-B177-592C39F9CA10}"

	// Registry paths for Windows Audio Endpoints
	AudioRenderKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render`

	// Property keys for device metadata
	PKEY_DeviceFriendlyName = "{a45c254e-df1c-4efd-8020-67d146a850e0},14"
)

// AudioEndpoint represents a Windows audio render device
type AudioEndpoint struct {
	GUID         string
	FriendlyName string
	IsDefault    bool
	HasViPER     bool
}

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}

// DriverManager handles the registration and lifecycle of the APO.
// Corresponds to the 'Installer' logic in ViPERDSP/Alpha.
type DriverManager struct{}

// ListAudioEndpoints enumerates all available audio render devices
func (dm *DriverManager) ListAudioEndpoints() ([]AudioEndpoint, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, AudioRenderKey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil, fmt.Errorf("cannot access audio devices registry: %w", err)
	}
	defer k.Close()

	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return nil, fmt.Errorf("cannot enumerate subkeys: %w", err)
	}

	endpoints := make([]AudioEndpoint, 0, len(names))
	for _, guid := range names {
		if ep, err := dm.readEndpointInfo(guid); err == nil {
			endpoints = append(endpoints, ep)
		}
	}

	return endpoints, nil
}

// readEndpointInfo retrieves metadata for a specific audio endpoint
func (dm *DriverManager) readEndpointInfo(guid string) (AudioEndpoint, error) {
	// Read friendly name from Properties
	propsPath := fmt.Sprintf(`%s\%s\Properties`, AudioRenderKey, guid)
	pk, err := registry.OpenKey(registry.LOCAL_MACHINE, propsPath, registry.QUERY_VALUE)

	name := "Unknown Device"
	if err == nil {
		if val, _, err := pk.GetStringValue(PKEY_DeviceFriendlyName); err == nil {
			name = val
		}
		pk.Close()
	}

	// Check if ViPER is already attached
	fxPath := fmt.Sprintf(`%s\%s\FxProperties`, AudioRenderKey, guid)
	fxk, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.QUERY_VALUE)

	hasViPER := false
	if err == nil {
		lfxKey := "{d04e05a6-594b-4fb6-a80d-01af5eedf162},0"
		if val, _, err := fxk.GetStringValue(lfxKey); err == nil && val == ViPER_CLSID {
			hasViPER = true
		}
		fxk.Close()
	}

	return AudioEndpoint{
		GUID:         guid,
		FriendlyName: name,
		IsDefault:    false, // Populated by GetDefaultEndpoint
		HasViPER:     hasViPER,
	}, nil
}

// GetDefaultEndpoint retrieves the GUID of the default playback device
func (dm *DriverManager) GetDefaultEndpoint() (string, error) {
	// Option 1: Read from User's Sound Mapper (unreliable)
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Multimedia\Sound Mapper`, registry.QUERY_VALUE)

	if err == nil {
		defer k.Close()
		if playback, _, err := k.GetStringValue("Playback"); err == nil {
			return playback, nil
		}
	}

	// Option 2: Fallback - find first device (requires better detection)
	endpoints, err := dm.ListAudioEndpoints()
	if err != nil || len(endpoints) == 0 {
		return "", fmt.Errorf("no audio endpoints found")
	}

	// For now, return first available (should use IMMDeviceEnumerator COM API)
	log.Printf("⚠️ Warning: Using first available endpoint as default (implement COM detection)")
	return endpoints[0].GUID, nil
}

// AttachToEndpoint binds the APO to a specific Audio Device (GUID).
// In ViPERFX_RE, this involves modifying the Multi-Endpoint (MME) properties.
func (dm *DriverManager) AttachToEndpoint(endpointID string) error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	// Path: HKLM\...\MMDevices\Audio\Render\{GUID}\FxProperties
	propPath := fmt.Sprintf(`%s\%s\FxProperties`, AudioRenderKey, endpointID)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, propPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("endpoint not found or access denied: %w", err)
	}
	defer k.Close()

	// These GUIDs represent the 'Pre-Mix' and 'Post-Mix' effect slots in Windows
	// {d04e05a6-594b-4fb6-a80d-01af5eedf162},0 = LFX (Local Effect)
	// {d04e05a6-594b-4fb6-a80d-01af5eedf162},1 = GFX (Global Effect)
	lfxKey := "{d04e05a6-594b-4fb6-a80d-01af5eedf162},0"
	if err := k.SetStringValue(lfxKey, ViPER_CLSID); err != nil {
		return fmt.Errorf("failed to set LFX CLSID: %w", err)
	}

	log.Printf("✓ APO attached to endpoint: %s", endpointID)
	return nil
}

// AttachToDefaultEndpoint simplifies the most common setup scenario
func (dm *DriverManager) AttachToDefaultEndpoint() error {
	guid, err := dm.GetDefaultEndpoint()
	if err != nil {
		return fmt.Errorf("cannot detect default endpoint: %w", err)
	}

	log.Printf("Detected default endpoint: %s", guid)
	return dm.AttachToEndpoint(guid)
}

// DetachFromEndpoint removes ViPER APO from a specific endpoint
func (dm *DriverManager) DetachFromEndpoint(endpointID string) error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	propPath := fmt.Sprintf(`%s\%s\FxProperties`, AudioRenderKey, endpointID)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, propPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("endpoint not found: %w", err)
	}
	defer k.Close()

	lfxKey := "{d04e05a6-594b-4fb6-a80d-01af5eedf162},0"
	if err := k.DeleteValue(lfxKey); err != nil && err != registry.ErrNotExist {
		return fmt.Errorf("failed to remove APO: %w", err)
	}

	log.Printf("✓ APO detached from endpoint: %s", endpointID)
	return nil
}

// RegisterAPO injects the APO CLSID into the Windows Registry.
// This allows the Audio Engine to "see" the ViPER processor.
func (dm *DriverManager) RegisterAPO(dllPath string) error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	system32Path := filepath.Join(os.Getenv("SystemRoot"), "System32", filepath.Base(dllPath))

	dllBytes, err := os.ReadFile(dllPath)
	if err != nil {
		return fmt.Errorf("failed to read DLL from %s: %w", dllPath, err)
	}
	if err := os.WriteFile(system32Path, dllBytes, 0644); err != nil {
		return fmt.Errorf("failed to copy DLL to System32: %w", err)
	}
	log.Printf("✓ DLL copied to System32: %s", system32Path)

	// APO registration
	apoPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create APO key: %w", err)
	}
	defer k.Close()

	k.SetStringValue("FriendlyName", "ViPER4Windows APO")
	k.SetStringValue("Copyright", "ViPER's Audio")
	k.SetStringValue("Library", system32Path)
	k.SetDWordValue("MajorVersion", 1)
	k.SetDWordValue("MinorVersion", 0)
	k.SetDWordValue("Flags", 0x0000000d)

	ik, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath+`\AudioInterface0`, registry.ALL_ACCESS)
	if err != nil {
		registry.DeleteKey(registry.LOCAL_MACHINE, apoPath)
		return fmt.Errorf("failed to create AudioInterface0: %w", err)
	}
	defer ik.Close()
	ik.SetStringValue("IID", ViPER_IID)

	// COM InprocServer32
	comPath := `SOFTWARE\Classes\CLSID\` + ViPER_CLSID + `\InprocServer32`
	ck, _, err := registry.CreateKey(registry.LOCAL_MACHINE, comPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("failed to create COM InprocServer32: %w", err)
	}
	defer ck.Close()
	ck.SetStringValue("", system32Path)
	ck.SetStringValue("ThreadingModel", "Both")

	log.Printf("✓ APO registered (CLSID: %s)", ViPER_CLSID)
	log.Printf("✓ Library: %s", system32Path)
	return nil
}

// UnregisterAPO removes all APO registry entries cleanly
func (dm *DriverManager) UnregisterAPO() error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	apoPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\`
	comPath := `SOFTWARE\Classes\CLSID\`

	paths := []string{
		apoPath + ViPER_CLSID + `\AudioInterface0`,
		apoPath + ViPER_CLSID,
		comPath + ViPER_CLSID + `\InprocServer32`,
		comPath + ViPER_CLSID,
	}

	for _, path := range paths {
		if err := registry.DeleteKey(registry.LOCAL_MACHINE, path); err != nil {
			log.Printf("⚠️ Could not delete %s: %v", path, err)
		}
	}

	system32Path := filepath.Join(os.Getenv("SystemRoot"), "System32", "Hydrogen_Inst.dll")
	if err := os.Remove(system32Path); err != nil && !os.IsNotExist(err) {
		log.Printf("⚠️ Could not remove DLL from System32: %v", err)
	}

	log.Printf("✓ APO unregistered (CLSID: %s)", ViPER_CLSID)
	return nil
}

// RestartAudioEngine forces Windows to reload APOs by bouncing Audiosrv.
func (dm *DriverManager) RestartAudioEngine() error {
	stopOrder := []string{"AudioEndpointBuilder", "AudioSrv"}
	startOrder := []string{"AudioSrv", "AudioEndpointBuilder"}

	// Stop services
	for _, svc := range stopOrder {
		exec.Command("sc", "stop", svc).Run()
	}

	// Wait until AudioSrv is fully stopped (max 6s)
	for i := 0; i < 12; i++ {
		time.Sleep(500 * time.Millisecond)
		out, _ := exec.Command("sc", "query", "AudioSrv").Output()
		if strings.Contains(string(out), "STOPPED") {
			break
		}
	}
	time.Sleep(500 * time.Millisecond)

	// Start services
	for _, svc := range startOrder {
		out, err := exec.Command("sc", "start", svc).CombinedOutput()
		output := strings.TrimSpace(string(out))
		if err != nil {
			// "already started" is not a real error
			if strings.Contains(output, "1056") || strings.Contains(output, "already") {
				log.Printf("✓ %s was already running", svc)
			} else {
				log.Printf("⚠️ %s start failed: %v", svc, err)
				return fmt.Errorf("%s start failed: %v", svc, err)
			}
		} else {
			log.Printf("✓ %s started", svc)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Verify services are running
	time.Sleep(1 * time.Second)
	for _, svc := range startOrder {
		out, err := exec.Command("sc", "query", svc).Output()
		if err != nil || !strings.Contains(string(out), "RUNNING") {
			log.Printf("⚠️ %s failed to start properly", svc)
			return fmt.Errorf("%s is not running after restart", svc)
		}
	}

	log.Printf("✓ Audio engine restart completed successfully")
	return nil
}

// CheckInstallation verifies if the APO is registered and the DLL exists.
func (dm *DriverManager) CheckInstallation() bool {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	// Additionally verify DLL path is valid
	if dllPath, _, err := k.GetStringValue("Library"); err == nil {
		if _, err := os.Stat(dllPath); err == nil {
			return true
		}
		log.Printf("⚠️ APO registered but DLL not found: %s", dllPath)
	}

	return false
}
