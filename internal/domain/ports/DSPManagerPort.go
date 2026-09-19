package ports

import "viper4windows/internal/domain/models"

// DSPManagerPort coordinates DSP control (DLL and/or shared memory paths)
type DSPManagerPort interface {
	// ApplyChanges applies DSP state via all available paths (DLL + SHM)
	ApplyChanges(state models.DSPState) error

	// IsDLLReady returns true if DLL path is available
	IsDLLReady() bool

	// IsSHMReady returns true if shared memory path is available
	IsSHMReady() bool
}
