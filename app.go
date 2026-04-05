package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/options"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed Hydrogen_Inst.dll
var driverBinary []byte

// ── DSP Parameter Constants ──────────────────────────────────────────────────

const (
	MinPreVol        = -12.0
	MaxPreVol        = 0
	MinPostVol       = 0
	MaxPostVol       = 12.0
	MinEqBand        = -12.0
	MaxEqBand        = 12.0
	MinXBassLevel    = -12.0
	MaxXBassLevel    = 12.0
	MinXClarityLevel = -12.0
	MaxXClarityLevel = 12.0
	MinSpeakerSize   = 0
	MaxSpeakerSize   = 10
	MinSpaceSize     = 0
	MaxSpaceSize     = 10
	MinImageSize     = 0
	MaxImageSize     = 10
)

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
	dsp   *DSPManager    // Maneja la memoria compartida (Hot Path)
	drv   *DriverManager // Maneja el registro y servicios (Control Path)
}

// NewApp creates a new App instance, initializing the default DSP state.
func NewApp() *App {
	return &App{
		state: defaultState(),
		dsp:   &DSPManager{},
		drv:   &DriverManager{},
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.dsp.InitSharedMemory(); err != nil {
		log.Printf("⚠️ No se pudo conectar con el Driver (SharedMem): %v", err)
	}
	a.synchronize() // Carga inicial de parámetros

	log.Printf("🤖 Backend iniciado. Estado del Driver: %v", a.CheckDriver())
}

// synchronize envía el estado actual de Go al driver de Windows
func (a *App) synchronize() {
	a.dsp.ApplyChanges(a.state)
}

// ── Manejo de Instancia Única ────────────────────────────────────────────────

func (a *App) OnSecondInstanceLaunch(data options.SecondInstanceData) {
	// Traer la ventana al frente
	wailsRuntime.WindowUnminimise(a.ctx)
	wailsRuntime.WindowShow(a.ctx)

	// Opcional: Avisar al frontend que alguien intentó abrir otra instancia
	wailsRuntime.EventsEmit(a.ctx, "second_instance_attempt", data.Args)
}

// IsElevated checks if the process has administrative privileges
func (dm *DriverManager) IsElevated() bool {
	// Method 1: Token elevation (primary)
	var token windows.Token
	if err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token); err == nil {
		defer token.Close()

		var elevation TOKEN_ELEVATION
		var returnedLen uint32
		err = windows.GetTokenInformation(
			token,
			windows.TokenElevation,
			(*byte)(unsafe.Pointer(&elevation)),
			uint32(unsafe.Sizeof(elevation)),
			&returnedLen,
		)
		if err == nil {
			return elevation.TokenIsElevated != 0
		}
	}

	// Method 2: Fallback — try opening a protected registry key
	// If we can write to HKLM, we're elevated
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.SET_VALUE,
	)
	if err == nil {
		k.Close()
		return true
	}

	return false
}

// RequireAdmin validates administrative privileges before operations
func (dm *DriverManager) RequireAdmin() error {
	if !dm.IsElevated() {
		return fmt.Errorf("ACCESS_DENIED: Administrator privileges required.\nRight-click the application and select 'Run as Administrator'")
	}
	return nil
}

// ── APO Driver Management ─────────────────────────────────────────────────────

// CheckDriver verifica si el registro existe Y si el archivo DLL físico está presente
// Ahora delega la verificación al DriverManager.
func (a *App) CheckDriver() bool {
	return a.drv.CheckInstallation()
}

// SetDriverStatus permite instalar o desinstalar el driver desde la App
func (a *App) SetDriverStatus(install bool) bool {
	if install {
		log.Println("Iniciando instalación del driver APO...")
		exePath, err := os.Executable()
		if err != nil {
			log.Printf("failed to get executable path: %v", err)
			return false
		}
		appDir := filepath.Dir(exePath)
		driverPath := filepath.Join(appDir, "Hydrogen_Inst.dll")

		log.Printf("Portable Mode: extracting driver to %s", driverPath)
		if err := os.WriteFile(driverPath, driverBinary, 0644); err != nil {
			log.Printf("failed to extract driver: %v", err)
			return false
		}
		system32Path := filepath.Join(os.Getenv("SystemRoot"), "System32", "Hydrogen_Inst.dll")
		if err := os.WriteFile(system32Path, driverBinary, 0644); err != nil {
			log.Printf("⚠️ Could not copy to System32: %v", err)
		}
		if err := a.drv.RegisterAPO(driverPath); err != nil {
			log.Printf("failed to register APO: %v", err)
			return false
		}

		if err := a.drv.RestartAudioEngine(); err != nil {
			log.Printf("failed to restart audio engine: %v", err)
			return false
		}
		log.Println("Inicializando memoria compartida post-instalación...")
		if err := a.dsp.InitSharedMemory(); err != nil {
			log.Printf("⚠️ Error init shared mem after install: %v", err)
		} else {
			a.synchronize()
		}
		return true

	} else {
		log.Println("Desinstalando driver APO...")
		return a.drv.UnregisterAPO() == nil && a.drv.RestartAudioEngine() == nil
	}
}

// FixDriver ha sido eliminada ya que su lógica se ha integrado en SetDriverStatus
// y delegada al DriverManager.

// registerAPO ha sido eliminada ya que su lógica se ha delegado al DriverManager.

// RestartAudioServices ha sido eliminada ya que su lógica se ha delegado al DriverManager.

func (a *App) UnregisterAPO() error {
	return a.drv.UnregisterAPO()
}

func (a *App) PatchDefaultEndpoint() error {
	// Esta ruta varía según el ID de tu tarjeta de sonido (GUID del Endpoint)
	// Para hacerlo automático, habría que iterar sobre:
	// HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\MMDevices\Audio\Render

	log.Println("Buscando dispositivos de audio para parchear...")
	// Por ahora, lo más seguro es usar 'Equalizer APO' o el configurador original
	// para marcar el check del dispositivo una sola vez.
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

// GetDeviceStatus returns the current status (Depende de CheckDriver ahora para más robustez)
func (a *App) GetDeviceStatus() (string, error) {
	if a.CheckDriver() {
		// Ahora CheckDriver usa a.drv.CheckInstallation()
		return "Registered", nil
	}
	return "Driver Missing", nil
}

// ── DSP State Methods ─────────────────────────────────────────────────────────

// GetState returns the full DSP state to the frontend.
func (a *App) GetState() DSPState {
	return a.state
}

// ResetState restores factory defaults.
func (a *App) ResetState() DSPState {
	a.state = defaultState()
	a.synchronize()
	return a.state
}

// SetMode switches between music / movie / freestyle.
func (a *App) SetMode(mode string) {
	a.state.Mode = mode
	a.synchronize()
}

// SetPower toggles the master power switch in the state.
func (a *App) SetPower(on bool) {
	a.state.Master.Power = on
	a.synchronize()
}

// SetPreVolume sets the pre-DSP volume in dB (range −40..+12).
func (a *App) SetPreVolume(db float64) {
	a.state.Master.PreVol = clamp(db, MinPreVol, MaxPreVol)
	a.synchronize()
}

// SetPostVolume sets the post-DSP volume in dB (range −40..+12).
func (a *App) SetPostVolume(db float64) {
	a.state.Master.PostVol = clamp(db, MinPostVol, MaxPostVol)
	a.synchronize()
}

// SetXBass replaces the full XBass state.
func (a *App) SetXBass(s XBassState) {
	s.Level = clamp(s.Level, MinXBassLevel, MaxXBassLevel)
	s.SpeakerSize = clampInt(s.SpeakerSize, MinSpeakerSize, MaxSpeakerSize)
	a.state.XBass = s
	a.synchronize()
}

// SetXClarity replaces the full XClarity state.
func (a *App) SetXClarity(s XClarityState) {
	s.Level = clamp(s.Level, MinXClarityLevel, MaxXClarityLevel)
	a.state.XClarity = s
	a.synchronize()
}

// SetSurround3D replaces the full 3D Surround state.
func (a *App) SetSurround3D(s Surround3DState) {
	s.SpaceSize = clampInt(s.SpaceSize, MinSpaceSize, MaxSpaceSize)
	s.ImageSize = clampInt(s.ImageSize, MinImageSize, MaxImageSize)
	a.state.Surround3D = s
	a.synchronize()
}

// SetReverb replaces the full reverb parameter set.
func (a *App) SetReverb(p ReverbParams) {
	a.state.Reverb = p
	a.synchronize()
}

// SetReverbPanel updates the bottom reverb strip state.
func (a *App) SetReverbPanel(p ReverbPanelState) {
	a.state.ReverbPanel = p
	a.synchronize()
}

func (a *App) SetEqBand(index int, db float64) {
	if index >= 0 && index < len(a.state.Equalizer) {
		a.state.Equalizer[index] = clamp(db, MinEqBand, MaxEqBand)

		params := VIPER_DSP_PARAMS{
			Enabled:   1.0,
			PreVol:    float32(a.state.Master.PreVol),
			PostVol:   float32(a.state.Master.PostVol),
			EqEnabled: 1.0,
		}

		for i := 0; i < 18; i++ {
			params.EqBands[i] = float32(a.state.Equalizer[i])
		}

		// Pasar directamente como *VIPER_DSP_PARAMS, sin conversión a []byte
		if err := a.writeToSharedMemory(&params); err != nil {
			fmt.Printf("❌ ERROR APO: %v\n", err)
		} else {
			fmt.Printf("✅ Banda %d ajustada a %.2f dB\n", index, db)
		}
	}
}

// ResetEq pone todas las bandas a 0
func (a *App) ResetEq() {
	for i := range a.state.Equalizer {
		a.state.Equalizer[i] = 0
	}
	a.synchronize()
}

// SetFullEq sets all equalizer bands at once.
func (a *App) SetFullEq(bands []float64) {
	if len(bands) == len(a.state.Equalizer) {
		for i, db := range bands {
			a.state.Equalizer[i] = clamp(db, MinEqBand, MaxEqBand)
		}
		a.synchronize()
	}
}

func (a *App) writeToSharedMemory(params *VIPER_DSP_PARAMS) error {
	return a.dsp.WriteParams(params)
}

// ── Presets Management ────────────────────────────────────────────────────────

// presetsDir returns the OS-specific presets directory, creating it if needed.
func getPresetsDir() string {
	// 1. Obtenemos la ruta completa del ejecutable (.exe)
	exePath, err := os.Executable()
	if err != nil {
		log.Println("Error obteniendo la ruta del exe:", err)
		return "presets" // Fallback a relativo si falla
	}

	// 2. Obtenemos solo el directorio (quitamos el nombre del archivo)
	exeDir := filepath.Dir(exePath)

	// 3. Creamos la ruta final uniendo el directorio con la carpeta 'presets'
	finalPath := filepath.Join(exeDir, "presets")

	// 4. Nos aseguramos de que la carpeta exista
	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		_ = os.MkdirAll(finalPath, 0755)
	}
	return finalPath
}

// SavePreset persists the current DSP state to a named JSON file.
func (a *App) SavePreset(name string) error {
	dir := getPresetsDir()
	data, err := json.MarshalIndent(a.state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	return os.WriteFile(path, data, 0644)
}

// LoadPreset reads a preset by name and returns the full DSP state.
func (a *App) LoadPreset(name string) (DSPState, error) {
	dir := getPresetsDir()
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
	dir := getPresetsDir()
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
