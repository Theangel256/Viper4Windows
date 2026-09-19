package ports

import "viper4windows/internal/domain/models"

// DSPParameterPort handles DSP parameter encoding and validation
type DSPParameterPort interface {
	// EncodeState converts DSPState to the binary format expected by APO
	EncodeState(state models.DSPState) ([]byte, error)

	// ValidateState ensures all parameters are within valid ranges
	ValidateState(state *models.DSPState) error

	// NormalizeState clamps all parameters to valid ranges
	NormalizeState(state models.DSPState) models.DSPState
}
