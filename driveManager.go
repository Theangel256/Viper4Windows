// audio_devices.go
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

// ── Audio Device Types ─────────────────────────────────────────────────────────

type AudioDevice struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"` // "render" | "capture"
	HasAPO  bool   `json:"hasAPO"`
	State   int    `json:"state"`
	Default bool   `json:"default"`
}

const (
	// APO Class ID y Interface ID (exactos del ViPERDSP)
	ViPER_CLSID = "{DA2FB532-3014-4B93-AD05-21B2C620F9C2}"
	ViPER_IID   = "{FD7F2B29-24D0-4B5C-B177-592C39F9CA10}"

	// Registry paths
	OutputBasePath = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render`
	InputBasePath  = `SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Capture`

	// Property keys (Windows 10/11)
	pkeyFriendlyName = "{a45c254e-df1c-4efd-8020-67d146a850e0},2"
	pkeyFXName       = "{b725f130-47ef-101a-a5f1-02608c9eebac},10"
	pkeyFXPreMix     = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},5" // SFX
	pkeyFXPostMix    = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},6" // MFX
	pkeyFXEndpoint   = "{d04e05a6-594b-4fb6-a80d-01af5eed7d1d},7" // EFX

	DeviceStateActive = 1
)

// ── DriverManager ─────────────────────

type DriverManager struct{}

// InstallAPOOnDevice (ahora usa DriverManager internamente)
func (a *App) InstallAPOOnDevice(deviceID, deviceType string) error {
	dm := &DriverManager{}
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	if err := dm.AttachToEndpoint(deviceID, deviceType); err != nil {
		return err
	}

	log.Printf("✓ APO instalado en %s (%s)", deviceID, deviceType)
	return dm.RestartAudioEngine()
}

// UninstallAPOFromDevice
func (a *App) UninstallAPOFromDevice(deviceID, deviceType string) error {
	dm := &DriverManager{}
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	if err := dm.DetachFromEndpoint(deviceID, deviceType); err != nil {
		return err
	}

	log.Printf("✓ APO removido de %s (%s)", deviceID, deviceType)

	return dm.RestartAudioEngine()
}

// InstallAPOOnAllRender instala en todos los dispositivos de reproducción activos
func (a *App) InstallAPOOnAllRender() error {
	devices, err := enumerateDevices("render")
	if err != nil {
		return err
	}

	var lastErr error
	dm := &DriverManager{}
	for _, d := range devices {
		if d.State == DeviceStateActive {
			if err := dm.AttachToEndpoint(d.ID, "render"); err != nil {
				log.Printf("⚠️ Error en %s: %v", d.Name, err)
				lastErr = err
			}
		}
	}
	if lastErr == nil {
		return dm.RestartAudioEngine()
	}
	return lastErr
}

// ── Métodos internos de DriverManager ───────────────────────────────────────

func (dm *DriverManager) AttachToEndpoint(endpointID, deviceType string) error {
	basePath := resolveBasePath(deviceType)

	fxPath := basePath + `\` + endpointID + `\FxProperties`
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
	return nil
}
func (dm *DriverManager) GetDefaultEndpoint() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Multimedia\Sound Mapper`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		if playback, _, err := k.GetStringValue("Playback"); err == nil && playback != "" {
			return playback, nil
		}
	}

	// fallback
	devices, _ := enumerateDevices("render")
	if len(devices) > 0 {
		return devices[0].ID, nil
	}
	return "", fmt.Errorf("no audio endpoint found")
}
func (dm *DriverManager) AttachToDefaultEndpoint() error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}
	guid, err := dm.GetDefaultEndpoint()
	if err != nil {
		return err
	}
	log.Printf("Dispositivo por defecto: %s", guid)
	return dm.AttachToEndpoint(guid, "render")
}
func (dm *DriverManager) DetachFromEndpoint(endpointID, deviceType string) error {
	basePath := OutputBasePath
	if deviceType == "capture" {
		basePath = InputBasePath
	}

	fxPath := basePath + `\` + endpointID + `\FxProperties`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, fxPath, registry.SET_VALUE)
	if err != nil {
		return nil // ya no estaba
	}
	defer k.Close()

	_ = k.DeleteValue(pkeyFXPreMix)
	_ = k.DeleteValue(pkeyFXPostMix)
	_ = k.DeleteValue(pkeyFXEndpoint)
	return nil
}

// RegisterAPO, UnregisterAPO, CheckInstallation, RestartAudioEngine (del driver_control.go)

// RegisterAPO registra el APO en el sistema (copia DLL + claves de registro)
func (dm *DriverManager) RegisterAPO(dllPath string) error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	// Copiar DLL a System32
	system32Path := filepath.Join(os.Getenv("SystemRoot"), "System32", filepath.Base(dllPath))
	dllBytes, err := os.ReadFile(dllPath)
	if err != nil {
		return fmt.Errorf("no se pudo leer %s: %w", dllPath, err)
	}
	if err := os.WriteFile(system32Path, dllBytes, 0644); err != nil {
		return fmt.Errorf("no se pudo copiar DLL a System32: %w", err)
	}
	log.Printf("✓ DLL copiado a System32: %s", system32Path)

	// Registro APO
	apoPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("no se pudo crear clave APO: %w", err)
	}
	defer k.Close()

	k.SetStringValue("FriendlyName", "ViPER4Windows APO")
	k.SetStringValue("Copyright", "ViPER's Audio")
	k.SetStringValue("Library", system32Path)
	k.SetDWordValue("MajorVersion", 1)
	k.SetDWordValue("MinorVersion", 0)
	k.SetDWordValue("Flags", 0x0000000d)

	// AudioInterface0
	ik, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoPath+`\AudioInterface0`, registry.ALL_ACCESS)
	if err != nil {
		ik.SetStringValue("IID", ViPER_IID)
		ik.Close()
	}

	// InprocServer32
	comPath := `SOFTWARE\Classes\CLSID\` + ViPER_CLSID + `\InprocServer32`
	ck, _, err := registry.CreateKey(registry.LOCAL_MACHINE, comPath, registry.ALL_ACCESS)
	if err != nil {
		ck.SetStringValue("", system32Path)
		ck.SetStringValue("ThreadingModel", "Both")
		ck.Close()
	}

	log.Printf("✓ APO registrado correctamente (CLSID: %s)", ViPER_CLSID)
	return nil
}

// UnregisterAPO elimina completamente el APO del registro y borra la DLL
func (dm *DriverManager) UnregisterAPO() error {
	if err := dm.RequireAdmin(); err != nil {
		return err
	}

	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID + `\AudioInterface0`,
		`SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID,
		`SOFTWARE\Classes\CLSID\` + ViPER_CLSID + `\InprocServer32`,
		`SOFTWARE\Classes\CLSID\` + ViPER_CLSID,
	}

	for _, p := range paths {
		registry.DeleteKey(registry.LOCAL_MACHINE, p)
	}

	dllPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "Hydrogen_Inst.dll")
	_ = os.Remove(dllPath)

	log.Printf("✓ APO desregistrado completamente")
	return nil
}

// RestartAudioEngine reinicia el motor de audio para aplicar cambios
// Sugerencia: Solo llama a esto cuando sea estrictamente necesario (Instalación/Desinstalación)
func (dm *DriverManager) RestartAudioEngine() error {

	log.Println("Reconectando motor de audio...")

	exec.Command("net", "stop", "AudioEndpointBuilder", "/y").Run()

	time.Sleep(1 * time.Second)

	// Reiniciar en orden
	services := []string{"AudioEndpointBuilder", "AudioSrv"}
	for _, svc := range services {
		cmd := exec.Command("net", "start", svc)
		if err := cmd.Run(); err != nil {
			// Ignorar error si ya está corriendo
			log.Printf("Aviso: %s pudo no haber arrancado: %v", svc, err)
		}
	}

	log.Printf("✓ Motor de audio refrescado")
	return nil
}

// CheckInstallation verifica si el APO está correctamente registrado
func (dm *DriverManager) CheckInstallation() bool {
	keyPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\` + ViPER_CLSID
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	if dllPath, _, err := k.GetStringValue("Library"); err == nil {
		if _, err := os.Stat(dllPath); err == nil {
			return true
		}
	}
	return false
}

// ── Helpers internos (mantenidos y optimizados) ─────────────────────────────

func resolveBasePath(deviceType string) string {
	switch deviceType {
	case "render":
		return OutputBasePath
	case "capture":
		return InputBasePath
	}
	return ""
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
		ID:      guid,
		Type:    deviceType,
		Default: false,
		Name:    guid, // fallback if friendly name not found
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
			dev.HasAPO = strings.EqualFold(val, ViPER_CLSID)
		}
	}

	return dev, nil
}
