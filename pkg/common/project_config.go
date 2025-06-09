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

// SaveProjectIdAndTelemetryToggle saves project settings to config.yaml
func SaveProjectIdAndTelemetryToggle(projectDir string, projectUuid string, telemetryEnabled bool) error {
	configPath := filepath.Join(projectDir, "config", "config.yaml")

	// Load existing config.yaml
	config, err := loadConfigFromPath(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.yaml: %w", err)
	}

	// Update project fields
	config.Config.Project.ProjectUUID = projectUuid
	config.Config.Project.TelemetryEnabled = telemetryEnabled

	// Save back to config.yaml
	return saveConfigToPath(configPath, config)
}

// SetProjectTelemetry sets telemetry preference for the current project only
func SetProjectTelemetry(enabled bool) error {
	// Find project directory by looking for config/config.yaml
	projectDir, err := FindProjectRoot()
	if err != nil {
		return fmt.Errorf("not in a devkit project directory: %w", err)
	}

	configPath := filepath.Join(projectDir, "config", "config.yaml")

	// Load existing config.yaml
	config, err := loadConfigFromPath(configPath)
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	// Update only telemetry setting
	config.Config.Project.TelemetryEnabled = enabled

	// Save back to config.yaml
	return saveConfigToPath(configPath, config)
}

// FindProjectRoot searches upward from current directory to find config/config.yaml
func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Search upward for config/config.yaml
	for {
		configPath := filepath.Join(currentDir, "config", "config.yaml")
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

	return "", fmt.Errorf("not in a devkit project (no config/config.yaml found)")
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

// loadConfigFromPath loads the complete config.yaml structure
func loadConfigFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// saveConfigToPath saves the complete config.yaml structure
func saveConfigToPath(configPath string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadProjectSettings loads project settings from config.yaml
func LoadProjectSettings() (*ProjectSettings, error) {
	configPath := filepath.Join("config", "config.yaml")
	config, err := loadConfigFromPath(configPath)
	if err != nil {
		return nil, err
	}

	return &ProjectSettings{
		ProjectUUID:      config.Config.Project.ProjectUUID,
		TelemetryEnabled: config.Config.Project.TelemetryEnabled,
	}, nil
}

// GetProjectUUID returns the project UUID from config.yaml or empty string if not found
func GetProjectUUID() string {
	settings, err := LoadProjectSettings()
	if err != nil {
		return ""
	}
	return settings.ProjectUUID
}

// IsTelemetryEnabled returns whether telemetry is enabled for the project
// It checks both global and project-level preferences, with project taking precedence
func IsTelemetryEnabled() bool {
	// Use the effective preference which handles precedence correctly
	enabled, err := GetEffectiveTelemetryPreference()
	if err != nil {
		return false
	}
	return enabled
}

// Helper functions for testing
func isTelemetryEnabledAtPath(location string) bool {
	// For testing: load config.yaml at specific path and check telemetry setting
	config, err := loadConfigFromPath(location)
	if err != nil {
		return false // Config doesn't exist, assume telemetry disabled
	}

	return config.Config.Project.TelemetryEnabled
}

func getProjectUUIDFromLocation(location string) string {
	config, err := loadConfigFromPath(location)
	if err != nil {
		return ""
	}
	return config.Config.Project.ProjectUUID
}
