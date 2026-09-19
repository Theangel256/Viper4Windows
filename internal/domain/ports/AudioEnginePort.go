package ports

import "viper4windows/internal/domain/models"

// AudioEnginePort controls the Windows Audio Engine
type AudioEnginePort interface {
	// Restart restarts the audio service to apply changes
	Restart() error

	// GetStatus returns current engine status
	GetStatus() (models.AudioEngineStatus, error)

	// IsRunning checks if the audio service is active
	IsRunning() bool
}
