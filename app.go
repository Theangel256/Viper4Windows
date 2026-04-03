package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/options"
	"golang.org/x/sys/windows/registry"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed Hydrogen_Inst.dll
var driverBinary []byte

// ── DSP Data types ────────────────────────────────────────────────────────────

type MasterState struct {
	Power   bool    `json:"power"`
	PreVol  float64 `json:"preVol"`
	PostVol float64 `json:"postVol"`
}

type XBassState struct {
	On          bool    `json:"on"`
	SpeakerSize int     `json:"speakerSize"`
	Level       float64 `json:"level"`
	Mode        string  `json:"mode"` // "Natural Bass" | "Pure Bass"
}

type XClarityState struct {
	On    bool    `json:"on"`
	Level float64 `json:"level"`
	Mode  string  `json:"mode"` // "Natural" | "OZone+" | "X-HiFi"
}

type Surround3DState struct {
	On        bool   `json:"on"`
	SpaceSize int    `json:"spaceSize"`
	RoomSize  string `json:"roomSize"`
	ImageSize int    `json:"imageSize"`
}

type ReverbParams struct {
	On        bool    `json:"on"`
	RoomSize  float64 `json:"roomSize"`
	Damping   float64 `json:"damping"`
	Density   float64 `json:"density"`
	Bandwidth float64 `json:"bandwidth"`
	Decay     float64 `json:"decay"`
	PreDelay  float64 `json:"preDelay"`
	EarlyMix  float64 `json:"earlyMix"`
	WetMix    float64 `json:"wetMix"`
}

type ReverbPanelState struct {
	On       bool    `json:"on"`
	RoomSize string  `json:"roomSize"`
	Size     float64 `json:"size"`
	WetMix   float64 `json:"wetMix"`
}

// DSPState is the complete serializable app state.
type DSPState struct {
	Master      MasterState      `json:"master"`
	XBass       XBassState       `json:"xBass"`
	XClarity    XClarityState    `json:"xClarity"`
	Surround3D  Surround3DState  `json:"surround3D"`
	Reverb      ReverbParams     `json:"reverb"`
	ReverbPanel ReverbPanelState `json:"reverbPanel"`
	Mode        string           `json:"mode"` // "music" | "movie" | "freestyle"
	Equalizer   []float64        `json:"equalizer"`
}

// defaultState returns factory-reset values matching the original WinForms UI.
func defaultState() DSPState {
	return DSPState{
		Equalizer: make([]float64, 18),
		Mode:      "freestyle",
		Master: MasterState{
			Power:   true,
			PreVol:  0.0,
			PostVol: 12.00,
		},
		XBass: XBassState{
			On:          true,
			SpeakerSize: 5,
			Level:       0.0,
			Mode:        "Natural Bass",
		},
		XClarity: XClarityState{
			On:    true,
			Level: 0.0,
			Mode:  "X-HiFi",
		},
		Surround3D: Surround3DState{
			On:        true,
			SpaceSize: 5,
			RoomSize:  "Smallest Room",
			ImageSize: 2,
		},
		Reverb: ReverbParams{
			On:        true,
			RoomSize:  500,
			Damping:   1.03,
			Density:   12.2,
			Bandwidth: 44,
			Decay:     13,
			PreDelay:  0,
			EarlyMix:  91,
			WetMix:    50,
		},
		ReverbPanel: ReverbPanelState{
			On:       false,
			RoomSize: "Smallest Room",
			Size:     40,
			WetMix:   50,
		},
	}
}

// ── App Core ──────────────────────────────────────────────────────────────────

// App is the Wails application struct. All exported methods are automatically
// bound to the JS runtime as window.go.main.App.<MethodName>().
type App struct {
	ctx   context.Context
	state DSPState
}

// NewApp creates a new App instance, initializing the default DSP state.
func NewApp() *App {
	return &App{state: defaultState()}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ── Manejo de Instancia Única ────────────────────────────────────────────────

func (a *App) OnSecondInstanceLaunch(data options.SecondInstanceData) {
	// Traer la ventana al frente
	wailsRuntime.WindowUnminimise(a.ctx)
	wailsRuntime.WindowShow(a.ctx)

	// Opcional: Avisar al frontend que alguien intentó abrir otra instancia
	wailsRuntime.EventsEmit(a.ctx, "second_instance_attempt", data.Args)
}

// ── APO Driver Management ─────────────────────────────────────────────────────

func (a *App) CheckDriver() bool {
	apoKey := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{DA2FB532-3014-4B93-AD05-21B2C620F9C2}`
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, apoKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	k.Close()
	return true
}

// SetDriverStatus permite instalar o desinstalar el driver desde la App
func (a *App) SetDriverStatus(install bool) bool {
	if install {
		err := a.FixDriver() // Tu función que extrae la DLL y registra
		return err == nil
	} else {
		// Lógica para desinstalar: eliminar la clave del registro
		apoKey := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{DA2FB532-3014-4B93-AD05-21B2C620F9C2}`
		err := registry.DeleteKey(registry.LOCAL_MACHINE, apoKey)
		return err == nil
	}
}

// FixDriver handles extracting, registering, and patching the APO driver.
func (a *App) FixDriver() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	appDir := filepath.Dir(exePath)
	driverPath := filepath.Join(appDir, "Hydrogen_Inst.dll")

	log.Printf("Portable Mode: extracting driver to %s", driverPath)
	err = os.WriteFile(driverPath, driverBinary, 0644)
	if err != nil {
		return fmt.Errorf("failed to extract driver: %w", err)
	}

	// 1. Register APO Class with full local path
	err = a.registerAPO(driverPath)
	if err != nil {
		return fmt.Errorf("failed to register APO: %w", err)
	}

	// 2. Restart Audio Services
	return a.RestartAudioServices()
}

func (a *App) registerAPO(driverPath string) error {
	apoKey := `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{DA2FB532-3014-4B93-AD05-21B2C620F9C2}`
	k, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoKey, registry.WRITE)
	if err != nil {
		return err
	}
	defer k.Close()

	_ = k.SetStringValue("FriendlyName", "ViPER4Windows APO")
	_ = k.SetStringValue("Copyright", "ViPER's Audio")
	_ = k.SetDWordValue("MajorVersion", 1)
	_ = k.SetDWordValue("MinorVersion", 0)
	_ = k.SetDWordValue("Flags", 0x0000000d)
	_ = k.SetStringValue("Library", driverPath)

	// Create required interface subkey
	interfaceKey, _, err := registry.CreateKey(registry.LOCAL_MACHINE, apoKey+`\AudioInterface0`, registry.WRITE)
	if err == nil {
		_ = interfaceKey.SetStringValue("IID", "{FD7F2B29-24D0-4B5C-B177-592C39F9CA10}")
		interfaceKey.Close()
	}

	return nil
}

// RestartAudioServices restarts Audiosrv and AudioEndpointBuilder.
func (a *App) RestartAudioServices() error {
	log.Println("Restarting Audio Services...")
	_ = exec.Command("net", "stop", "Audiosrv", "/y").Run()
	_ = exec.Command("net", "stop", "AudioEndpointBuilder", "/y").Run()
	_ = exec.Command("net", "start", "AudioEndpointBuilder").Run()
	_ = exec.Command("net", "start", "Audiosrv").Run()
	return nil
}

// ToggleEnable toggles the ViperFX driver state in the Windows Registry
func (a *App) ToggleEnable(enabled bool) error {
	var k registry.Key
	var err error

	k, err = registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\ViPER4Windows`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		// Create if missing
		k, _, err = registry.CreateKey(registry.LOCAL_MACHINE, `SOFTWARE\ViPER4Windows`, registry.WRITE)
		if err != nil {
			return fmt.Errorf("failed to create registry (Admin required): %w", err)
		}
	}
	defer k.Close()

	val := uint32(0)
	if enabled {
		val = 1
	}

	return k.SetDWordValue("Enabled", val)
}

// GetDeviceStatus returns the current status
func (a *App) GetDeviceStatus() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows\CurrentVersion\AudioEngine\AudioProcessingObjects\{DA2FB532-3014-4B93-AD05-21B2C620F9C2}`, registry.QUERY_VALUE)
	if err != nil {
		return "Driver Missing", nil
	}
	k.Close()
	return "Registered", nil
}

// ── DSP State Methods ─────────────────────────────────────────────────────────

// GetState returns the full DSP state to the frontend.
func (a *App) GetState() DSPState {
	return a.state
}

// ResetState restores factory defaults.
func (a *App) ResetState() DSPState {
	a.state = defaultState()
	return a.state
}

// SetMode switches between music / movie / freestyle.
func (a *App) SetMode(mode string) {
	a.state.Mode = mode
}

// SetPower toggles the master power switch in the state.
func (a *App) SetPower(on bool) {
	a.state.Master.Power = on
}

// SetPreVolume sets the pre-DSP volume in dB (range −40..+12).
func (a *App) SetPreVolume(db float64) {
	a.state.Master.PreVol = clamp(db, -40, 12)
}

// SetPostVolume sets the post-DSP volume in dB (range −40..+12).
func (a *App) SetPostVolume(db float64) {
	a.state.Master.PostVol = clamp(db, -40, 12)
}

// SetXBass replaces the full XBass state.
func (a *App) SetXBass(s XBassState) {
	s.Level = clamp(s.Level, -12, 12)
	s.SpeakerSize = clampInt(s.SpeakerSize, 0, 10)
	a.state.XBass = s
}

// SetXClarity replaces the full XClarity state.
func (a *App) SetXClarity(s XClarityState) {
	s.Level = clamp(s.Level, -12, 12)
	a.state.XClarity = s
}

// SetSurround3D replaces the full 3D Surround state.
func (a *App) SetSurround3D(s Surround3DState) {
	s.SpaceSize = clampInt(s.SpaceSize, 0, 10)
	s.ImageSize = clampInt(s.ImageSize, 0, 10)
	a.state.Surround3D = s
}

// SetReverb replaces the full reverb parameter set.
func (a *App) SetReverb(p ReverbParams) {
	a.state.Reverb = p
}

// SetReverbPanel updates the bottom reverb strip state.
func (a *App) SetReverbPanel(p ReverbPanelState) {
	a.state.ReverbPanel = p
}

// SetEqBand actualiza una banda específica
func (a *App) SetEqBand(index int, db float64) {
	// Validamos que el índice exista en nuestro array de 18
	if index >= 0 && index < len(a.state.Equalizer) {
		// Limitamos el rango de -12dB a +12dB
		a.state.Equalizer[index] = clamp(db, -12, 12)
		// Aquí llamarías a tu motor de audio nativo para aplicar el cambio en tiempo real

		// LOG para debug:
		// fmt.Printf("Band %d set to %.2f dB\n", index, db)
	}
}

// ResetEq pone todas las bandas a 0
func (a *App) ResetEq() {
	for i := range a.state.Equalizer {
		a.state.Equalizer[i] = 0
	}
}

// ── Presets Management ────────────────────────────────────────────────────────

// presetsDir returns the OS-specific presets directory, creating it if needed.
func presetsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "LumaFX", "presets")
	return dir, os.MkdirAll(dir, 0755)
}

// SavePreset persists the current DSP state to a named JSON file.
func (a *App) SavePreset(name string) error {
	dir, err := presetsDir()
	if err != nil {
		return fmt.Errorf("presets dir: %w", err)
	}
	data, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	return os.WriteFile(path, data, 0644)
}

// LoadPreset reads a preset by name and returns the full DSP state.
func (a *App) LoadPreset(name string) (DSPState, error) {
	dir, err := presetsDir()
	if err != nil {
		return a.state, fmt.Errorf("presets dir: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return a.state, fmt.Errorf("read preset: %w", err)
	}
	var s DSPState
	if err := json.Unmarshal(data, &s); err != nil {
		return a.state, fmt.Errorf("unmarshal: %w", err)
	}
	a.state = s
	return a.state, nil
}

// ListPresets returns the names of all saved presets.
func (a *App) ListPresets() ([]string, error) {
	dir, err := presetsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name()[:len(e.Name())-5])
		}
	}
	return names, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
