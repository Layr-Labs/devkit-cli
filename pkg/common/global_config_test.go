package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalConfig(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Override XDG_CONFIG_HOME for testing
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	t.Run("LoadGlobalConfig_FirstTime", func(t *testing.T) {
		config, err := LoadGlobalConfig()
		require.NoError(t, err)
		assert.True(t, config.FirstRun)
		assert.Nil(t, config.TelemetryEnabled)
	})

	t.Run("SaveAndLoadGlobalConfig", func(t *testing.T) {
		config := &GlobalConfig{
			FirstRun:         false,
			TelemetryEnabled: boolPtr(true),
		}

		err := SaveGlobalConfig(config)
		require.NoError(t, err)

		loadedConfig, err := LoadGlobalConfig()
		require.NoError(t, err)
		assert.False(t, loadedConfig.FirstRun)
		assert.NotNil(t, loadedConfig.TelemetryEnabled)
		assert.True(t, *loadedConfig.TelemetryEnabled)
	})

	t.Run("IsFirstRun", func(t *testing.T) {
		// First time should be true
		isFirst, err := IsFirstRun()
		require.NoError(t, err)
		assert.True(t, isFirst)

		// After marking complete, should be false
		err = MarkFirstRunComplete()
		require.NoError(t, err)

		isFirst, err = IsFirstRun()
		require.NoError(t, err)
		assert.False(t, isFirst)
	})

	t.Run("TelemetryPreferences", func(t *testing.T) {
		// Initially should be nil
		pref, err := GetGlobalTelemetryPreference()
		require.NoError(t, err)
		assert.Nil(t, pref)

		// Set to true
		err = SetGlobalTelemetryPreference(true)
		require.NoError(t, err)

		pref, err = GetGlobalTelemetryPreference()
		require.NoError(t, err)
		require.NotNil(t, pref)
		assert.True(t, *pref)

		// Set to false
		err = SetGlobalTelemetryPreference(false)
		require.NoError(t, err)

		pref, err = GetGlobalTelemetryPreference()
		require.NoError(t, err)
		require.NotNil(t, pref)
		assert.False(t, *pref)
	})

	t.Run("GetGlobalConfigPath", func(t *testing.T) {
		configPath, err := GetGlobalConfigPath()
		require.NoError(t, err)
		assert.Contains(t, configPath, "devkit")
		assert.Contains(t, configPath, "config.yaml")
		assert.Contains(t, configPath, tmpDir)
	})
}

func TestGlobalConfigWithHomeDir(t *testing.T) {
	// Test fallback to home directory when XDG_CONFIG_HOME is not set
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	os.Unsetenv("XDG_CONFIG_HOME")

	configDir, err := GetGlobalConfigDir()
	require.NoError(t, err)

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	expectedDir := filepath.Join(homeDir, ".config", "devkit")
	assert.Equal(t, expectedDir, configDir)
}

// Helper function to create a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}
