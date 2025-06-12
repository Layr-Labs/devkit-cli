package common

import (
	"encoding/json"
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

// GetEffectiveTelemetryPreference returns the effective telemetry preference
// Project setting takes precedence over global setting
func GetEffectiveTelemetryPreference() (bool, error) {
	// Try to load project settings first
	projectSettings, err := LoadProjectSettings()
	if err == nil && projectSettings != nil {
		return projectSettings.TelemetryEnabled, nil
	}

	// Fall back to global preference
	globalPref, err := GetGlobalTelemetryPreference()
	if err != nil {
		return false, err
	}

	if globalPref == nil {
		return false, nil // Default to disabled
	}

	return *globalPref, nil
}

// GetGlobalTelemetryPreference returns the global telemetry preference
func GetGlobalTelemetryPreference() (*bool, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	globalConfigPath := filepath.Join(homeDir, ".devkit", "config.json")

	// Check if file exists
	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		return nil, nil // No global preference set
	}

	data, err := os.ReadFile(globalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var config struct {
		TelemetryEnabled *bool `json:"telemetry_enabled"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	return config.TelemetryEnabled, nil
}

// SetGlobalTelemetryPreference sets the global telemetry preference
func SetGlobalTelemetryPreference(enabled bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".devkit")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	globalConfigPath := filepath.Join(configDir, "config.json")

	// Load existing config or create new one
	var config struct {
		TelemetryEnabled *bool `json:"telemetry_enabled"`
	}

	if data, err := os.ReadFile(globalConfigPath); err == nil {
		json.Unmarshal(data, &config) // Ignore errors, just overwrite
	}

	config.TelemetryEnabled = &enabled

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal global config: %w", err)
	}

	if err := os.WriteFile(globalConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	return nil
}

// SaveProjectSettings saves project settings to the current directory
func SaveProjectSettings(settings *ProjectSettings) error {
	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	if err := os.WriteFile(DevkitConfigFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetProjectTelemetry sets the telemetry preference for the current project
func SetProjectTelemetry(enabled bool) error {
	settings, err := LoadProjectSettings()
	if err != nil {
		// Create new settings if they don't exist
		settings = &ProjectSettings{
			TelemetryEnabled: enabled,
		}
	} else {
		settings.TelemetryEnabled = enabled
	}

	return SaveProjectSettings(settings)
}
