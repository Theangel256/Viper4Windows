package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
)

// ═══════════════════════════════════════════════════════════════════════════
// Preset Manager Service
// ═══════════════════════════════════════════════════════════════════════════
// Manages DSP preset storage and retrieval from the filesystem

type PresetManagerService struct {
	logger     ports.Logger
	paramSvc   ports.DSPParameterPort
	presetsDir string
}

// NewPresetManagerService creates a new preset manager
func NewPresetManagerService(
	logger ports.Logger,
	paramSvc ports.DSPParameterPort,
	presetsDir string,
) *PresetManagerService {
	// Ensure presets directory exists
	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		logger.Warn("failed to create presets directory", "error", err, "path", presetsDir)
	}

	return &PresetManagerService{
		logger:     logger.WithContext("PresetManager"),
		paramSvc:   paramSvc,
		presetsDir: presetsDir,
	}
}

// Save persists a preset to disk
func (p *PresetManagerService) Save(name string, state models.DSPState) error {
	// Sanitize filename
	safeName := sanitizeFilename(name)
	if safeName == "" {
		return fmt.Errorf("invalid preset name: %s", name)
	}

	// Normalize state before saving
	state = p.paramSvc.NormalizeState(state)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal preset: %w", err)
	}

	// Write to file
	path := filepath.Join(p.presetsDir, safeName+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write preset file: %w", err)
	}

	p.logger.Info("preset saved", "name", name, "path", path)
	return nil
}

// Load retrieves a preset from disk
func (p *PresetManagerService) Load(name string) (models.DSPState, error) {
	safeName := sanitizeFilename(name)
	path := filepath.Join(p.presetsDir, safeName+".json")

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return models.DSPState{}, fmt.Errorf("read preset: %w", err)
	}

	// Unmarshal JSON
	var state models.DSPState
	if err := json.Unmarshal(data, &state); err != nil {
		return models.DSPState{}, fmt.Errorf("unmarshal preset: %w", err)
	}

	// Normalize loaded state
	state = p.paramSvc.NormalizeState(state)

	p.logger.Info("preset loaded", "name", name)
	return state, nil
}

// List returns all available preset names
func (p *PresetManagerService) List() ([]string, error) {
	entries, err := os.ReadDir(p.presetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read presets directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		// Remove .json extension
		presetName := name[:len(name)-5]
		names = append(names, presetName)
	}

	return names, nil
}

// Delete removes a preset from disk
func (p *PresetManagerService) Delete(name string) error {
	safeName := sanitizeFilename(name)
	path := filepath.Join(p.presetsDir, safeName+".json")

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete preset: %w", err)
	}

	p.logger.Info("preset deleted", "name", name)
	return nil
}

// Exists checks if a preset exists
func (p *PresetManagerService) Exists(name string) bool {
	safeName := sanitizeFilename(name)
	path := filepath.Join(p.presetsDir, safeName+".json")

	_, err := os.Stat(path)
	return err == nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Private Helpers
// ═══════════════════════════════════════════════════════════════════════════

// sanitizeFilename removes dangerous characters from preset names
func sanitizeFilename(name string) string {
	// Remove path separators and other dangerous characters
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, ":", "")
	name = strings.ReplaceAll(name, "*", "")
	name = strings.ReplaceAll(name, "?", "")
	name = strings.ReplaceAll(name, "\"", "")
	name = strings.ReplaceAll(name, "<", "")
	name = strings.ReplaceAll(name, ">", "")
	name = strings.ReplaceAll(name, "|", "")

	return strings.TrimSpace(name)
}
