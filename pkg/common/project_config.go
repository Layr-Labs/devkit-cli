package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ProjectSettings contains the project-level configuration
type ProjectSettings struct {
	ProjectUUID      string `yaml:"project_uuid"`
	TelemetryEnabled bool   `yaml:"telemetry_enabled"`
	PostHogAPIKey    string `yaml:"posthog_api_key,omitempty"`
}

const (
	configFileName = ".config.devkit.yml"
)

// SaveProjectSettings saves project settings to the project directory
func SaveProjectSettings(projectDir string, telemetryEnabled bool) error {
	// Try to load existing settings first to preserve UUID if it exists
	var settings ProjectSettings
	existingSettings, err := LoadProjectSettings()
	if err == nil && existingSettings != nil {
		settings = *existingSettings
		// Only update telemetry setting
		settings.TelemetryEnabled = telemetryEnabled
	} else {
		// Create new settings with a new UUID
		settings = ProjectSettings{
			ProjectUUID:      uuid.New().String(),
			TelemetryEnabled: telemetryEnabled,
		}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	configPath := filepath.Join(projectDir, configFileName)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadProjectSettings loads project settings from the current directory
func LoadProjectSettings() (*ProjectSettings, error) {
	configPath := configFileName

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &settings, nil
}

// IsTelemetryEnabled returns whether telemetry is enabled for the project
// Returns false if config file doesn't exist or telemetry is explicitly disabled
func IsTelemetryEnabled() bool {
	settings, err := LoadProjectSettings()
	if err != nil {
		return false // Config doesn't exist, assume telemetry disabled
	}

	return settings.TelemetryEnabled
}

// GetProjectUUID returns the project UUID or empty string if not found
func GetProjectUUID() string {
	settings, err := LoadProjectSettings()
	if err != nil {
		return ""
	}

	return settings.ProjectUUID
}
