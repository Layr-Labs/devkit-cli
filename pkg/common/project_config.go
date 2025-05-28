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

// CreateProjectWithGlobalTelemetryDefaults creates a new project config using global telemetry preference
func CreateProjectWithGlobalTelemetryDefaults(projectDir string, projectUuid string) error {
	// Get global telemetry preference
	globalTelemetryEnabled, err := GetGlobalTelemetryPreference()
	if err != nil {
		// If we can't get global preference, default to false for safety
		return SaveProjectIdAndTelemetryToggle(projectDir, projectUuid, false)
	}

	// Use global preference if set, otherwise default to false
	telemetryEnabled := false
	if globalTelemetryEnabled != nil {
		telemetryEnabled = *globalTelemetryEnabled
	}

	return SaveProjectIdAndTelemetryToggle(projectDir, projectUuid, telemetryEnabled)
}

// SetProjectTelemetry sets telemetry preference for the current project only
func SetProjectTelemetry(enabled bool) error {
	// Find project directory by looking for .config.devkit.yml
	projectDir, err := FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in a devkit project directory: %w", err)
	}

	// Load existing settings to preserve other values
	settings, err := LoadProjectSettings()
	if err != nil {
		return fmt.Errorf("failed to load project settings: %w", err)
	}

	if settings == nil {
		return fmt.Errorf("no project configuration found")
	}

	// Update only telemetry setting
	settings.TelemetryEnabled = enabled

	// Save back to project
	return SaveProjectIdAndTelemetryToggle(projectDir, settings.ProjectUUID, enabled)
}

// FindProjectRoot searches upward from current directory to find .config.devkit.yml
func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Search upward for .config.devkit.yml
	for {
		configPath := filepath.Join(currentDir, ".config.devkit.yml")
		if _, err := os.Stat(configPath); err == nil {
			return currentDir, nil
		}

		parent := filepath.Dir(currentDir)
		// Reached filesystem root
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("not in a devkit project (no .config.devkit.yml found)")
}

// GetEffectiveTelemetryPreference returns the effective telemetry preference
// Project setting takes precedence over global setting
func GetEffectiveTelemetryPreference() (bool, error) {
	// First try to get project-specific setting
	projectSettings, err := LoadProjectSettings()
	if err == nil && projectSettings != nil {
		return projectSettings.TelemetryEnabled, nil
	}

	// Fall back to global setting
	globalPreference, err := GetGlobalTelemetryPreference()
	if err != nil {
		return false, err
	}

	// If no global preference set, default to false
	if globalPreference == nil {
		return false, nil
	}

	return *globalPreference, nil
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
// It checks both global and project-level preferences, with global taking precedence
func IsTelemetryEnabled() bool {
	return isTelemetryEnabledAtPath(DevkitConfigFile)
}

func isTelemetryEnabledAtPath(location string) bool {
	// First check global preference - this takes precedence
	globalPref, err := GetGlobalTelemetryPreference()
	if err == nil && globalPref != nil {
		return *globalPref
	}

	// Fallback to project-level preference
	settings, err := loadProjectSettingsFromLocation(location)
	if err != nil {
		return false // Config doesn't exist, assume telemetry disabled
	}

	return settings.TelemetryEnabled
}
