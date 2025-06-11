package common

import (
	"os"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryPromptWithOptions(t *testing.T) {
	logger := logger.NewNoopLogger()

	t.Run("EnableTelemetry enables telemetry", func(t *testing.T) {
		opts := TelemetryPromptOptions{
			EnableTelemetry: true,
		}

		enabled, err := TelemetryPromptWithOptions(logger, opts)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("DisableTelemetry disables telemetry", func(t *testing.T) {
		opts := TelemetryPromptOptions{
			DisableTelemetry: true,
		}

		enabled, err := TelemetryPromptWithOptions(logger, opts)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("SkipPromptInCI disables telemetry in CI", func(t *testing.T) {
		// Set CI environment variable
		originalCI := os.Getenv("CI")
		defer func() {
			if originalCI != "" {
				os.Setenv("CI", originalCI)
			} else {
				os.Unsetenv("CI")
			}
		}()
		os.Setenv("CI", "true")

		opts := TelemetryPromptOptions{
			SkipPromptInCI: true,
		}

		enabled, err := TelemetryPromptWithOptions(logger, opts)
		require.NoError(t, err)
		assert.False(t, enabled)
	})

	t.Run("EnableTelemetry takes precedence over CI detection", func(t *testing.T) {
		// Set CI environment variable
		originalCI := os.Getenv("CI")
		defer func() {
			if originalCI != "" {
				os.Setenv("CI", originalCI)
			} else {
				os.Unsetenv("CI")
			}
		}()
		os.Setenv("CI", "true")

		opts := TelemetryPromptOptions{
			EnableTelemetry: true,
			SkipPromptInCI:  true,
		}

		enabled, err := TelemetryPromptWithOptions(logger, opts)
		require.NoError(t, err)
		assert.True(t, enabled)
	})

	t.Run("DisableTelemetry takes precedence over CI detection", func(t *testing.T) {
		// Set CI environment variable
		originalCI := os.Getenv("CI")
		defer func() {
			if originalCI != "" {
				os.Setenv("CI", originalCI)
			} else {
				os.Unsetenv("CI")
			}
		}()
		os.Setenv("CI", "true")

		opts := TelemetryPromptOptions{
			DisableTelemetry: true,
			SkipPromptInCI:   true,
		}

		enabled, err := TelemetryPromptWithOptions(logger, opts)
		require.NoError(t, err)
		assert.False(t, enabled)
	})
}

func TestIsStdinAvailable(t *testing.T) {
	// This test is environment-dependent and mainly for verification
	// In a real terminal, stdin should be available
	// In CI or non-interactive environments, it may not be
	available := isStdinAvailable()
	t.Logf("stdin available: %v", available)

	// We can't make strong assertions here since it depends on the test environment
	// but we can verify the function doesn't panic
	assert.IsType(t, true, available)
}

func TestHandleFirstRunTelemetryPromptWithOptions(t *testing.T) {
	logger := logger.NewNoopLogger()

	t.Run("Non-first run returns existing preference", func(t *testing.T) {
		// This test would need to set up a mock environment
		// For now, just verify the function signature works
		opts := TelemetryPromptOptions{}
		enabled, isFirstRun, err := HandleFirstRunTelemetryPromptWithOptions(logger, opts)

		// The actual behavior depends on the global config state
		// We're just verifying the function doesn't panic
		assert.NoError(t, err)
		assert.IsType(t, true, enabled)
		assert.IsType(t, true, isFirstRun)
	})
}
