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
	"gopkg.in/yaml.v3"
)

func TestSaveAndLoadProjectSettings(t *testing.T) {
	// Create temp dir for test with config structure
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create a basic config.yaml file first
	configContent := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: ""
    telemetry_enabled: false`

	configPath := filepath.Join(configDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config.yaml: %v", err)
	}

	// Test saving project settings
	testUUID := uuid.New().String()
	err = SaveProjectIdAndTelemetryToggle(tmpDir, testUUID, true)
	if err != nil {
		t.Fatalf("Failed to save project settings: %v", err)
	}

	// Verify config.yaml was updated
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not found")
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
	if settings.ProjectUUID != testUUID {
		t.Errorf("Expected ProjectUUID %s, got %s", testUUID, settings.ProjectUUID)
	}

	if !settings.TelemetryEnabled {
		t.Error("TelemetryEnabled should be true")
	}
}

func TestGetProjectUUIDFromLocation_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	expectedUUID := uuid.New().String()

	// Create config.yaml format
	configContent := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "` + expectedUUID + `"
    telemetry_enabled: true`

	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
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
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, invalidContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config file: %v", err)
	}

	_, err = loadConfigFromPath(configPath)
	if err == nil {
		t.Error("Expected YAML parsing error, got nil")
	}
}

func TestIsTelemetryEnabled_TrueAndFalse(t *testing.T) {
	tmpDir := t.TempDir()
	truePath := filepath.Join(tmpDir, "telemetry_true.yaml")
	falsePath := filepath.Join(tmpDir, "telemetry_false.yaml")

	// Write "true" config
	trueContent := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "` + uuid.New().String() + `"
    telemetry_enabled: true`

	err := os.WriteFile(truePath, []byte(trueContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write telemetry config: %v", err)
	}

	// Write "false" config
	falseContent := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "` + uuid.New().String() + `"
    telemetry_enabled: false`

	err = os.WriteFile(falsePath, []byte(falseContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write telemetry config: %v", err)
	}

	// Test telemetry enabled
	if !isTelemetryEnabledAtPath(truePath) {
		t.Error("Expected telemetry to be enabled")
	}

	if isTelemetryEnabledAtPath(falsePath) {
		t.Error("Expected telemetry to be disabled")
	}
}

func TestIsTelemetryEnabled_FileMissing(t *testing.T) {
	truePath := filepath.Join(t.TempDir(), "missing.yaml")
	if isTelemetryEnabledAtPath(truePath) {
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
	content := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "` + expectedUUID + `"
    telemetry_enabled: false`

	tempFile := writeTempConfig(t, content)
	actualUUID := getProjectUUIDFromLocation(tempFile)
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
	content := `version: 0.0.2
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "` + expectedUUID + `"
    telemetry_enabled: false`

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

// Test the new migration functionality
func TestConfigStructureWithNewFields(t *testing.T) {
	tmpDir := t.TempDir()
	testUUID := uuid.New().String()

	// Create config.yaml with new structure
	configContent := Config{
		Version: "0.0.2",
		Config: ConfigBlock{
			Project: ProjectConfig{
				Name:             "test-project",
				Version:          "0.1.0",
				Context:          "devnet",
				ProjectUUID:      testUUID,
				TelemetryEnabled: true,
			},
		},
	}

	configDir := filepath.Join(tmpDir, "config")
	err := os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)

	configPath := filepath.Join(configDir, "config.yaml")
	data, err := yaml.Marshal(configContent)
	assert.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	assert.NoError(t, err)

	// Test loading the config
	loadedConfig, err := loadConfigFromPath(configPath)
	assert.NoError(t, err)
	assert.Equal(t, testUUID, loadedConfig.Config.Project.ProjectUUID)
	assert.True(t, loadedConfig.Config.Project.TelemetryEnabled)
}
