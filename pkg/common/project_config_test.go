package common

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func TestSaveAndLoadProjectSettings(t *testing.T) {
	// Create temp dir for test
	tmpDir := t.TempDir()

	// Test saving project settings
	err := SaveProjectIdAndTelemetryToggle(tmpDir, uuid.New().String(), true)
	if err != nil {
		t.Fatalf("Failed to save project settings: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, DevkitConfigFile)
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

func TestGetProjectUUIDFromLocation_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	expectedUUID := uuid.New().String()

	content := []byte("project_uuid: " + expectedUUID + "\ntelemetry_enabled: true\n")
	configPath := filepath.Join(tmpDir, "devkit.yaml")
	err := os.WriteFile(configPath, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	actual := getProjectUUIDFromLocation(configPath)
	if actual != expectedUUID {
		t.Errorf("Expected UUID %s, got %s", expectedUUID, actual)
	}
}

func TestGetProjectUUIDFromLocation_FileMissing(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "nonexistent.yaml")
	uuid := getProjectUUIDFromLocation(missingPath)
	if uuid != "" {
		t.Errorf("Expected empty UUID for missing file, got %s", uuid)
	}
}

func TestLoadProjectSettings_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	invalidContent := []byte("{invalid_yaml:::")
	configPath := filepath.Join(tmpDir, "devkit.yaml")
	err := os.WriteFile(configPath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err = loadProjectSettingsFromLocation(configPath)
	if err == nil {
		t.Error("Expected YAML parsing error, got nil")
	}
}

func TestIsTelemetryEnabled_TrueAndFalse(t *testing.T) {
	tmpDir := t.TempDir()
	truePath := filepath.Join(tmpDir, "telemetry_true.yaml")
	falsePath := filepath.Join(tmpDir, "telemetry_false.yaml")

	// Write "true" config
	os.WriteFile(truePath, []byte("project_uuid: "+uuid.New().String()+"\ntelemetry_enabled: true\n"), 0644)
	// Write "false" config
	os.WriteFile(falsePath, []byte("project_uuid: "+uuid.New().String()+"\ntelemetry_enabled: false\n"), 0644)

	// Override global path
	if !isTelemetryEnabled(truePath) {
		t.Error("Expected telemetry to be enabled")
	}

	if isTelemetryEnabled(falsePath) {
		t.Error("Expected telemetry to be disabled")
	}
}

func TestIsTelemetryEnabled_FileMissing(t *testing.T) {
	truePath := filepath.Join(t.TempDir(), "missing.yaml")
	if isTelemetryEnabled(truePath) {
		t.Error("Expected telemetry to be disabled when config is missing")
	}
}

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "devkit-config-test.yaml")
	assert.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	assert.NoError(t, err)
	tmpFile.Close()
	return tmpFile.Name()
}

func TestGetProjectUUID_WhenUUIDIsPresent(t *testing.T) {
	expectedUUID := uuid.New().String()
	content := "project_uuid: " + expectedUUID + "\n"
	writeTempConfig(t, content)

	actualUUID := getProjectUUIDFromLocation(writeTempConfig(t, content))
	assert.Equal(t, expectedUUID, actualUUID)
}

func TestGetProjectUUID_WhenConfigMissing(t *testing.T) {
	actualUUID := GetProjectUUID()
	assert.Equal(t, "", actualUUID)
}

func TestWithAppEnvironment_GeneratesUUIDWhenMissing(t *testing.T) {
	ctx := &cli.Context{
		Context: context.Background(),
	}
	WithAppEnvironment(ctx)

	env, ok := AppEnvironmentFromContext(ctx.Context)
	if !ok {
		t.Errorf("No app environment found in context")
	}
	assert.Equal(t, runtime.GOOS, env.OS)
	assert.Equal(t, runtime.GOARCH, env.Arch)
	_, err := uuid.Parse(env.ProjectUUID)
	assert.NoError(t, err)
}

func TestWithAppEnvironment_UsesUUIDFromConfig(t *testing.T) {
	expectedUUID := uuid.New().String()
	content := "project_uuid: " + expectedUUID + "\n"
	tempFile := writeTempConfig(t, content)
	ctx := &cli.Context{
		Context: context.Background(),
	}

	withAppEnvironmentFromLocation(ctx, tempFile)

	env, ok := AppEnvironmentFromContext(ctx.Context)
	if !ok {
		t.Errorf("No app environment found in context")
	}
	assert.Equal(t, runtime.GOOS, env.OS)
	assert.Equal(t, runtime.GOARCH, env.Arch)
	assert.Equal(t, expectedUUID, env.ProjectUUID)
}
