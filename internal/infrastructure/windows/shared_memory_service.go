package windows

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"

	"golang.org/x/sys/windows"
)

// ═══════════════════════════════════════════════════════════════════════════
// Shared Memory Service
// ═══════════════════════════════════════════════════════════════════════════
// Manages inter-process communication with the ViPER APO via shared memory.
// Based on the reference Flutter implementation's SharedMemoryService.

const (
	pageReadWrite    = 0x04
	fileMapReadWrite = 0x0006 // FILE_MAP_READ | FILE_MAP_WRITE
	invalidHandle    = 0
)

type SharedMemoryService struct {
	// Dependencies (injected)
	logger   ports.Logger
	security ports.SecurityPort
	paramSvc ports.DSPParameterPort

	// Windows API functions
	kernel32          *windows.DLL
	createFileMapping *windows.Proc
	mapViewOfFile     *windows.Proc
	unmapViewOfFile   *windows.Proc
	closeHandle       *windows.Proc
	createEvent       *windows.Proc
	setEvent          *windows.Proc

	// Shared memory handles
	hMap   windows.Handle
	pView  uintptr
	hEvent windows.Handle

	// Thread safety
	mu     sync.RWMutex
	isOpen bool
}

// NewSharedMemoryService creates a new shared memory service instance
func NewSharedMemoryService(
	logger ports.Logger,
	security ports.SecurityPort,
	paramSvc ports.DSPParameterPort,
) *SharedMemoryService {
	return &SharedMemoryService{
		logger:   logger.WithContext("SharedMemory"),
		security: security,
		paramSvc: paramSvc,
	}
}

// Open initializes the shared memory region and event
func (s *SharedMemoryService) Open() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isOpen {
		return nil // Already open
	}

	// Load kernel32.dll
	if err := s.loadWindowsAPIs(); err != nil {
		return fmt.Errorf("load Windows APIs: %w", err)
	}

	// Build security attributes for cross-session IPC
	sa, err := s.security.BuildSecurityAttributes()
	if err != nil {
		s.logger.Warn("failed to build security attributes", "error", err)
		sa = nil // Continue without security descriptor
	}

	// Create shared memory mapping
	if err := s.createMapping(sa); err != nil {
		return fmt.Errorf("create mapping: %w", err)
	}

	// Map view of file
	if err := s.mapView(); err != nil {
		s.closeHandles()
		return fmt.Errorf("map view: %w", err)
	}

	// Create synchronization event
	if err := s.createSyncEvent(sa); err != nil {
		s.closeHandles()
		return fmt.Errorf("create event: %w", err)
	}

	s.isOpen = true
	s.logger.Info("shared memory opened",
		"name", models.SharedMemName,
		"size", models.SharedMemBytes)

	return nil
}

// Close releases all shared memory resources
func (s *SharedMemoryService) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isOpen {
		return nil
	}

	s.closeHandles()
	s.isOpen = false
	s.logger.Info("shared memory closed")

	return nil
}

// WriteParams encodes and writes DSP state to shared memory
func (s *SharedMemoryService) WriteParams(state models.DSPState) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return fmt.Errorf("shared memory not open")
	}

	// Encode state to binary format
	data, err := s.paramSvc.EncodeState(state)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}

	// Write to shared memory
	if err := s.writeBytes(data); err != nil {
		return fmt.Errorf("write bytes: %w", err)
	}

	// Signal the APO of changes
	if err := s.Signal(); err != nil {
		s.logger.Warn("failed to signal APO", "error", err)
	}

	return nil
}

// ReadAPOStatus reads status information written by the APO
func (s *SharedMemoryService) ReadAPOStatus() (models.AudioEngineStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := models.AudioEngineStatus{}

	if !s.isOpen {
		return status, fmt.Errorf("shared memory not open")
	}

	// Read status from shared memory (implementation would read specific offsets)
	// This is a placeholder - real implementation would parse binary data
	s.logger.Debug("reading APO status from shared memory")

	return status, nil
}

// IsConnected checks if shared memory is accessible
func (s *SharedMemoryService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.isOpen && s.pView != 0
}

// Signal notifies the APO that parameters have changed
func (s *SharedMemoryService) Signal() error {
	if s.hEvent == 0 {
		return fmt.Errorf("event handle not initialized")
	}

	ret, _, err := s.setEvent.Call(uintptr(s.hEvent))
	if ret == 0 {
		return fmt.Errorf("SetEvent failed: %w", err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Private Methods
// ═══════════════════════════════════════════════════════════════════════════

func (s *SharedMemoryService) loadWindowsAPIs() error {
	var err error
	s.kernel32, err = windows.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}

	s.createFileMapping, err = s.kernel32.FindProc("CreateFileMappingW")
	if err != nil {
		return err
	}

	s.mapViewOfFile, err = s.kernel32.FindProc("MapViewOfFile")
	if err != nil {
		return err
	}

	s.unmapViewOfFile, err = s.kernel32.FindProc("UnmapViewOfFile")
	if err != nil {
		return err
	}

	s.closeHandle, err = s.kernel32.FindProc("CloseHandle")
	if err != nil {
		return err
	}

	s.createEvent, err = s.kernel32.FindProc("CreateEventW")
	if err != nil {
		return err
	}

	s.setEvent, err = s.kernel32.FindProc("SetEvent")
	if err != nil {
		return err
	}

	return nil
}

func (s *SharedMemoryService) createMapping(sa interface{}) error {
	namePtr, err := syscall.UTF16PtrFromString(models.SharedMemName)
	if err != nil {
		return err
	}

	ret, _, err := s.createFileMapping.Call(
		uintptr(invalidHandle),
		uintptr(0), // Security attributes
		pageReadWrite,
		0,
		uintptr(models.SharedMemBytes),
		uintptr(unsafe.Pointer(namePtr)),
	)

	if ret == 0 {
		return fmt.Errorf("CreateFileMappingW failed: %w", err)
	}

	s.hMap = windows.Handle(ret)
	return nil
}

func (s *SharedMemoryService) mapView() error {
	ret, _, err := s.mapViewOfFile.Call(
		uintptr(s.hMap),
		fileMapReadWrite,
		0,
		0,
		uintptr(models.SharedMemBytes),
	)

	if ret == 0 {
		return fmt.Errorf("MapViewOfFile failed: %w", err)
	}

	s.pView = ret
	return nil
}

func (s *SharedMemoryService) createSyncEvent(sa interface{}) error {
	namePtr, err := syscall.UTF16PtrFromString(models.SharedEventName)
	if err != nil {
		return err
	}

	ret, _, err := s.createEvent.Call(
		uintptr(0), // Security attributes
		0,          // Manual reset
		0,          // Initial state
		uintptr(unsafe.Pointer(namePtr)),
	)

	if ret == 0 {
		return fmt.Errorf("CreateEventW failed: %w", err)
	}

	s.hEvent = windows.Handle(ret)
	return nil
}

func (s *SharedMemoryService) writeBytes(data []byte) error {
	if s.pView == 0 {
		return fmt.Errorf("mapped view not available")
	}

	// Copy data to shared memory
	dst := (*[models.SharedMemBytes]byte)(unsafe.Pointer(s.pView))
	copy(dst[:], data)

	return nil
}

// IsReady checks if shared memory is open and connected
func (s *SharedMemoryService) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isOpen && s.pView != 0 && s.hEvent != 0
}

// Write writes raw bytes to shared memory and signals the APO
func (s *SharedMemoryService) Write(data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isOpen {
		return fmt.Errorf("shared memory not open")
	}

	if err := s.writeBytes(data); err != nil {
		return fmt.Errorf("write bytes: %w", err)
	}

	// Signal the APO of changes
	if err := s.Signal(); err != nil {
		s.logger.Warn("failed to signal APO", "error", err)
	}

	return nil
}

func (s *SharedMemoryService) closeHandles() {
	if s.pView != 0 {
		s.unmapViewOfFile.Call(s.pView)
		s.pView = 0
	}

	if s.hMap != 0 {
		s.closeHandle.Call(uintptr(s.hMap))
		s.hMap = 0
	}

	if s.hEvent != 0 {
		s.closeHandle.Call(uintptr(s.hEvent))
		s.hEvent = 0
	}
}
