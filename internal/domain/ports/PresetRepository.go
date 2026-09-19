package ports

import "viper4windows/internal/domain/models"

// PresetRepository manages preset storage and retrieval
type PresetRepository interface {
	// Save persists a preset to storage
	Save(name string, state models.DSPState) error

	// Load retrieves a preset by name
	Load(name string) (models.DSPState, error)

	// List returns all available preset names
	List() ([]string, error)

	// Delete removes a preset
	Delete(name string) error

	// Exists checks if a preset exists
	Exists(name string) bool
}
