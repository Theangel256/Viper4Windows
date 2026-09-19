package ports

import "viper4windows/internal/domain/models"

// DeviceManagementPort handles audio device detection and configuration
type DeviceManagementPort interface {
	// EnumerateDevices lists all audio endpoints
	EnumerateDevices(role models.DeviceRole) ([]models.AudioDevice, error)

	// GetDefaultDevice returns the system's default audio device
	GetDefaultDevice(role models.DeviceRole) (*models.AudioDevice, error)

	// AttachAPO attaches the APO to a specific device
	AttachAPO(deviceID string) error

	// DetachAPO removes the APO from a device
	DetachAPO(deviceID string) error

	// IsAPOAttached checks if APO is attached to a device
	IsAPOAttached(deviceID string) bool
}
