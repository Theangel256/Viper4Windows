package ports

import "viper4windows/internal/domain/models"

// SharedMemoryPort manages communication with the APO via shared memory
type SharedMemoryPort interface {
	// Open initializes the shared memory region
	Open() error

	// Close releases shared memory resources
	Close() error

	// WriteParams writes DSP parameters to shared memory
	WriteParams(state models.DSPState) error

	// Write writes raw bytes to shared memory
	Write(data []byte) error

	// ReadAPOStatus reads status information from the APO
	ReadAPOStatus() (models.AudioEngineStatus, error)

	// IsConnected checks if shared memory is accessible
	IsConnected() bool

	// IsReady checks if shared memory is open and connected
	IsReady() bool

	// Signal notifies the APO of parameter changes
	Signal() error
}
