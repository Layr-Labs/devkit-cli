package common

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GlobalConfig contains user-level configuration that persists across all devkit usage
type GlobalConfig struct {
	// FirstRun tracks if this is the user's first time running devkit
	FirstRun bool `yaml:"first_run"`
	// TelemetryEnabled stores the user's global telemetry preference
	TelemetryEnabled *bool `yaml:"telemetry_enabled,omitempty"`
	// The users uuid to identify user across projects
	UserUUID string `yaml:"user_uuid"`
}

// GetGlobalConfigDir returns the XDG-compliant directory where global devkit config should be stored
func GetGlobalConfigDir() (string, error) {
	// First check XDG_CONFIG_HOME
	configHome := os.Getenv("XDG_CONFIG_HOME")

	var baseDir string
	if configHome != "" && filepath.IsAbs(configHome) {
		baseDir = configHome
	} else {
		// Fall back to ~/.config
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(baseDir, "devkit"), nil
}

// GetGlobalConfigPath returns the full path to the global config file
func GetGlobalConfigPath() (string, error) {
	configDir, err := GetGlobalConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, GlobalConfigFile), nil
}

// LoadGlobalConfig loads the global configuration, creating defaults if needed
func LoadGlobalConfig() (*GlobalConfig, error) {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		// If we can't determine config path (e.g., no home directory),
		// return first-run defaults so the CLI doesn't fail completely
		return &GlobalConfig{
			FirstRun: true,
		}, nil
	}

	// If file doesn't exist, return defaults for first run
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &GlobalConfig{
			FirstRun: true,
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config GlobalConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveGlobalConfig saves the global configuration to disk
func SaveGlobalConfig(config *GlobalConfig) error {
	configPath, err := GetGlobalConfigPath()
	if err != nil {
		return fmt.Errorf("cannot save global config (unable to determine config directory): %w", err)
	}

	// Ensure directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetGlobalTelemetryPreference returns the global telemetry preference
func GetGlobalTelemetryPreference() (*bool, error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return nil, err
	}
	return config.TelemetryEnabled, nil
}

// SetGlobalTelemetryPreference sets the global telemetry preference
func SetGlobalTelemetryPreference(enabled bool) error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.TelemetryEnabled = &enabled
	config.FirstRun = false // No longer first run after setting preference

	return SaveGlobalConfig(config)
}

// MarkFirstRunComplete marks that the first run has been completed
func MarkFirstRunComplete() error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.FirstRun = false

	return SaveGlobalConfig(config)
}

// IsFirstRun checks if this is the user's first time running devkit
func IsFirstRun() (bool, error) {
	config, err := LoadGlobalConfig()
	if err != nil {
		return false, err
	}
	return config.FirstRun, nil
}
