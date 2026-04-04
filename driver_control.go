package main

import (
	"fmt"
	"log"
	"os/exec"

	"golang.org/x/sys/windows/registry"
)

const (
	// APO Class ID for ViPER4Windows
	ViPER_CLSID = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}"
	// Interface ID for the APO
	ViPER_IID = "{FD7F2B29-24D0-4B5C-B177-592C39F9CA10}"

	// Registry paths for Windows Audio Endpoints
	AudioRenderKey = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render`
)

// DriverManager handles the registration and lifecycle of the APO.
// Corresponds to the 'Installer' logic in ViPERDSP/Alpha.
type DriverManager struct{}

// RegisterAPO injects the APO CLSID into the Windows Registry.
// This allows the Audio Engine to "see" the ViPER processor.
func (dm *DriverManager) RegisterAPO(dllPath string) error {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, keyPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("registry access denied (run as admin): %w", err)
	}
	defer k.Close()

	_ = k.SetStringValue("Library", dllPath)
	_ = k.SetStringValue("FriendlyName", "ViPER4Windows APO")
	_ = k.SetDWordValue("Flags", 0x0000000d) // APO_FLAG_INPLACE | APO_FLAG_SAMPLESPERFRAME_MUST_MATCH

	// Register Interface
	ik, _, _ := registry.CreateKey(registry.LOCAL_MACHINE, keyPath+`\AudioInterface0`, registry.ALL_ACCESS)
	defer ik.Close()
	_ = ik.SetStringValue("IID", ViPER_IID)

	return nil
}

// AttachToEndpoint binds the APO to a specific Audio Device (GUID).
// In ViPERFX_RE, this involves modifying the Multi-Endpoint (MME) properties.
func (dm *DriverManager) AttachToEndpoint(endpointID string) error {
	// Path: HKLM\...\MMDevices\Audio\Render\{GUID}\FxProperties
	propPath := fmt.Sprintf(`%s\%s\FxProperties`, AudioRenderKey, endpointID)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, propPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	// These GUIDs represent the 'Pre-Mix' and 'Post-Mix' effect slots in Windows
	// {d04e05a6-594b-4fb6-a80d-01af5eedf162},0 = LFX (Local Effect)
	// {d04e05a6-594b-4fb6-a80d-01af5eedf162},1 = GFX (Global Effect)
	lfxKey := "{d04e05a6-594b-4fb6-a80d-01af5eedf162},0"
	return k.SetStringValue(lfxKey, ViPER_CLSID)
}

// RestartAudioEngine forces Windows to reload APOs by bouncing Audiosrv.
func (dm *DriverManager) RestartAudioEngine() error {
	// 'net stop' is used because it handles dependency service AudioEndpointBuilder
	log.Println("Restarting Audio Services to apply driver changes...")
	_ = exec.Command("net", "stop", "Audiosrv", "/y").Run()
	err := exec.Command("net", "start", "Audiosrv").Run()
	if err != nil {
		return fmt.Errorf("failed to start Audiosrv: %w", err)
	}
	return exec.Command("net", "start", "AudioEndpointBuilder").Run()
}

// UnregisterAPO removes the APO CLSID from the Windows Registry.
func (dm *DriverManager) UnregisterAPO() error {
	log.Println("Unregistering APO...")
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID

	// It's good practice to delete subkeys before the parent key
	_ = registry.DeleteKey(registry.LOCAL_MACHINE, keyPath+`\AudioInterface0`)
	err := registry.DeleteKey(registry.LOCAL_MACHINE, keyPath)
	if err != nil {
		return fmt.Errorf("failed to unregister APO: %w", err)
	}
	return nil
}

// CheckInstallation verifies if the APO is registered and the DLL exists.
func (dm *DriverManager) CheckInstallation() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\`+ViPER_CLSID, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	return true
}
