package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
)

// ═══════════════════════════════════════════════════════════════════════════
// Settings Repository
// ═══════════════════════════════════════════════════════════════════════════
// Persists application settings and last DSP state to a JSON file.

type SettingsRepository struct {
	logger  ports.Logger
	exePath string
}

func NewSettingsRepository(logger ports.Logger) *SettingsRepository {
	exe, _ := os.Executable()
	return &SettingsRepository{
		logger:  logger.WithContext("Settings"),
		exePath: exe,
	}
}

type settingsFile struct {
	LastState      models.DSPState `json:"lastState"`
	Version        string          `json:"version"`
	StartupEnabled bool            `json:"startupEnabled"`
	MinimizeToTray bool            `json:"minimizeToTray"`
	LastPreset     string          `json:"lastPreset"`
}

func (s *SettingsRepository) settingsPath() string {
	dir := filepath.Dir(s.exePath)
	return filepath.Join(dir, "settings.json")
}

// GetLastState retrieves the last saved DSP state, or returns default
func (s *SettingsRepository) GetLastState() (models.DSPState, error) {
	path := s.settingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.logger.Debug("settings file not found, using defaults")
			return models.NewDefaultState(), nil
		}
		return models.DSPState{}, fmt.Errorf("read settings: %w", err)
	}

	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		s.logger.Warn("failed to parse settings, using defaults", "error", err)
		return models.NewDefaultState(), nil
	}

	s.logger.Debug("loaded last state from settings")
	return sf.LastState, nil
}

// SaveLastState persists the current DSP state
func (s *SettingsRepository) SaveLastState(state models.DSPState) error {
	path := s.settingsPath()

	// Load existing settings if present
	sf := settingsFile{LastState: state, Version: "1.0"}
	if existing, err := os.ReadFile(path); err == nil {
		var old settingsFile
		if json.Unmarshal(existing, &old) == nil {
			sf.StartupEnabled = old.StartupEnabled
			sf.MinimizeToTray = old.MinimizeToTray
			sf.LastPreset = old.LastPreset
		}
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	s.logger.Debug("saved last state to settings")
	return nil
}

// GetSetting retrieves a string setting
func (s *SettingsRepository) GetSetting(key string) (string, error) {
	path := s.settingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var sf settingsFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return "", err
	}

	switch key {
	case "startupEnabled":
		if sf.StartupEnabled {
			return "true", nil
		}
		return "false", nil
	case "minimizeToTray":
		if sf.MinimizeToTray {
			return "true", nil
		}
		return "false", nil
	case "lastPreset":
		return sf.LastPreset, nil
	}

	return "", fmt.Errorf("unknown setting: %s", key)
}

// SetSetting stores a string setting
func (s *SettingsRepository) SetSetting(key, value string) error {
	path := s.settingsPath()

	// Load existing
	sf := settingsFile{Version: "1.0"}
	if existing, err := os.ReadFile(path); err == nil {
		json.Unmarshal(existing, &sf)
	}

	switch key {
	case "startupEnabled":
		sf.StartupEnabled = value == "true"
	case "minimizeToTray":
		sf.MinimizeToTray = value == "true"
	case "lastPreset":
		sf.LastPreset = value
	default:
		return fmt.Errorf("unknown setting: %s", key)
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
