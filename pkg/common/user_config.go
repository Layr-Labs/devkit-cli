package common

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SaveUserId saves user settings to the global config, but preserves existing UUID if present
func SaveUserId(userUuid string) error {
	// Try to load existing settings first to preserve UUID if it exists
	var settings GlobalConfig
	existingSettings, err := LoadGlobalConfig()
	if err == nil && existingSettings != nil {
		settings = *existingSettings
		if settings.UserUUID == "" {
			settings.UserUUID = userUuid
		}
	} else {
		// Create new settings with provided UUID
		settings = GlobalConfig{
			FirstRun: true,
			UserUUID: userUuid,
		}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Get the global config dir so that we can create it
	globalConfigDir, err := GetGlobalConfigDir()
	if err != nil {
		return err
	}

	// Create global dir
	if err := os.MkdirAll(globalConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	globalConfigPath := filepath.Join(globalConfigDir, GlobalConfigFile)
	if err := os.WriteFile(globalConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func getUserUUIDFromGlobalConfig() string {
	config, err := LoadGlobalConfig()
	if err != nil {
		return ""
	}

	return config.UserUUID
}
