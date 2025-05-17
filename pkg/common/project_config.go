package common

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectSettings contains the project-level configuration
type ProjectSettings struct {
	ProjectUUID      string `yaml:"project_uuid"`
	TelemetryEnabled bool   `yaml:"telemetry_enabled"`
}

// SaveProjectIdAndTelemetryToggle saves project settings to the project directory
func SaveProjectIdAndTelemetryToggle(projectDir string, projectUuid string, telemetryEnabled bool) error {
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
			ProjectUUID:      projectUuid,
			TelemetryEnabled: telemetryEnabled,
		}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	configPath := filepath.Join(projectDir, DevkitConfigFile)
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func loadProjectSettingsFromLocation(location string) (*ProjectSettings, error) {
	data, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &settings, nil
}

// LoadProjectSettings loads project settings from the current directory
func LoadProjectSettings() (*ProjectSettings, error) {
	return loadProjectSettingsFromLocation(DevkitConfigFile)
}

func getProjectUUIDFromLocation(location string) string {
	settings, err := loadProjectSettingsFromLocation(location)
	if err != nil {
		return ""
	}

	return settings.ProjectUUID
}

// GetProjectUUID returns the project UUID or empty string if not found
func GetProjectUUID() string {
	return getProjectUUIDFromLocation(DevkitConfigFile)
}

// IsTelemetryEnabled returns whether telemetry is enabled for the project
// Returns false if config file doesn't exist or telemetry is explicitly disabled
// TODO: (brandon c) currently unused -- update to use after private preview
func IsTelemetryEnabled() bool {
	return isTelemetryEnabled(DevkitConfigFile)
}

func isTelemetryEnabled(location string) bool {
	settings, err := loadProjectSettingsFromLocation(location)
	if err != nil {
		return false // Config doesn't exist, assume telemetry disabled
	}

	return settings.TelemetryEnabled
}
