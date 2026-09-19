package ports

import "viper4windows/internal/domain/models"

// APORegistrationPort manages APO registration with Windows
type APORegistrationPort interface {
	// CheckInstallation verifies if APO is registered
	CheckInstallation() bool

	// Install registers the APO with Windows
	Install(dllPath string) error

	// Uninstall removes the APO from Windows registry
	Uninstall() error

	// GetStatus returns current APO installation status
	GetStatus() models.APOStatus
}
