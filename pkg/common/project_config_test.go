package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadProjectSettings(t *testing.T) {
	// Create temp dir for test
	tmpDir := t.TempDir()

	// Test saving project settings
	err := SaveTelemetrySetting(tmpDir, true)
	if err != nil {
		t.Fatalf("Failed to save project settings: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, configFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created")
	}

	// Set current directory to temp dir to test loading
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("Failed to restore original directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Test loading project settings
	settings, err := LoadProjectSettings()
	if err != nil {
		t.Fatalf("Failed to load project settings: %v", err)
	}

	// Verify settings content
	if settings.ProjectUUID == "" {
		t.Error("ProjectUUID is empty")
	}

	if !settings.TelemetryEnabled {
		t.Error("TelemetryEnabled should be true")
	}
}
