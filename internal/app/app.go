package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
	"viper4windows/internal/infrastructure/windows"
	"viper4windows/internal/services"
	"viper4windows/internal/utils"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ═══════════════════════════════════════════════════════════════════════════
// Application Core
// ═══════════════════════════════════════════════════════════════════════════
// Clean, single-responsibility coordinator following reference architecture.
// This replaces the monolithic 1051-line app.go with a focused orchestrator.
//
// Moved here from the repo root: main.go already lives under cmd/, and a
// Wails-bound App struct is presentation/orchestration wiring, not
// something that belongs at the module root next to go.mod. See
// RESTRUCTURE_NOTES.md for the full reasoning.

type App struct {
	// Dependencies (injected)
	logger        ports.Logger
	sharedMem     ports.SharedMemoryPort
	paramService  ports.DSPParameterPort
	presetManager ports.PresetRepository
	security      ports.SecurityPort

	// Application state
	state      models.DSPState
	stateMutex sync.RWMutex
	lastUpdate time.Time
	updateRate time.Duration

	// Wails context
	ctx context.Context
}

// ─────────────────────────────────────────────────────────────────────────────
// Constructor & Initialization
// ─────────────────────────────────────────────────────────────────────────────

// NewApp creates a new application instance with dependency injection
func NewApp(presetsDir string) *App {
	// Create logger
	logger := utils.NewLogger("ViPER4Windows")

	// Create services
	securitySvc := windows.NewSecurityService(logger)
	paramSvc := services.NewDSPParameterService(logger)
	// NewSharedMemoryService lives in internal/infrastructure/windows now
	// (it's raw kernel32 syscalls, not application logic — it used to sit
	// in internal/services under the same name, which this call site
	// still needs to reflect after that move).
	sharedMemSvc := windows.NewSharedMemoryService(logger, securitySvc, paramSvc)
	presetMgr := services.NewPresetManagerService(logger, paramSvc, presetsDir)

	return &App{
		logger:        logger,
		sharedMem:     sharedMemSvc,
		paramService:  paramSvc,
		presetManager: presetMgr,
		security:      securitySvc,
		state:         models.NewDefaultState(),
		updateRate:    16 * time.Millisecond, // 60Hz max update rate
	}
}

// Startup is called by Wails on application start
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.Info("application starting")

	// Initialize shared memory
	if err := a.sharedMem.Open(); err != nil {
		a.logger.Error("failed to open shared memory", err)
	}

	// Load last state if available
	// (would call a settings repository here)

	// Apply initial state
	a.synchronize()

	a.logger.Info("application started successfully")
}

// Shutdown is called by Wails on application exit
func (a *App) Shutdown(ctx context.Context) {
	a.logger.Info("application shutting down")

	// Save current state
	// (would save to settings repository here)

	// Close shared memory
	if err := a.sharedMem.Close(); err != nil {
		a.logger.Error("failed to close shared memory", err)
	}

	a.logger.Info("application shutdown complete")
}

// ─────────────────────────────────────────────────────────────────────────────
// State Management
// ─────────────────────────────────────────────────────────────────────────────

// GetState returns the current DSP state (safe copy)
func (a *App) GetState() models.DSPState {
	a.stateMutex.RLock()
	defer a.stateMutex.RUnlock()
	return a.state
}

// UpdateState applies partial state updates and synchronizes with APO
func (a *App) UpdateState(updates map[string]interface{}) error {
	a.stateMutex.Lock()
	defer a.stateMutex.Unlock()

	// Apply updates to state
	// (implementation would use reflection or a mapping strategy)

	// Validate updated state
	if err := a.paramService.ValidateState(&a.state); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Synchronize with rate limiting
	a.synchronize()

	return nil
}

// synchronize writes the current state to the APO with rate limiting
func (a *App) synchronize() {
	// Rate limiting: only update every 16ms (60Hz)
	if time.Since(a.lastUpdate) < a.updateRate {
		return
	}

	if err := a.sharedMem.WriteParams(a.state); err != nil {
		a.logger.Error("failed to write parameters", err)
	}

	a.lastUpdate = time.Now()
}

// ─────────────────────────────────────────────────────────────────────────────
// Master Controls (Wails-exposed methods)
// ─────────────────────────────────────────────────────────────────────────────

// SetPower toggles the DSP on/off
func (a *App) SetPower(enabled bool) {
	a.stateMutex.Lock()
	a.state.Master.Power = enabled
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetPreVol sets pre-amplification volume
func (a *App) SetPreVol(db float64) {
	a.stateMutex.Lock()
	a.state.Master.PreVol = utils.Clamp(db, models.MinPreVol, models.MaxPreVol)
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetPostVol sets post-amplification volume
func (a *App) SetPostVol(db float64) {
	a.stateMutex.Lock()
	a.state.Master.PostVol = utils.Clamp(db, models.MinPostVol, models.MaxPostVol)
	a.stateMutex.Unlock()
	a.synchronize()
}

// ─────────────────────────────────────────────────────────────────────────────
// Equalizer Controls
// ─────────────────────────────────────────────────────────────────────────────

// SetEqEnabled toggles the equalizer
func (a *App) SetEqEnabled(enabled bool) {
	a.stateMutex.Lock()
	a.state.EqOn = enabled
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetEqBand sets a single EQ band value
func (a *App) SetEqBand(band int, db float64) {
	if band < 0 || band >= models.EqBands {
		return
	}

	a.stateMutex.Lock()
	a.state.Equalizer[band] = utils.Clamp(db, models.MinEqBand, models.MaxEqBand)
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetFullEq sets all equalizer bands at once
func (a *App) SetFullEq(bands []float64) {
	if len(bands) != models.EqBands {
		return
	}

	a.stateMutex.Lock()
	for i, db := range bands {
		a.state.Equalizer[i] = utils.Clamp(db, models.MinEqBand, models.MaxEqBand)
	}
	a.stateMutex.Unlock()
	a.synchronize()
}

// ResetEq resets all EQ bands to 0 dB
func (a *App) ResetEq() {
	a.stateMutex.Lock()
	for i := range a.state.Equalizer {
		a.state.Equalizer[i] = 0
	}
	a.stateMutex.Unlock()
	a.synchronize()
}

// ─────────────────────────────────────────────────────────────────────────────
// Effect Controls
// ─────────────────────────────────────────────────────────────────────────────

// SetXBass updates XBass parameters
func (a *App) SetXBass(enabled bool, speakerSize int, level float64, mode string) {
	a.stateMutex.Lock()
	a.state.XBass.On = enabled
	a.state.XBass.SpeakerSize = utils.ClampInt(speakerSize, models.MinSpeakerSize, models.MaxSpeakerSize)
	a.state.XBass.Level = utils.Clamp(level, models.MinXBassLevel, models.MaxXBassLevel)
	a.state.XBass.Mode = models.XBassMode(mode)
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetXClarity updates XClarity parameters
func (a *App) SetXClarity(enabled bool, level float64, mode string) {
	a.stateMutex.Lock()
	a.state.XClarity.On = enabled
	a.state.XClarity.Level = utils.Clamp(level, models.MinXClarityLevel, models.MaxXClarityLevel)
	a.state.XClarity.Mode = models.XClarityMode(mode)
	a.stateMutex.Unlock()
	a.synchronize()
}

// SetSurround3D updates 3D surround parameters
func (a *App) SetSurround3D(enabled bool, spaceSize int, roomSize string, imageSize int) {
	a.stateMutex.Lock()
	a.state.Surround3D.On = enabled
	a.state.Surround3D.SpaceSize = utils.ClampInt(spaceSize, models.MinSpaceSize, models.MaxSpaceSize)
	a.state.Surround3D.RoomSize = roomSize
	a.state.Surround3D.ImageSize = utils.ClampInt(imageSize, models.MinImageSize, models.MaxImageSize)
	a.stateMutex.Unlock()
	a.synchronize()
}

// ─────────────────────────────────────────────────────────────────────────────
// Preset Management
// ─────────────────────────────────────────────────────────────────────────────

// SavePreset saves the current state as a named preset
func (a *App) SavePreset(name string) error {
	a.stateMutex.RLock()
	state := a.state
	a.stateMutex.RUnlock()

	return a.presetManager.Save(name, state)
}

// LoadPreset loads a preset and applies it
func (a *App) LoadPreset(name string) (models.DSPState, error) {
	state, err := a.presetManager.Load(name)
	if err != nil {
		return models.DSPState{}, err
	}

	a.stateMutex.Lock()
	a.state = state
	a.stateMutex.Unlock()

	a.synchronize()
	return state, nil
}

// ListPresets returns all available preset names
func (a *App) ListPresets() ([]string, error) {
	return a.presetManager.List()
}

// DeletePreset removes a preset
func (a *App) DeletePreset(name string) error {
	return a.presetManager.Delete(name)
}

// ─────────────────────────────────────────────────────────────────────────────
// System Information
// ─────────────────────────────────────────────────────────────────────────────

// IsElevated checks if running as administrator
func (a *App) IsElevated() bool {
	return a.security.IsElevated()
}

// GetAPOStatus returns APO connection status
func (a *App) GetAPOStatus() models.APOStatus {
	return models.APOStatus{
		IsInstalled: a.sharedMem.IsConnected(),
		IsAttached:  a.sharedMem.IsConnected(),
		Version:     "1.0.0", // Would read from shared memory
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Wails lifecycle glue (moved here from the old cmd/internal/main.go,
// which was calling a lowercase app.startup()/NewApp() with no args —
// leftover from before this Clean Architecture pass existed. See
// RESTRUCTURE_NOTES.md.
// ─────────────────────────────────────────────────────────────────────────────

// OnSecondInstanceLaunch focuses the existing window when the user
// launches the app again while it's already running (Wails'
// SingleInstanceLock hands this the second launch's argv instead of
// starting a second process). Needs a.ctx, which is why this lives on
// App rather than as a free function in cmd/ — ctx isn't exported.
func (a *App) OnSecondInstanceLaunch(secondInstanceData options.SecondInstanceData) {
	args := secondInstanceData.Args
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)
	runtime.EventsEmit(a.ctx, "launchArgs", args)
	a.logger.Info("second instance launch blocked", "args", strings.Join(args, " "))
}
