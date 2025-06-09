package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalConfig(t *testing.T) {
	t.Run("LoadGlobalConfig_FirstTime", func(t *testing.T) {
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

		config, err := LoadGlobalConfig()
		require.NoError(t, err)
		assert.True(t, config.FirstRun)
		assert.Nil(t, config.TelemetryEnabled)
	})

	t.Run("SaveAndLoadGlobalConfig", func(t *testing.T) {
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

		// Save a config
		config := &GlobalConfig{
			FirstRun:         false,
			TelemetryEnabled: boolPtr(true),
		}

		err := SaveGlobalConfig(config)
		require.NoError(t, err)

		// Load it back
		loadedConfig, err := LoadGlobalConfig()
		require.NoError(t, err)

		assert.False(t, loadedConfig.FirstRun)
		require.NotNil(t, loadedConfig.TelemetryEnabled)
		assert.True(t, *loadedConfig.TelemetryEnabled)
	})

	t.Run("SetGlobalTelemetryPreference", func(t *testing.T) {
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

	t.Run("IsFirstRun", func(t *testing.T) {
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

		// Should be true initially
		firstRun, err := IsFirstRun()
		require.NoError(t, err)
		assert.True(t, firstRun)

		// Mark as complete
		err = MarkFirstRunComplete()
		require.NoError(t, err)

		// Should be false now
		firstRun, err = IsFirstRun()
		require.NoError(t, err)
		assert.False(t, firstRun)
	})

	t.Run("GetGlobalConfigDir_XDG", func(t *testing.T) {
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

		configDir, err := GetGlobalConfigDir()
		require.NoError(t, err)

		expected := filepath.Join(tmpDir, "devkit")
		assert.Equal(t, expected, configDir)
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
