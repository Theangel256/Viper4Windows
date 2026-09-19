package windows

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"viper4windows/internal/domain/ports"

	"golang.org/x/sys/windows"
)

// ═══════════════════════════════════════════════════════════════════════════
// DLL Service
// ═══════════════════════════════════════════════════════════════════════════
// Manages direct loading and dispatch to ViPERDSP.dll.
// Ported from original dspmanager.go LoadDLLPath and dispatch functions.

type DLLService struct {
	logger ports.Logger

	mu        sync.Mutex
	dll       windows.Handle
	viperInst uintptr

	fnCreate        uintptr
	fnDestroy       uintptr
	fnSetSampleRate uintptr
	fnReset         uintptr
	fnDispatch      uintptr
	fnProcess       uintptr
	isLoaded        bool
}

func NewDLLService(logger ports.Logger) *DLLService {
	return &DLLService{
		logger: logger.WithContext("DLL"),
	}
}

// LoadDLLPath loads ViPERDSP.dll from the given path
func (s *DLLService) LoadDLLPath(dllPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLoaded {
		return nil
	}

	if !fileExistsAndNotEmpty(dllPath) {
		return fmt.Errorf("ViPERDSP.dll missing or empty: %s", dllPath)
	}

	h, err := windows.LoadLibrary(dllPath)
	if err != nil {
		return fmt.Errorf("LoadLibrary(%s): %w", dllPath, err)
	}

	type symbol struct {
		name string
		dst  *uintptr
	}

	symbols := []symbol{
		{name: "viper_create", dst: &s.fnCreate},
		{name: "viper_destroy", dst: &s.fnDestroy},
		{name: "viper_set_sample_rate", dst: &s.fnSetSampleRate},
		{name: "viper_reset", dst: &s.fnReset},
		{name: "viper_dispatch", dst: &s.fnDispatch},
		{name: "viper_process", dst: &s.fnProcess},
	}

	for _, sym := range symbols {
		addr, symErr := windows.GetProcAddress(h, sym.name)
		if symErr != nil {
			windows.FreeLibrary(h)
			return fmt.Errorf("GetProcAddress(%s): %w", sym.name, symErr)
		}
		*sym.dst = addr
	}

	ret, _, _ := syscall.SyscallN(s.fnCreate)
	if ret == 0 {
		windows.FreeLibrary(h)
		return fmt.Errorf("viper_create() returned NULL")
	}

	s.dll = h
	s.viperInst = ret
	s.callSetSampleRate(44100)
	s.isLoaded = true

	s.logger.Info("DLL loaded", "path", dllPath)
	return nil
}

// Close releases DLL resources
func (s *DLLService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isLoaded {
		return nil
	}

	if s.viperInst != 0 && s.fnDestroy != 0 {
		syscall.SyscallN(s.fnDestroy, s.viperInst)
		s.viperInst = 0
	}

	if s.dll != 0 {
		windows.FreeLibrary(s.dll)
		s.dll = 0
	}

	s.fnCreate = 0
	s.fnDestroy = 0
	s.fnSetSampleRate = 0
	s.fnReset = 0
	s.fnDispatch = 0
	s.fnProcess = 0
	s.isLoaded = false

	s.logger.Info("DLL closed")
	return nil
}

// IsLoaded returns true if DLL is loaded and ready
func (s *DLLService) IsLoaded() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.isLoaded
}

// Dispatch sends a parameter dispatch command
func (s *DLLService) Dispatch(param, val1, val2, val3, val4 int) {
	s.DispatchPayload(param, val1, val2, val3, val4, 0, nil)
}

// DispatchPayload sends a parameter with array payload
func (s *DLLService) DispatchPayload(param, val1, val2, val3, val4, arrSize int, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isLoaded || s.fnDispatch == 0 || s.viperInst == 0 {
		return
	}

	var payloadPtr uintptr
	if len(payload) > 0 {
		payloadPtr = uintptr(unsafe.Pointer(&payload[0]))
	}

	syscall.SyscallN(
		s.fnDispatch,
		s.viperInst,
		uintptr(param),
		uintptr(int32(val1)),
		uintptr(int32(val2)),
		uintptr(int32(val3)),
		uintptr(int32(val4)),
		uintptr(arrSize),
		payloadPtr,
	)
}

// SetSampleRate sets the DSP sample rate
func (s *DLLService) SetSampleRate(rate uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.callSetSampleRate(rate)
}

// Reset resets all DSP effect states
func (s *DLLService) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLoaded && s.fnReset != 0 && s.viperInst != 0 {
		syscall.SyscallN(s.fnReset, s.viperInst)
	}
}

// ── Internal helpers ───────────────────────────────────────────────────────────

func (s *DLLService) callSetSampleRate(rate uint32) {
	if s.fnSetSampleRate == 0 || s.viperInst == 0 {
		return
	}
	syscall.SyscallN(s.fnSetSampleRate, s.viperInst, uintptr(rate))
}
