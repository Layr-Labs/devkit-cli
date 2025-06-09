package common

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalSettings contains the user-level configuration
type GlobalSettings struct {
	UserUUID string `yaml:"user_uuid"`
}

// SaveUserId saves user settings to the global config, but preserves existing UUID if present
func SaveUserId(userUuid string) error {
	// Try to load existing settings first to preserve UUID if it exists
	var settings GlobalSettings
	existingSettings, err := LoadGlobalSettings()
	if err == nil && existingSettings != nil {
		settings = *existingSettings
	} else {
		// Create new settings with provided UUID
		settings = GlobalSettings{UserUUID: userUuid}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	globalConfigDir := filepath.Join(os.Getenv("HOME"), ".devkit")
	// Create global .devkit directory
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	globalConfigPath := filepath.Join(globalConfigDir, GlobalConfigFile)
	if err := os.WriteFile(globalConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func loadGlobalSettingsFromLocation(location string) (*GlobalSettings, error) {
	data, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var settings GlobalSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &settings, nil
}

// LoadGlobalSettings loads users settings from the home directory
func LoadGlobalSettings() (*GlobalSettings, error) {
	globalConfigPath := filepath.Join(os.Getenv("HOME"), ".devkit", GlobalConfigFile)

	return loadGlobalSettingsFromLocation(globalConfigPath)
}

func getUserUUIDFromGlobalSettings() string {
	settings, err := LoadGlobalSettings()
	if err != nil {
		return ""
	}

	return settings.UserUUID
}
