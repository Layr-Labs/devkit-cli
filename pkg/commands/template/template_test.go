package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
)

func TestGetTemplateInfo(t *testing.T) {
	// Create a temporary directory for testing
	testDir := filepath.Join(os.TempDir(), "devkit-test-template")
	defer os.RemoveAll(testDir)

	// Create config directory and config.yaml
	configDir := filepath.Join(testDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Change to the test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	//nolint:errcheck
	defer os.Chdir(origDir)

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Test with template information
	t.Run("With template information in config", func(t *testing.T) {
		// Test with template information
		configContent := `config:
  project:
    name: test-project
    templateBaseUrl: https://github.com/Layr-Labs/hourglass-avs-template
    templateVersion: v0.0.3
`
		configPath := filepath.Join(configDir, common.BaseConfig)
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		projectName, templateURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			t.Fatalf("GetTemplateInfo failed: %v", err)
		}

		if projectName != "test-project" {
			t.Errorf("Expected project name 'test-project', got '%s'", projectName)
		}
		if templateURL != "https://github.com/Layr-Labs/hourglass-avs-template" {
			t.Errorf("Expected template URL 'https://github.com/Layr-Labs/hourglass-avs-template', got '%s'", templateURL)
		}
		if templateVersion != "v0.0.3" {
			t.Errorf("Expected template version 'v0.0.3', got '%s'", templateVersion)
		}
	})

	// Test with no template info in config and falling back to hardcoded values
	t.Run("Without template information falling back to hardcoded values", func(t *testing.T) {
		// Update config content to remove template info
		configContent := `config:
  project:
    name: test-project
`
		configPath := filepath.Join(configDir, common.BaseConfig)
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write config file: %v", err)
		}

		projectName, templateURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			t.Fatalf("GetTemplateInfo failed: %v", err)
		}

		if projectName != "test-project" {
			t.Errorf("Expected project name 'test-project', got '%s'", projectName)
		}

		// With the real implementation, we can't fully mock pkgtemplate.LoadConfig as it's not a variable,
		// so we'll check that we at least get a fallback value
		if templateURL == "" {
			t.Errorf("Expected a fallback template URL, got empty string")
		}
		// Most likely the hardcoded value from GetTemplateInfo()
		if templateVersion != "unknown" && templateVersion == "" {
			t.Errorf("Expected template version to be populated, got '%s'", templateVersion)
		}
	})

	// Test with missing config file
	t.Run("No config file", func(t *testing.T) {
		// Remove config file
		err = os.Remove(filepath.Join(configDir, common.BaseConfig))
		if err != nil {
			t.Fatalf("Failed to remove config file: %v", err)
		}

		_, _, _, err := GetTemplateInfo()
		if err == nil {
			t.Errorf("Expected error for missing config file, got nil")
		}
	})
}
