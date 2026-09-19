package ports

import "viper4windows/internal/domain/models"

// SettingsRepository manages application settings
type SettingsRepository interface {
	// GetLastState retrieves the last saved DSP state
	GetLastState() (models.DSPState, error)

	// SaveLastState persists current state
	SaveLastState(state models.DSPState) error

	// GetSetting retrieves a setting value
	GetSetting(key string) (string, error)

	// SetSetting stores a setting value
	SetSetting(key, value string) error
}
