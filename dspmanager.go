package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	SharedMemName = "Global\\ViPER4Windows_SharedMemory" // ← cambiado
	EventName     = "Global\\ViPER4Windows_Event"        // ← nuevo

	SharedMemSize = 4096 // la mayoría de forks usan exactamente 4096 bytes
)

type VIPER_DSP_PARAMS struct {
	Params [256]float32
}

const (
	IDX_ENABLED            = 0
	IDX_PREVOL             = 1
	IDX_POSTVOL            = 2
	IDX_EQ_ENABLED         = 4
	IDX_EQ_BANDS           = 5 // 18 bandas consecutivas
	IDX_XBASS_ENABLED      = 23
	IDX_XBASS_MODE         = 24
	IDX_XBASS_SPKSIZE      = 25
	IDX_XBASS_GAIN         = 26
	IDX_XCLARITY_ENABLED   = 27
	IDX_XCLARITY_MODE      = 28
	IDX_XCLARITY_GAIN      = 29
	IDX_SURROUND_ENABLED   = 30
	IDX_SURROUND_SIZE      = 31
	IDX_REVERB_ENABLED     = 32
	IDX_REVERB_ROOM        = 33
	IDX_REVERB_DAMP        = 34
	IDX_REVERB_MIX         = 35
	IDX_CONVOLVER_ENABLED  = 36
	IDX_CURETECH_ENABLED   = 37
	IDX_CURETECH_LEVEL     = 38
	IDX_ANALOGX_ENABLED    = 39
	IDX_ANALOGX_MODE       = 40
	IDX_SPEAKEROPT_ENABLED = 41
	IDX_SPEAKEROPT_MODE    = 42
	IDX_SPEAKEROPT_GAIN    = 43
	IDX_LIMITER_ENABLED    = 44
	IDX_LIMITER_THRESHOLD  = 45
	IDX_MAX                = 255
	// ... otros índices para efectos adicionales

)

/*
	type VIPER_DSP_PARAMS struct {
		// Master
		Enabled float32
		PreVol  float32
		PostVol float32

		// EQ (18 bandas)
		EqEnabled float32
		EqBands   [18]float32

		// Bass
		BassEnabled float32
		BassMode    float32 // 0 = Natural, 1 = Pure
		BassSpkSize float32
		BassGain    float32

		// Clarity
		ClarityEnabled float32
		ClarityMode    float32 // 0=Natural, 1=OZone+, 2=X-HiFi
		ClarityGain    float32

		// Surround
		SurroundEnabled float32
		SurroundSize    float32

		// Reverb
		ReverbEnabled float32
		ReverbRoom    float32
		ReverbDamp    float32
		ReverbMix     float32

		// Resto (padding)
		_ [2003]float32 // mantiene exactamente 4096 bytes
	}
*/
type DSPManager struct {
	memHandle   windows.Handle
	eventHandle windows.Handle
	dataPtr     uintptr
	params      *VIPER_DSP_PARAMS
	connected   bool

	activeBuf uint32
	frontBuf  VIPER_DSP_PARAMS
	backBuf   VIPER_DSP_PARAMS
}

func (dm *DSPManager) SyncSharedMemory() error {
	// 1. Validar instalación
	if !(&DriverManager{}).CheckInstallation() {
		return fmt.Errorf("APO no instalado en el registro")
	}

	// 2. CreateFileMapping (Si existe, lo abre; si no, lo crea)
	// Usamos PAGE_READWRITE para tener control total
	hMap, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		0,
		1024, // Tamaño exacto de tus 256 floats
		windows.StringToUTF16Ptr("Global\\ViPER4Windows_SharedMemory"),
	)
	if err != nil {
		dm.connected = false
		return fmt.Errorf("error al mapear archivo: %v", err)
	}

	// 3. MapViewOfFile - ¡IMPORTANTE!: Debe ser FILE_MAP_WRITE o FILE_MAP_ALL_ACCESS
	ptr, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, 1024)
	if err != nil {
		windows.CloseHandle(hMap)
		dm.connected = false
		return fmt.Errorf("error al mapear vista: %v", err)
	}

	// Guardamos los handles para evitar que el GC los limpie
	dm.memHandle = hMap
	dm.params = (*VIPER_DSP_PARAMS)(unsafe.Pointer(ptr))

	// 4. Sincronización del Evento
	// Intentamos crearlo. Si ya existe, Windows nos da el handle al existente.
	hEvent, err := windows.CreateEvent(nil, 0, 0, windows.StringToUTF16Ptr("Global\\ViPER4Windows_Event"))
	if err != nil {
		// Si falla crear, intentamos solo abrir
		hEvent, _ = windows.OpenEvent(windows.EVENT_MODIFY_STATE, false, windows.StringToUTF16Ptr("Global\\ViPER4Windows_Event"))
	}
	dm.eventHandle = hEvent

	dm.connected = (dm.params != nil)
	log.Printf("🚀 Motor de Memoria Sincronizado (Connected: %v)", dm.connected)
	return nil
}

func (dm *DSPManager) ApplyChanges(state DSPState) error {
	if dm.connected && dm.params == nil {
		return fmt.Errorf("shared memory no inicializada")
	}

	// Double buffering (sin locks)
	target := &dm.frontBuf
	if atomic.LoadUint32(&dm.activeBuf) == 1 {
		target = &dm.backBuf
	}

	dm.fillBuffer(target, state)

	// Copia atómica
	src := unsafe.Pointer(target)
	dst := unsafe.Pointer(dm.params)
	memmove(dst, src, SharedMemSize)

	// === SEÑAL CRÍTICA ===
	if dm.eventHandle != 0 {
		windows.SetEvent(dm.eventHandle) // ← esto es lo que faltaba
	}

	atomic.StoreUint32(&dm.activeBuf, 1-atomic.LoadUint32(&dm.activeBuf))
	return nil
}

// fillBuffer populates a VIPER_DSP_PARAMS buffer from DSPState
// This is separated from ApplyChanges to improve testability
func (dm *DSPManager) fillBuffer(target *VIPER_DSP_PARAMS, state DSPState) {
	// Limpiamos el buffer por si acaso
	*target = VIPER_DSP_PARAMS{}

	// --- MAREO DE ÍNDICES ---
	target.Params[IDX_ENABLED] = 1.0
	if !state.Master.Power {
		target.Params[IDX_ENABLED] = 0.0
	}

	target.Params[IDX_PREVOL] = float32(state.Master.PreVol)
	target.Params[IDX_POSTVOL] = float32(state.Master.PostVol)

	// EQ
	target.Params[IDX_EQ_ENABLED] = 1.0
	for i := 0; i < len(state.Equalizer) && i < 18; i++ {
		target.Params[IDX_EQ_BANDS+i] = float32(state.Equalizer[i])
	}

	// X-Bass
	if state.XBass.On {
		target.Params[IDX_XBASS_ENABLED] = 1.0
		target.Params[IDX_XBASS_GAIN] = float32(state.XBass.Level)
		target.Params[IDX_XBASS_SPKSIZE] = float32(state.XBass.SpeakerSize)
	}

	// X-Clarity
	if state.XClarity.On {
		target.Params[IDX_XCLARITY_ENABLED] = 1.0
		target.Params[IDX_XCLARITY_GAIN] = float32(state.XClarity.Level)
	}

	// Reverb (El panel simplificado)
	if state.Reverb.On {
		target.Params[IDX_REVERB_ENABLED] = 1.0
		target.Params[IDX_REVERB_MIX] = float32(state.Reverb.WetMix)
	}
}

/*
func (dm *DSPManager) fillBuffer(buffer *VIPER_DSP_PARAMS, state DSPState) {
	// Master controls
	buffer.Enabled = boolToFloat32(state.Master.Power)
	buffer.PreVol = float32(state.Master.PreVol)
	buffer.PostVol = float32(state.Master.PostVol)

	// Equalizer
	buffer.EqEnabled = boolToFloat32(state.EqOn)
	for i := 0; i < 18 && i < len(state.Equalizer); i++ {
		buffer.EqBands[i] = float32(state.Equalizer[i])
	}

	// ViPER Bass
	buffer.BassEnabled = boolToFloat32(state.XBass.On)
	buffer.BassGain = float32(state.XBass.Level)
	buffer.BassSpkSize = float32(state.XBass.SpeakerSize)
	buffer.BassMode = float32(parseBassMode(state.XBass.Mode))

	// ViPER Clarity
	buffer.ClarityEnabled = boolToFloat32(state.XClarity.On)
	buffer.ClarityGain = float32(state.XClarity.Level)
	buffer.ClarityMode = float32(parseClarityMode(state.XClarity.Mode))

	// Surround 3D
	buffer.SurroundEnabled = boolToFloat32(state.Surround3D.On)
	buffer.SurroundSize = float32(state.Surround3D.SpaceSize)

	// Reverb
	buffer.ReverbEnabled = boolToFloat32(state.Reverb.On)
	buffer.ReverbRoom = float32(state.Reverb.RoomSize)
	buffer.ReverbDamp = float32(state.Reverb.Damping)
	buffer.ReverbMix = float32(state.Reverb.WetMix)

	// Additional effects (if state structure supports them)
	// These are placeholders - implement based on actual DSPState
	// buffer.ConvolverEnabled = 0.0
	// buffer.CureTechEnabled = 0.0
	// buffer.CureTechLevel = 0.0
	// buffer.AnalogXEnabled = 0.0
	// buffer.AnalogXMode = 0.0
	// buffer.SpeakerOptEnabled = 0.0
	// buffer.SpeakerOptMode = 0.0
	// buffer.SpeakerOptGain = 0.0
	// buffer.LimiterEnabled = 0.0
	// buffer.LimiterThreshold = 0.0
}
*/

func (dm *DSPManager) Close() {
	if dm.eventHandle != 0 {
		windows.CloseHandle(dm.eventHandle)
	}
	if dm.dataPtr != 0 {
		windows.UnmapViewOfFile(dm.dataPtr)
	}
	if dm.memHandle != 0 {
		windows.CloseHandle(dm.memHandle)
	}
	dm.connected = false
	log.Println("✓ Shared memory resources released")
}

// helper functions (boolToFloat32, parseBassMode, etc.)

// boolToFloat32 converts boolean to float32 (0.0 or 1.0)
func boolToFloat32(b bool) float32 {
	if b {
		return 1.0
	}
	return 0.0
}

// parseBassMode converts string mode to numeric value
func parseBassMode(mode string) int {
	switch mode {
	case "Pure Bass":
		return 1
	default: // "Natural Bass"
		return 0
	}
}

// parseClarityMode converts string mode to numeric value
func parseClarityMode(mode string) int {
	switch mode {
	case "OZone+":
		return 1
	case "X-HiFi":
		return 2
	default: // "Natural"
		return 0
	}
}

// memmove is a low-level memory copy function from the Go runtime
// We link to it directly to ensure atomic behavior for our buffer swaps
//
//go:linkname memmove runtime.memmove
func memmove(to, from unsafe.Pointer, n uintptr)
