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
  task:
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
	url, _, err := GetTemplateURLs(config, "task", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	if url != "https://github.com/Layr-Labs/hourglass-avs-template" {
		t.Errorf("Unexpected template URL: got %s, want %s", url, "https://github.com/Layr-Labs/hourglass-avs-template")
	}

	// Test non-existent architecture
	url, _, err = GetTemplateURLs(config, "nonexistent", "go")
	if err != nil {
		t.Fatalf("Failed to get template URL: %v", err)
	}
	if url != "" {
		t.Errorf("Expected empty URL for nonexistent architecture, got %s", url)
	}
}
