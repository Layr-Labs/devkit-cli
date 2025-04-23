package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "templates.yml")
	configContent := `
architectures:
  hourglass:
    languages:
      go:
        template: "https://github.com/Layr-Labs/hourglass-avs-template"
`

	if err := os.WriteFile(testConfigPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Override the config path for testing
	oldConfigPath := configPath
	configPath = testConfigPath
	defer func() { configPath = oldConfigPath }()

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test template URL lookup
	url, err := GetTemplateURL(config, "hourglass", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	if url != "https://github.com/Layr-Labs/hourglass-avs-template" {
		t.Errorf("Unexpected template URL: got %s, want %s", url, "https://github.com/Layr-Labs/hourglass-avs-template")
	}

	// Test non-existent architecture
	url, err = GetTemplateURL(config, "nonexistent", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	if url != "" {
		t.Errorf("Expected empty URL for nonexistent architecture, got %s", url)
	}
}
