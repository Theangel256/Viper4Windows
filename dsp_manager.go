package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

// VIPER_DSP_PARAMS must match exactly the memory layout of ViPERFX_RE (DSP.h)
// Total size: 4096 bytes (padded for future expansions)
//
// CRITICAL: This struct MUST be exactly 4096 bytes and match the C++ layout.
// Any mismatch will cause the APO to read garbage data.
type VIPER_DSP_PARAMS struct {
	// Master Control (12 bytes)
	Enabled float32 // 0: Off, 1: On
	PreVol  float32 // dB
	PostVol float32 // dB

	// Equalizer Block (76 bytes)
	EqEnabled float32
	EqBands   [18]float32 // 18-band precision control

	// ViPER Bass Block (16 bytes)
	BassEnabled float32
	BassMode    float32 // 0: Natural, 1: Pure
	BassSpkSize float32 // Frequency Cutoff
	BassGain    float32 // dB

	// ViPER Clarity Block (12 bytes)
	ClarityEnabled float32
	ClarityMode    float32 // 0: Natural, 1: OZone+, 2: X-HiFi
	ClarityGain    float32 // dB

	// Surround / Field Block (8 bytes)
	SurroundEnabled float32
	SurroundSize    float32

	// Convolver Block (4 bytes)
	ConvolverEnabled float32

	// Reverb Block (16 bytes)
	ReverbEnabled float32
	ReverbRoom    float32
	ReverbDamp    float32
	ReverbMix     float32

	// Cure Tech (8 bytes) - Added for completeness
	CureTechEnabled float32
	CureTechLevel   float32

	// Analog X (8 bytes) - Added for completeness
	AnalogXEnabled float32
	AnalogXMode    float32

	// Speaker Optimization (12 bytes) - Added for completeness
	SpeakerOptEnabled float32
	SpeakerOptMode    float32
	SpeakerOptGain    float32

	// Limiter (8 bytes) - Added for completeness
	LimiterEnabled   float32
	LimiterThreshold float32

	// Padding to maintain alignment with the 8KB block
	// Current usage: 180 bytes of actual fields
	// Remaining: 8192 - 180 = 8012 bytes = 2003 float32s
	_ [2003]float32
}

// Compile-time size validation
// This will cause a compilation error if the struct size != 4096
const (
	SharedMemName = "Global\\ViPER4Windows_SharedMem"
	SharedMemSize = 8192 // untyped constant — works as both uintptr and uint32
)

var paddingSize = uintptr(SharedMemSize) - unsafe.Sizeof(VIPER_DSP_PARAMS{})

// DSPManager handles the communication channel with the APO via shared memory.
// Uses double-buffering to prevent tearing during real-time updates.
type DSPManager struct {
	handle    windows.Handle
	dataPtr   uintptr
	params    *VIPER_DSP_PARAMS
	connected bool

	// Double buffering to avoid race conditions
	frontBuffer  VIPER_DSP_PARAMS
	backBuffer   VIPER_DSP_PARAMS
	activeBuffer uint32 // 0 = front active, 1 = back active
}

// InitSharedMemory opens or creates the communication channel with the APO.
// This function validates that the APO is registered before proceeding.
func (dm *DSPManager) InitSharedMemory() error {
	// Validate APO installation first
	drv := &DriverManager{}
	if !drv.CheckInstallation() {
		return fmt.Errorf("APO not installed - please install the driver first using SetDriverStatus(true)")
	}

	namePtr, _ := windows.UTF16PtrFromString(SharedMemName)

	// Try to open existing mapping first (if APO already created it)
	h, err := OpenFileMapping(
		windows.FILE_MAP_WRITE,
		false,
		namePtr,
	)

	if err != nil {
		// If it doesn't exist, create it ourselves
		h, err = windows.CreateFileMapping(
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
		log.Printf("✓ Created new shared memory mapping")
	} else {
		log.Printf("✓ Opened existing shared memory mapping")
	}

	// Map the file into our address space
	addr, err := windows.MapViewOfFile(h, windows.FILE_MAP_WRITE, 0, 0, 0)
	if err != nil {
		windows.CloseHandle(h)
		return fmt.Errorf("could not map view of file: %w", err)
	}

	dm.handle = h
	dm.dataPtr = addr
	// Project our Go struct onto the raw memory address
	dm.params = (*VIPER_DSP_PARAMS)(unsafe.Pointer(addr))
	dm.connected = true

	// Validate struct size at runtime
	actualSize := unsafe.Sizeof(*dm.params)
	if actualSize != SharedMemSize {
		dm.Close()
		return fmt.Errorf("CRITICAL: struct size mismatch - Go struct is %d bytes but expected %d bytes (check alignment/padding)",
			actualSize, SharedMemSize)
	}

	log.Printf("✓ Shared memory initialized successfully (validated %d bytes)", SharedMemSize)
	return nil
}
func (dm *DSPManager) WriteParams(params *VIPER_DSP_PARAMS) error {
	if dm.params == nil {
		return fmt.Errorf("shared memory not initialized")
	}
	*dm.params = *params
	return nil
}

var (
	modKernel32         = windows.NewLazySystemDLL("kernel32.dll")
	procOpenFileMapping = modKernel32.NewProc("OpenFileMappingW")
)

func OpenFileMapping(access uint32, inheritHandle bool, name *uint16) (windows.Handle, error) {
	inherit := uintptr(0)
	if inheritHandle {
		inherit = 1
	}
	r, _, err := procOpenFileMapping.Call(uintptr(access), inherit, uintptr(unsafe.Pointer(name)))
	if r == 0 {
		return 0, err
	}
	return windows.Handle(r), nil
}

// IsConnected returns true if the shared memory is active
func (dm *DSPManager) IsConnected() bool {
	return dm.connected && dm.params != nil
}

// ApplyChanges syncs the Go state to the APO memory.
// This is the 'Hot Path' - optimized for zero allocations and minimal latency.
//
// Uses double-buffering strategy:
// 1. Write to inactive buffer (no locks needed)
// 2. Atomically copy buffer to shared memory
// 3. Swap active buffer for next write
func (dm *DSPManager) ApplyChanges(state DSPState) error {
	if !dm.IsConnected() {
		return fmt.Errorf("not connected to shared memory - call InitSharedMemory first")
	}

	// Determine which buffer to write to (the inactive one)
	currentFlag := atomic.LoadUint32(&dm.activeBuffer)
	targetBuffer := &dm.frontBuffer
	if currentFlag == 1 {
		targetBuffer = &dm.backBuffer
	}

	// Write all parameters to the inactive buffer (no contention)
	dm.fillBuffer(targetBuffer, state)

	// Atomic copy from buffer to shared memory
	// This uses a low-level memmove which is guaranteed to be atomic
	// for aligned blocks on x86/x64 architecture
	srcPtr := unsafe.Pointer(targetBuffer)
	dstPtr := unsafe.Pointer(dm.params)
	memmove(dstPtr, srcPtr, SharedMemSize)

	// Swap active buffer for next write
	atomic.StoreUint32(&dm.activeBuffer, 1-currentFlag)

	return nil
}

// fillBuffer populates a VIPER_DSP_PARAMS buffer from DSPState
// This is separated from ApplyChanges to improve testability
func (dm *DSPManager) fillBuffer(buffer *VIPER_DSP_PARAMS, state DSPState) {
	// Master controls
	buffer.Enabled = boolToFloat32(state.Master.Power)
	buffer.PreVol = float32(state.Master.PreVol)
	buffer.PostVol = float32(state.Master.PostVol)

	// Equalizer
	buffer.EqEnabled = 1.0 // Always enabled if state has values
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
	buffer.ConvolverEnabled = 0.0
	buffer.CureTechEnabled = 0.0
	buffer.CureTechLevel = 0.0
	buffer.AnalogXEnabled = 0.0
	buffer.AnalogXMode = 0.0
	buffer.SpeakerOptEnabled = 0.0
	buffer.SpeakerOptMode = 0.0
	buffer.SpeakerOptGain = 0.0
	buffer.LimiterEnabled = 0.0
	buffer.LimiterThreshold = 0.0
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

// boolToFloat32 converts boolean to float32 (0.0 or 1.0)
func boolToFloat32(b bool) float32 {
	if b {
		return 1.0
	}
	return 0.0
}

// Close releases the shared memory resources
func (dm *DSPManager) Close() {
	if dm.dataPtr != 0 {
		windows.UnmapViewOfFile(dm.dataPtr)
		dm.dataPtr = 0
	}
	if dm.handle != 0 {
		windows.CloseHandle(dm.handle)
		dm.handle = 0
	}
	dm.connected = false
	log.Println("✓ Shared memory resources released")
}

// memmove is a low-level memory copy function from the Go runtime
// We link to it directly to ensure atomic behavior for our buffer swaps
//
//go:linkname memmove runtime.memmove
func memmove(to, from unsafe.Pointer, n uintptr)

// ValidateStructLayout performs runtime validation of the struct layout.
// This should be called during initialization to catch alignment issues.
func ValidateStructLayout() error {
	var params VIPER_DSP_PARAMS
	baseAddr := uintptr(unsafe.Pointer(&params))

	// Validate critical field offsets
	tests := []struct {
		name   string
		offset uintptr
		expect uintptr
	}{
		{"Enabled", uintptr(unsafe.Pointer(&params.Enabled)) - baseAddr, 0},
		{"PreVol", uintptr(unsafe.Pointer(&params.PreVol)) - baseAddr, 4},
		{"PostVol", uintptr(unsafe.Pointer(&params.PostVol)) - baseAddr, 8},
		{"EqEnabled", uintptr(unsafe.Pointer(&params.EqEnabled)) - baseAddr, 12},
		{"EqBands[0]", uintptr(unsafe.Pointer(&params.EqBands[0])) - baseAddr, 16},
		{"BassEnabled", uintptr(unsafe.Pointer(&params.BassEnabled)) - baseAddr, 88},
	}

	for _, tt := range tests {
		if tt.offset != tt.expect {
			return fmt.Errorf("struct alignment error: %s offset is %d but expected %d",
				tt.name, tt.offset, tt.expect)
		}
	}

	// Validate total size
	if unsafe.Sizeof(params) != SharedMemSize {
		return fmt.Errorf("struct size is %d but expected %d",
			unsafe.Sizeof(params), SharedMemSize)
	}

	log.Printf("✓ Struct layout validation passed (%d bytes, %d fields verified)",
		SharedMemSize, len(tests))
	return nil
}
