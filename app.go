package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

type dllResolution struct {
	path        string
	emptyPaths  []string
	searchPaths []string
}

type toastMessage struct {
	Message string `json:"message"`
	Tone    string `json:"tone"`
}

func logV(format string, args ...any) {
	log.Printf("[ViPER] "+format, args...)
}

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

type XBassMonoState struct {
	On          bool    `json:"on"`
	SpeakerSize int     `json:"speakerSize"`
	Level       float64 `json:"level"`
	Mode        string  `json:"mode"`
}

type ConvolverState struct {
	On           bool    `json:"on"`
	KernelPath   string  `json:"kernelPath"`
	CrossChannel float64 `json:"crossChannel"`
}

type DDCState struct {
	On          bool      `json:"on"`
	Coeffs44100 []float64 `json:"coeffs44100"`
	Coeffs48000 []float64 `json:"coeffs48000"`
}

type AGCState struct {
	On        bool    `json:"on"`
	Ratio     float64 `json:"ratio"`
	Volume    float64 `json:"volume"`
	MaxScaler float64 `json:"maxScaler"`
}

type DynamicSystemState struct {
	On          bool    `json:"on"`
	XCoeffsLow  int     `json:"xCoeffsLow"`
	XCoeffsHigh int     `json:"xCoeffsHigh"`
	YCoeffsLow  int     `json:"yCoeffsLow"`
	YCoeffsHigh int     `json:"yCoeffsHigh"`
	SideGainX   float64 `json:"sideGainX"`
	SideGainY   float64 `json:"sideGainY"`
	Strength    float64 `json:"strength"`
}

type SpectrumExtensionState struct {
	On                 bool    `json:"on"`
	ReferenceFrequency int     `json:"referenceFrequency"`
	Exciter            float64 `json:"exciter"`
}

type FieldSurroundState struct {
	On       bool    `json:"on"`
	Widening float64 `json:"widening"`
	MidImage float64 `json:"midImage"`
	Depth    int     `json:"depth"`
}

type DiffSurroundState struct {
	On    bool    `json:"on"`
	Delay float64 `json:"delay"`
}

type CureState struct {
	On             bool `json:"on"`
	StrengthPreset int  `json:"strengthPreset"`
}

type TubeSimulatorState struct {
	On bool `json:"on"`
}

type AnalogXState struct {
	On   bool `json:"on"`
	Mode int  `json:"mode"`
}

type OutputState struct {
	Pan     float64 `json:"pan"`
	Limiter float64 `json:"limiter"`
}

type FETCompressorState struct {
	On          bool    `json:"on"`
	Threshold   float64 `json:"threshold"`
	Ratio       float64 `json:"ratio"`
	Knee        float64 `json:"knee"`
	AutoKnee    bool    `json:"autoKnee"`
	Gain        float64 `json:"gain"`
	AutoGain    bool    `json:"autoGain"`
	Attack      float64 `json:"attack"`
	AutoAttack  bool    `json:"autoAttack"`
	Release     float64 `json:"release"`
	AutoRelease bool    `json:"autoRelease"`
	KneeMulti   float64 `json:"kneeMulti"`
	MaxAttack   float64 `json:"maxAttack"`
	MaxRelease  float64 `json:"maxRelease"`
	Crest       float64 `json:"crest"`
	Adapt       float64 `json:"adapt"`
	NoClip      bool    `json:"noClip"`
}

type SpeakerCorrectionState struct {
	On bool `json:"on"`
}

// DSPState is the complete serializable app state.
type DSPState struct {
	Master            MasterState            `json:"master"`
	Output            OutputState            `json:"output"`
	XBass             XBassState             `json:"xBass"`
	XBassMono         XBassMonoState         `json:"xBassMono"`
	XClarity          XClarityState          `json:"xClarity"`
	Surround3D        Surround3DState        `json:"surround3D"`
	Reverb            ReverbParams           `json:"reverb"`
	ReverbPanel       ReverbPanelState       `json:"reverbPanel"`
	Convolver         ConvolverState         `json:"convolver"`
	DDC               DDCState               `json:"ddc"`
	AGC               AGCState               `json:"agc"`
	DynamicSystem     DynamicSystemState     `json:"dynamicSystem"`
	SpectrumExtension SpectrumExtensionState `json:"spectrumExtension"`
	FieldSurround     FieldSurroundState     `json:"fieldSurround"`
	DiffSurround      DiffSurroundState      `json:"diffSurround"`
	Cure              CureState              `json:"cure"`
	TubeSimulator     TubeSimulatorState     `json:"tubeSimulator"`
	AnalogX           AnalogXState           `json:"analogX"`
	FETCompressor     FETCompressorState     `json:"fetCompressor"`
	SpeakerCorrection SpeakerCorrectionState `json:"speakerCorrection"`
	Mode              string                 `json:"mode"` // "music" | "movie" | "freestyle"
	EqOn              bool                   `json:"eqOn"`
	Equalizer         []float64              `json:"equalizer"`
}

// defaultState returns factory-reset values matching the original WinForms UI.
func defaultState() DSPState {
	return DSPState{
		Equalizer: make([]float64, 18),
		EqOn:      true,
		Mode:      "freestyle",
		Master: MasterState{
			Power:   true,
			PreVol:  0.0,
			PostVol: 12.00,
		},
		Output: OutputState{
			Pan:     0.0,
			Limiter: 1.0,
		},
		XBass: XBassState{
			On:          true,
			SpeakerSize: 5,
			Level:       0.0,
			Mode:        "Natural Bass",
		},
		XBassMono: XBassMonoState{
			On:          false,
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
		Convolver: ConvolverState{
			On:           false,
			KernelPath:   "",
			CrossChannel: 0,
		},
		DDC: DDCState{
			On:          false,
			Coeffs44100: []float64{},
			Coeffs48000: []float64{},
		},
		AGC: AGCState{
			On:        false,
			Ratio:     1.0,
			Volume:    1.0,
			MaxScaler: 1.0,
		},
		DynamicSystem: DynamicSystemState{
			On:          false,
			XCoeffsLow:  120,
			XCoeffsHigh: 120,
			YCoeffsLow:  200,
			YCoeffsHigh: 200,
			SideGainX:   0.0,
			SideGainY:   0.0,
			Strength:    0.0,
		},
		SpectrumExtension: SpectrumExtensionState{
			On:                 false,
			ReferenceFrequency: 7600,
			Exciter:            0.0,
		},
		FieldSurround: FieldSurroundState{
			On:       false,
			Widening: 0.0,
			MidImage: 0.0,
			Depth:    0,
		},
		DiffSurround: DiffSurroundState{
			On:    false,
			Delay: 0.0,
		},
		Cure: CureState{
			On:             false,
			StrengthPreset: 0,
		},
		TubeSimulator: TubeSimulatorState{
			On: false,
		},
		AnalogX: AnalogXState{
			On:   false,
			Mode: 0,
		},
		FETCompressor: FETCompressorState{
			On:          false,
			Threshold:   0.0,
			Ratio:       1.0,
			Knee:        0.0,
			AutoKnee:    false,
			Gain:        0.0,
			AutoGain:    false,
			Attack:      0.0,
			AutoAttack:  false,
			Release:     0.0,
			AutoRelease: false,
			KneeMulti:   1.0,
			MaxAttack:   0.0,
			MaxRelease:  0.0,
			Crest:       0.0,
			Adapt:       0.0,
			NoClip:      false,
		},
		SpeakerCorrection: SpeakerCorrectionState{
			On: false,
		},
	}
}

// ── App Core ──────────────────────────────────────────────────────────────────

// App is the Wails application struct. All exported methods are automatically
// bound to the JS runtime as window.go.main.App.<MethodName>().
type App struct {
	ctx   context.Context
	mu    sync.Mutex
	state DSPState
	dsp   *DSPManager    // Maneja la memoria compartida (Hot Path)
	drv   *DriverManager // Maneja el registro y servicios (Control Path)

	lastEqUpdate time.Time
	lastDSPToast time.Time
}

// NewApp creates a new App instance, initializing the default DSP state.
func NewApp() *App {
	return &App{
		state: defaultState(),
		dsp:   &DSPManager{},
		drv:   &DriverManager{},
	}
}

// startup is called when the Wails app starts.
// Order:
//  1. Try to load ViPERDSP.dll for the direct C++ path.
//  2. Try to open shared memory for the APO path.
//  3. Push the initial state to whichever paths are available.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// ── 1. Direct DLL path ───────────────────────────────────────────────────
	directReady := false
	exePath, err := os.Executable()
	if err == nil {
		dllPath := filepath.Join(filepath.Dir(exePath), "ViPERDSP.dll")
		if err := a.dsp.LoadDLLPath(dllPath); err != nil {
			logV("dll unavailable: %v", err)
		} else {
			directReady = true
		}
	}

	// ── 2. Shared memory path (APO/Hydrogen fallback) ────────────────────────
	if err := a.dsp.SyncSharedMemory(); err != nil {
		logV("shared memory unavailable: %v", err)
	}

	// ── 3. Push initial state ────────────────────────────────────────────────
	a.synchronize()

	migratePresets()

	logV("ready driver=%t direct=%t", a.CheckDriver(), directReady)
}

// shutdown is called when the Wails app is closing.
// Releases DLL and shared memory handles cleanly.
func (a *App) shutdown(ctx context.Context) {
	logV("shutdown")
	a.dsp.Close()
}

// synchronize aplica el estado actual de Go al motor de audio disponible
// (directo DLL o memoria compartida) y reporta los errores como toast.
func (a *App) synchronize() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.dsp.ApplyChanges(a.state); err != nil {
		a.reportDSPError(err)
	}
}

// setState actualiza el estado bajo el mutex y lo sincroniza con el DSP,
// devolviendo una copia profunda del nuevo estado para la UI.
func (a *App) setState(mutate func(*DSPState)) DSPState {
	a.mu.Lock()
	defer a.mu.Unlock()
	mutate(&a.state)
	if err := a.dsp.ApplyChanges(a.state); err != nil {
		a.reportDSPError(err)
	}
	return cloneState(a.state)
}

// reportDSPError registra el error y muestra un toast, como máximo una vez
// cada 5 segundos, para no inundar la UI durante el arrastre de sliders.
func (a *App) reportDSPError(err error) {
	if err == nil {
		return
	}
	logV("dsp apply failed: %v", err)
	if time.Since(a.lastDSPToast) < 5*time.Second {
		return
	}
	a.lastDSPToast = time.Now()
	a.emitToast(fmt.Sprintf("DSP settings could not be applied: %v", err), "warning")
}

// cloneState devuelve una copia profunda del estado (los slices comparten
// backing array, por lo que GetState no puede devolver el estado interno).
func cloneState(s DSPState) DSPState {
	c := s
	c.Equalizer = append([]float64(nil), s.Equalizer...)
	c.DDC.Coeffs44100 = append([]float64(nil), s.DDC.Coeffs44100...)
	c.DDC.Coeffs48000 = append([]float64(nil), s.DDC.Coeffs48000...)
	return c
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
		if err := a.drv.RequireAdmin(); err != nil {
			a.emitToast("Se necesitan permisos de administrador para instalar ViPER.", "warning")
			logV("install blocked: admin required")
			return false
		}

		exePath, err := os.Executable()
		if err != nil {
			logV("install failed: executable path: %v", err)
			a.emitToast("No se pudo ubicar la carpeta de la aplicación.", "error")
			return false
		}

		resolution := resolveViPERDSPDLL(filepath.Dir(exePath))
		if resolution.path == "" {
			if len(resolution.emptyPaths) > 0 {
				a.emitToast("ViPERDSP.dll está vacío. Recompílalo o reemplázalo antes de instalar.", "error")
				logV("install blocked: empty dll at %s", strings.Join(resolution.emptyPaths, ", "))
			} else {
				a.emitToast("No se encontró ViPERDSP.dll dentro de la carpeta del ejecutable.", "error")
				logV("install blocked: dll not found in %s", strings.Join(resolution.searchPaths, ", "))
			}
			return false
		}

		if err := a.drv.RegisterAPO(resolution.path); err != nil {
			logV("install failed: register apo: %v", err)
			a.emitToast("No se pudo registrar el driver ViPER.", "error")
			return false
		}

		if err := a.drv.AttachToDefaultEndpoint(); err != nil {
			logV("attach default endpoint skipped: %v", err)
		}

		if err := a.drv.RestartAudioEngine(); err != nil {
			logV("install failed: restart audio engine: %v", err)
			a.emitToast("ViPER se registró, pero no pudo reiniciar el motor de audio.", "warning")
			return false
		}

		if err := a.dsp.SyncSharedMemory(); err != nil {
			logV("post-install shared memory unavailable: %v", err)
		} else {
			a.synchronize()
		}
		a.emitToast("ViPER quedó instalado correctamente.", "success")
		logV("install ok: %s", resolution.path)
		return true

	} else {
		if err := a.drv.RequireAdmin(); err != nil {
			a.emitToast("Se necesitan permisos de administrador para desinstalar ViPER.", "warning")
			logV("uninstall blocked: admin required")
			return false
		}
		if err := a.drv.UnregisterAPO(); err != nil {
			logV("uninstall failed: %v", err)
			a.emitToast("No se pudo desinstalar el driver ViPER.", "error")
			return false
		}
		if err := a.drv.RestartAudioEngine(); err != nil {
			logV("uninstall restart failed: %v", err)
			a.emitToast("ViPER se quitó, pero no se pudo reiniciar el motor de audio.", "warning")
			return false
		}
		a.emitToast("ViPER fue desinstalado.", "success")
		logV("uninstall ok")
		return true
	}
}

func (a *App) emitToast(message, tone string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "app:toast", toastMessage{
		Message: message,
		Tone:    tone,
	})
}

func resolveViPERDSPDLL(appDir string) dllResolution {
	candidates := driverSearchPaths(appDir)
	result := dllResolution{searchPaths: candidates}

	for _, path := range candidates {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if info.Size() > 0 {
			result.path = path
			return result
		}
		result.emptyPaths = append(result.emptyPaths, path)
	}

	return result
}

func driverSearchPaths(appDir string) []string {
	return []string{filepath.Join(appDir, "ViPERDSP.dll")}
}

// FixDriver ha sido eliminada ya que su lógica se ha integrado en SetDriverStatus
// y delegada al DriverManager.

// registerAPO ha sido eliminada ya que su lógica se ha delegado al DriverManager.

// RestartAudioServices ha sido eliminada ya que su lógica se ha delegado al DriverManager.

func (a *App) PatchDefaultEndpoint() error {
	logV("patch endpoint requested")
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

// GetState returns a deep copy of the full DSP state to the frontend.
func (a *App) GetState() DSPState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneState(a.state)
}

// ResetState restores factory defaults.
func (a *App) ResetState() DSPState {
	return a.setState(func(s *DSPState) { *s = defaultState() })
}

// SetMode switches between music / movie / freestyle.
func (a *App) SetMode(mode string) {
	a.setState(func(s *DSPState) { s.Mode = mode })
}

// SetPower toggles the master power switch in the state.
func (a *App) SetPower(on bool) {
	a.setState(func(s *DSPState) { s.Master.Power = on })
}

// SetPreVolume sets the pre-DSP volume in dB (range −40..+12).
func (a *App) SetPreVolume(db float64) {
	a.setState(func(s *DSPState) {
		s.Master.PreVol = clamp(db, MinPreVol, MaxPreVol)
	})
}

// SetPostVolume sets the post-DSP volume in dB (range −40..+12).
func (a *App) SetPostVolume(db float64) {
	a.setState(func(s *DSPState) {
		s.Master.PostVol = clamp(db, MinPostVol, MaxPostVol)
	})
}

// SetXBass replaces the full XBass state.
func (a *App) SetXBass(s XBassState) {
	a.setState(func(state *DSPState) {
		s.Level = clamp(s.Level, MinXBassLevel, MaxXBassLevel)
		s.SpeakerSize = clampInt(s.SpeakerSize, MinSpeakerSize, MaxSpeakerSize)
		state.XBass = s
	})
}

// SetXClarity replaces the full XClarity state.
func (a *App) SetXClarity(s XClarityState) {
	a.setState(func(state *DSPState) {
		s.Level = clamp(s.Level, MinXClarityLevel, MaxXClarityLevel)
		state.XClarity = s
	})
}

// SetSurround3D replaces the full 3D Surround state.
func (a *App) SetSurround3D(s Surround3DState) {
	a.setState(func(state *DSPState) {
		s.SpaceSize = clampInt(s.SpaceSize, MinSpaceSize, MaxSpaceSize)
		s.ImageSize = clampInt(s.ImageSize, MinImageSize, MaxImageSize)
		state.Surround3D = s
	})
}

// SetReverb replaces the full reverb parameter set.
func (a *App) SetReverb(p ReverbParams) {
	a.setState(func(s *DSPState) { s.Reverb = p })
}

// SetReverbPanel updates the bottom reverb strip state.
func (a *App) SetReverbPanel(p ReverbPanelState) {
	a.setState(func(s *DSPState) { s.ReverbPanel = p })
}

func (a *App) SetOutput(s OutputState) {
	a.setState(func(state *DSPState) {
		s.Pan = clamp(s.Pan, -1.0, 1.0)
		s.Limiter = clamp(s.Limiter, 0.0, 1.0)
		state.Output = s
	})
}

func (a *App) SetXBassMono(s XBassMonoState) {
	a.setState(func(state *DSPState) {
		s.Level = clamp(s.Level, MinXBassLevel, MaxXBassLevel)
		s.SpeakerSize = clampInt(s.SpeakerSize, MinSpeakerSize, MaxSpeakerSize)
		state.XBassMono = s
	})
}

func (a *App) SetConvolver(s ConvolverState) {
	a.setState(func(state *DSPState) {
		s.CrossChannel = clamp(s.CrossChannel, 0.0, 1.0)
		state.Convolver = s
	})
}

func (a *App) SetDDC(s DDCState) {
	a.setState(func(state *DSPState) {
		s.Coeffs44100 = sanitizeCoeffArray(s.Coeffs44100)
		s.Coeffs48000 = sanitizeCoeffArray(s.Coeffs48000)
		state.DDC = s
	})
}

func (a *App) SetAGC(s AGCState) {
	a.setState(func(state *DSPState) {
		s.Ratio = clamp(s.Ratio, 0.0, 20.0)
		s.Volume = clamp(s.Volume, 0.0, 20.0)
		s.MaxScaler = clamp(s.MaxScaler, 0.0, 20.0)
		state.AGC = s
	})
}

func (a *App) SetDynamicSystem(s DynamicSystemState) {
	a.setState(func(state *DSPState) {
		s.SideGainX = clamp(s.SideGainX, -20.0, 20.0)
		s.SideGainY = clamp(s.SideGainY, -20.0, 20.0)
		s.Strength = clamp(s.Strength, -20.0, 20.0)
		state.DynamicSystem = s
	})
}

func (a *App) SetSpectrumExtension(s SpectrumExtensionState) {
	a.setState(func(state *DSPState) {
		s.ReferenceFrequency = clampInt(s.ReferenceFrequency, 0, 24000)
		s.Exciter = clamp(s.Exciter, 0.0, 20.0)
		state.SpectrumExtension = s
	})
}

func (a *App) SetFieldSurround(s FieldSurroundState) {
	a.setState(func(state *DSPState) {
		s.Widening = clamp(s.Widening, 0.0, 1.0)
		s.MidImage = clamp(s.MidImage, 0.0, 1.0)
		s.Depth = clampInt(s.Depth, 0, 100)
		state.FieldSurround = s
	})
}

func (a *App) SetDiffSurround(s DiffSurroundState) {
	a.setState(func(state *DSPState) {
		s.Delay = clamp(s.Delay, 0.0, 20.0)
		state.DiffSurround = s
	})
}

func (a *App) SetCure(s CureState) {
	a.setState(func(state *DSPState) {
		s.StrengthPreset = clampInt(s.StrengthPreset, 0, 2)
		state.Cure = s
	})
}

func (a *App) SetTubeSimulator(s TubeSimulatorState) {
	a.setState(func(state *DSPState) { state.TubeSimulator = s })
}

func (a *App) SetAnalogX(s AnalogXState) {
	a.setState(func(state *DSPState) {
		s.Mode = clampInt(s.Mode, 0, 8)
		state.AnalogX = s
	})
}

func (a *App) SetFETCompressor(s FETCompressorState) {
	a.setState(func(state *DSPState) {
		s.Ratio = clamp(s.Ratio, 0.0, 20.0)
		s.Knee = clamp(s.Knee, 0.0, 20.0)
		s.Gain = clamp(s.Gain, -24.0, 24.0)
		s.Attack = clamp(s.Attack, 0.0, 100.0)
		s.Release = clamp(s.Release, 0.0, 100.0)
		s.KneeMulti = clamp(s.KneeMulti, 0.0, 20.0)
		s.MaxAttack = clamp(s.MaxAttack, 0.0, 100.0)
		s.MaxRelease = clamp(s.MaxRelease, 0.0, 100.0)
		s.Crest = clamp(s.Crest, 0.0, 20.0)
		s.Adapt = clamp(s.Adapt, 0.0, 20.0)
		state.FETCompressor = s
	})
}

func (a *App) SetSpeakerCorrection(s SpeakerCorrectionState) {
	a.setState(func(state *DSPState) { state.SpeakerCorrection = s })
}

func (a *App) SetEqBand(index int, db float64) {
	a.mu.Lock()
	if index < 0 || index >= len(a.state.Equalizer) {
		a.mu.Unlock()
		return
	}
	a.state.Equalizer[index] = clamp(db, MinEqBand, MaxEqBand)

	// Control de flujo (Throttle): la maquinaria de sincronización se invoca
	// como máximo cada 16 ms, mientras que el estado siempre queda guardado.
	shouldSync := time.Since(a.lastEqUpdate) >= 16*time.Millisecond
	if shouldSync {
		a.lastEqUpdate = time.Now()
	}
	a.mu.Unlock()

	if shouldSync {
		a.synchronize()
	}
}

// ResetEq pone todas las bandas a 0
func (a *App) ResetEq() {
	a.setState(func(s *DSPState) {
		for i := range s.Equalizer {
			s.Equalizer[i] = 0
		}
	})
}

// SetFullEq sets all equalizer bands at once.
func (a *App) SetFullEq(bands []float64) {
	a.setState(func(s *DSPState) {
		if len(bands) != len(s.Equalizer) {
			return
		}
		for i, db := range bands {
			s.Equalizer[i] = clamp(db, MinEqBand, MaxEqBand)
		}
	})
}

// CommitDSPChanges forces an explicit flush of the current state to the APO.
// This is used as a fallback when a soft DLL commit is not available.
func (a *App) CommitDSPChanges() error {
	// Fast path: try to apply to shared memory
	if err := a.dsp.ApplyChanges(a.getStateCopy()); err == nil {
		return nil
	}

	// Try to (re)initialize shared memory then apply again
	if err := a.dsp.SyncSharedMemory(); err == nil {
		if err := a.dsp.ApplyChanges(a.getStateCopy()); err == nil {
			return nil
		}
	}

	// Fallback: if the APO is registered, restart the audio engine to force reload
	if a.drv.CheckInstallation() {
		if err := a.drv.RestartAudioEngine(); err != nil {
			return fmt.Errorf("commit failed and audio restart failed: %w", err)
		}

		// After restart attempt to re-init shared memory and apply once more
		if err := a.dsp.SyncSharedMemory(); err == nil {
			_ = a.dsp.ApplyChanges(a.getStateCopy())
		}
		return nil
	}

	return fmt.Errorf("commit failed: shared memory not available and APO not installed")
}

// CommitDSPChangesAsync calls CommitDSPChanges in a goroutine so the JS caller
// won't block the UI while commit operations (which may include service restarts)
// execute. This method is intentionally fire-and-forget from the caller's view.
func (a *App) CommitDSPChangesAsync() {
	go func() {
		if err := a.CommitDSPChanges(); err != nil {
			logV("commit async failed: %v", err)
		}
	}()
}

// InstallAPOOnDefaultDevice attempts to attach the registered APO to the default
// playback endpoint and restarts the audio engine to apply changes. Returns an
// error if the APO is not registered or the attach fails.
func (a *App) InstallAPOOnDefaultDevice() error {
	dm := a.drv
	if !dm.CheckInstallation() {
		return fmt.Errorf("APO not registered")
	}

	if err := dm.AttachToDefaultEndpoint(); err != nil {
		logV("attach default endpoint failed: %v", err)
		return err
	}

	if err := a.drv.RestartAudioEngine(); err != nil {
		logV("restart audio engine failed after attach: %v", err)
		return err
	}

	// Try to initialize shared memory after successful attach
	if err := a.dsp.SyncSharedMemory(); err == nil {
		a.synchronize()
	}

	return nil
}

// ── Presets Management ────────────────────────────────────────────────────────

// presetNameRe allows only safe filename characters: it must start with a
// letter or digit (blocking ".", "..", ".hidden" and traversal), the rest may
// contain letters, digits, spaces, dots, '-' and '_' (max 64 chars).
var presetNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]{0,63}$`)

func validPresetName(name string) bool {
	return presetNameRe.MatchString(name)
}

// getPresetsDir returns the user data presets directory (%APPDATA%\Viper4Windows\presets),
// creating it if needed. Read-only installs (Program Files) no longer break saving.
func getPresetsDir() string {
	if base := os.Getenv("APPDATA"); base != "" {
		dir := filepath.Join(base, "Viper4Windows", "presets")
		if err := os.MkdirAll(dir, 0755); err == nil {
			return dir
		} else {
			logV("presets dir unavailable at %s: %v", dir, err)
		}
	}
	return legacyPresetsDir()
}

// legacyPresetsDir is the pre-0.2.x location (next to the executable).
func legacyPresetsDir() string {
	exePath, err := os.Executable()
	if err != nil {
		logV("preset dir fallback: %v", err)
		return "presets"
	}
	return filepath.Join(filepath.Dir(exePath), "presets")
}

// migratePresets copies presets from the old executable-adjacent directory to
// the AppData directory once, when the new directory has no presets yet.
func migratePresets() {
	dir := getPresetsDir()
	legacy := legacyPresetsDir()
	if filepath.Clean(dir) == filepath.Clean(legacy) {
		return
	}
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				return // already has presets
			}
		}
	}
	entries, err := os.ReadDir(legacy)
	if err != nil {
		return
	}
	migrated := 0
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(legacy, e.Name()))
		if err != nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), data, 0644); err == nil {
			migrated++
		}
	}
	if migrated > 0 {
		logV("migrated %d preset(s) from %s to %s", migrated, legacy, dir)
	}
}

// SavePreset persists the current DSP state to a named JSON file.
func (a *App) SavePreset(name string) error {
	name = strings.TrimSpace(name)
	if !validPresetName(name) {
		return fmt.Errorf("invalid preset name %q: use letters, digits, spaces, '-' or '_' (max 64 chars)", name)
	}

	dir := getPresetsDir()
	data, err := json.MarshalIndent(a.getStateCopy(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(dir, name+".json")
	return os.WriteFile(path, data, 0644)
}

// LoadPreset reads a preset by name and returns the full DSP state.
func (a *App) LoadPreset(name string) (DSPState, error) {
	name = strings.TrimSpace(name)
	if !validPresetName(name) {
		return a.getStateCopy(), fmt.Errorf("invalid preset name %q", name)
	}

	dir := getPresetsDir()
	data, err := os.ReadFile(filepath.Join(dir, name+".json"))
	if err != nil {
		return a.getStateCopy(), fmt.Errorf("read preset: %w", err)
	}
	s, err := decodePresetState(data)
	if err != nil {
		return a.getStateCopy(), fmt.Errorf("unmarshal: %w", err)
	}
	a.setState(func(state *DSPState) { *state = s })
	return cloneState(a.state), nil
}

// getStateCopy returns a deep copy of the current state.
func (a *App) getStateCopy() DSPState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneState(a.state)
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
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".json")
		if validPresetName(base) {
			names = append(names, base)
		}
	}
	return names, nil
}

// DeletePreset removes a saved preset by name.
func (a *App) DeletePreset(name string) error {
	name = strings.TrimSpace(name)
	if !validPresetName(name) {
		return fmt.Errorf("invalid preset name %q", name)
	}
	dir := getPresetsDir()
	path := filepath.Join(dir, name+".json")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete preset: %w", err)
	}
	logV("preset deleted: %s", name)
	return nil
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

func sanitizeCoeffArray(values []float64) []float64 {
	if len(values) == 0 {
		return []float64{}
	}

	count := (len(values) / 5) * 5
	if count == 0 {
		return []float64{}
	}

	clean := make([]float64, count)
	copy(clean, values[:count])
	return clean
}

func decodePresetState(data []byte) (DSPState, error) {
	s := defaultState()
	if err := json.Unmarshal(data, &s); err != nil {
		return DSPState{}, err
	}
	return normalizeDSPState(s), nil
}

func normalizeDSPState(s DSPState) DSPState {
	defaults := defaultState()

	s.Master.PreVol = clamp(s.Master.PreVol, MinPreVol, MaxPreVol)
	s.Master.PostVol = clamp(s.Master.PostVol, MinPostVol, MaxPostVol)
	s.Output.Pan = clamp(s.Output.Pan, -1.0, 1.0)
	s.Output.Limiter = clamp(s.Output.Limiter, 0.0, 1.0)

	s.XBass.Level = clamp(s.XBass.Level, MinXBassLevel, MaxXBassLevel)
	s.XBass.SpeakerSize = clampInt(s.XBass.SpeakerSize, MinSpeakerSize, MaxSpeakerSize)
	s.XBassMono.Level = clamp(s.XBassMono.Level, MinXBassLevel, MaxXBassLevel)
	s.XBassMono.SpeakerSize = clampInt(s.XBassMono.SpeakerSize, MinSpeakerSize, MaxSpeakerSize)
	s.XClarity.Level = clamp(s.XClarity.Level, MinXClarityLevel, MaxXClarityLevel)
	s.Surround3D.SpaceSize = clampInt(s.Surround3D.SpaceSize, MinSpaceSize, MaxSpaceSize)
	s.Surround3D.ImageSize = clampInt(s.Surround3D.ImageSize, MinImageSize, MaxImageSize)

	s.Convolver.CrossChannel = clamp(s.Convolver.CrossChannel, 0.0, 1.0)
	s.DDC.Coeffs44100 = sanitizeCoeffArray(s.DDC.Coeffs44100)
	s.DDC.Coeffs48000 = sanitizeCoeffArray(s.DDC.Coeffs48000)
	s.AGC.Ratio = clamp(s.AGC.Ratio, 0.0, 20.0)
	s.AGC.Volume = clamp(s.AGC.Volume, 0.0, 20.0)
	s.AGC.MaxScaler = clamp(s.AGC.MaxScaler, 0.0, 20.0)
	s.DynamicSystem.SideGainX = clamp(s.DynamicSystem.SideGainX, -20.0, 20.0)
	s.DynamicSystem.SideGainY = clamp(s.DynamicSystem.SideGainY, -20.0, 20.0)
	s.DynamicSystem.Strength = clamp(s.DynamicSystem.Strength, -20.0, 20.0)
	s.SpectrumExtension.ReferenceFrequency = clampInt(s.SpectrumExtension.ReferenceFrequency, 0, 24000)
	s.SpectrumExtension.Exciter = clamp(s.SpectrumExtension.Exciter, 0.0, 20.0)
	s.FieldSurround.Widening = clamp(s.FieldSurround.Widening, 0.0, 1.0)
	s.FieldSurround.MidImage = clamp(s.FieldSurround.MidImage, 0.0, 1.0)
	s.FieldSurround.Depth = clampInt(s.FieldSurround.Depth, 0, 100)
	s.DiffSurround.Delay = clamp(s.DiffSurround.Delay, 0.0, 20.0)
	s.Cure.StrengthPreset = clampInt(s.Cure.StrengthPreset, 0, 2)
	s.AnalogX.Mode = clampInt(s.AnalogX.Mode, 0, 8)
	s.FETCompressor.Ratio = clamp(s.FETCompressor.Ratio, 0.0, 20.0)
	s.FETCompressor.Knee = clamp(s.FETCompressor.Knee, 0.0, 20.0)
	s.FETCompressor.Gain = clamp(s.FETCompressor.Gain, -24.0, 24.0)
	s.FETCompressor.Attack = clamp(s.FETCompressor.Attack, 0.0, 100.0)
	s.FETCompressor.Release = clamp(s.FETCompressor.Release, 0.0, 100.0)
	s.FETCompressor.KneeMulti = clamp(s.FETCompressor.KneeMulti, 0.0, 20.0)
	s.FETCompressor.MaxAttack = clamp(s.FETCompressor.MaxAttack, 0.0, 100.0)
	s.FETCompressor.MaxRelease = clamp(s.FETCompressor.MaxRelease, 0.0, 100.0)
	s.FETCompressor.Crest = clamp(s.FETCompressor.Crest, 0.0, 20.0)
	s.FETCompressor.Adapt = clamp(s.FETCompressor.Adapt, 0.0, 20.0)

	if len(s.Equalizer) != len(defaults.Equalizer) {
		eq := make([]float64, len(defaults.Equalizer))
		copy(eq, s.Equalizer)
		s.Equalizer = eq
	}
	for i := range s.Equalizer {
		s.Equalizer[i] = clamp(s.Equalizer[i], MinEqBand, MaxEqBand)
	}

	return s
}
