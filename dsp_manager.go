package main

import (
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// VIPER_DSP_PARAMS must match exactly the memory layout of ViPERFX_RE (DSP.h)
// Total size: 4096 bytes (padded for future expansions)
type VIPER_DSP_PARAMS struct {
	Enabled float32 // 0: Off, 1: On
	PreVol  float32 // dB
	PostVol float32 // dB

	// Equalizer Block
	EqEnabled float32
	EqBands   [18]float32 // 18-band precision control

	// ViPER Bass Block
	BassEnabled float32
	BassMode    float32 // 0: Natural, 1: Pure
	BassSpkSize float32 // Frequency Cutoff
	BassGain    float32 // dB

	// ViPER Clarity Block
	ClarityEnabled float32
	ClarityMode    float32 // 0: Natural, 1: OZone+, 2: X-HiFi
	ClarityGain    float32 // dB

	// Surround / Field Block
	SurroundEnabled float32
	SurroundSize    float32

	// Convolver Block (RE: /ViPERFX_RE/Convolver.cpp)
	ConvolverEnabled float32

	// Reverb Block
	ReverbEnabled float32
	ReverbRoom    float32
	ReverbDamp    float32
	ReverbMix     float32

	_ [950]float32 // Padding to maintain alignment with the 4KB block
}

const (
	SharedMemName = "Global\\ViPER4Windows_SharedMem"
	SharedMemSize = 4096
)

type DSPManager struct {
	mu      sync.Mutex
	handle  windows.Handle
	dataPtr uintptr
	params  *VIPER_DSP_PARAMS // Pre-allocated to avoid GC
}

// InitSharedMemory opens the communication channel with the APO.
// Prefers windows.NewLazySystemDLL (kernel32) to avoid CGO.
func (dm *DSPManager) InitSharedMemory() error {
	namePtr, _ := windows.UTF16PtrFromString(SharedMemName)

	// Open or create the file mapping
	h, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		0,
		SharedMemSize,
		namePtr,
	)
	if err != nil {
		return fmt.Errorf("could not create shared memory: %w", err)
	}

	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_WRITE, 0, 0, 0)
	if err != nil {
		windows.CloseHandle(h)
		return fmt.Errorf("could not map view of file: %w", err)
	}

	dm.handle = h
	dm.dataPtr = addr
	// Project our Go struct onto the raw memory address
	dm.params = (*VIPER_DSP_PARAMS)(unsafe.Pointer(addr))

	return nil
}

// writeToSharedMemory escribe los datos del preset en el bloque de memoria compartida
// que el driver de audio espera leer.
func (a *App) writeToSharedMemory(data []byte) error {
	memName := "Global\\ViPER4Windows_DSP_Memory"
	namePtr, _ := windows.UTF16PtrFromString(memName)

	// 1. La función es OpenFileMapping (con 3 argumentos en el paquete windows)
	// Argumentos: (access, inheritHandle, name)
	// OJO: En Go, 'windows.OpenFileMapping' requiere:
	// access (uint32), inherit (bool), name (*uint16)
	handle, err := windows.OpenFileMapping(uint32(windows.FILE_MAP_WRITE), false, namePtr)
	if err != nil {
		return fmt.Errorf("error al abrir el mapeo (¿está el driver activo?): %v", err)
	}
	defer windows.CloseHandle(handle)

	// 2. MapViewOfFile (5 argumentos)
	// (handle, access, offsetHigh, offsetLow, size)
	ptr, err := windows.MapViewOfFile(handle, windows.FILE_MAP_WRITE, 0, 0, uintptr(len(data)))
	if err != nil {
		return fmt.Errorf("error en MapViewOfFile: %v", err)
	}
	defer windows.UnmapViewOfFile(ptr)

	// 3. Copiar datos usando unsafe
	outBuf := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:len(data):len(data)]
	copy(outBuf, data)

	return nil
}

// ApplyChanges syncs the Go state to the APO memory.
// This is the 'Hot Path'. No allocations occur here.
func (dm *DSPManager) ApplyChanges(state DSPState) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.params == nil {
		return
	}

	// Atomic-like update via memory projection
	p := dm.params
	p.Enabled = boolToFloat32(state.Master.Power)
	p.PreVol = float32(state.Master.PreVol)
	p.PostVol = float32(state.Master.PostVol)

	// EQ Port
	p.EqEnabled = 1.0
	for i := 0; i < 18; i++ {
		p.EqBands[i] = float32(state.Equalizer[i])
	}

	// Bass Logic (Mapping from ViPERFX_RE Algorithms)
	p.BassEnabled = boolToFloat32(state.XBass.On)
	p.BassGain = float32(state.XBass.Level)
	p.BassSpkSize = float32(state.XBass.SpeakerSize)
	if state.XBass.Mode == "Pure Bass" {
		p.BassMode = 1.0
	} else {
		p.BassMode = 0.0
	}

	// Clarity Logic
	p.ClarityEnabled = boolToFloat32(state.XClarity.On)
	p.ClarityGain = float32(state.XClarity.Level)
	switch state.XClarity.Mode {
	case "OZone+":
		p.ClarityMode = 1.0
	case "X-HiFi":
		p.ClarityMode = 2.0
	default: // Natural
		p.ClarityMode = 0.0
	}

	// Surround Logic
	p.SurroundEnabled = boolToFloat32(state.Surround3D.On)
	p.SurroundSize = float32(state.Surround3D.SpaceSize)

	// Reverb Logic
	p.ReverbEnabled = boolToFloat32(state.Reverb.On)
	p.ReverbRoom = float32(state.Reverb.RoomSize)
	p.ReverbDamp = float32(state.Reverb.Damping)
	p.ReverbMix = float32(state.Reverb.WetMix)
}

func boolToFloat32(b bool) float32 {
	if b {
		return 1.0
	}
	return 0.0
}

// Close libera los recursos del mapeo de memoria
func (dm *DSPManager) Close() {
	if dm.dataPtr != 0 {
		windows.UnmapViewOfFile(dm.dataPtr)
	}
	if dm.handle != 0 {
		windows.CloseHandle(dm.handle)
	}
}
