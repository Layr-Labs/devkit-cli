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

	// Test with template information
	configContent := `config:
  project:
    name: test-project
    templateBaseUrl: https://github.com/Layr-Labs/custom-template
    templateVersion: v1.2.3
`
	configPath := filepath.Join(configDir, common.BaseConfig)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Change to the test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(origDir)

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Test with template information
	t.Run("With template information", func(t *testing.T) {
		projectName, templateURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			t.Fatalf("GetTemplateInfo failed: %v", err)
		}

		if projectName != "test-project" {
			t.Errorf("Expected project name 'test-project', got '%s'", projectName)
		}
		if templateURL != "https://github.com/Layr-Labs/custom-template" {
			t.Errorf("Expected template URL 'https://github.com/Layr-Labs/custom-template', got '%s'", templateURL)
		}
		if templateVersion != "v1.2.3" {
			t.Errorf("Expected template version 'v1.2.3', got '%s'", templateVersion)
		}
	})

	// Test without template information
	t.Run("Without template information", func(t *testing.T) {
		// Update config content to remove template info
		configContent := `config:
  project:
    name: test-project
`
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

		// Should get default values
		defaultURL := "https://github.com/Layr-Labs/hourglass-avs-template"
		if templateURL != defaultURL {
			t.Errorf("Expected default template URL '%s', got '%s'", defaultURL, templateURL)
		}
		if templateVersion != "unknown" {
			t.Errorf("Expected default template version 'unknown', got '%s'", templateVersion)
		}
	})

	// Test with missing config file
	t.Run("No config file", func(t *testing.T) {
		// Remove config file
		err = os.Remove(configPath)
		if err != nil {
			t.Fatalf("Failed to remove config file: %v", err)
		}

		_, _, _, err := GetTemplateInfo()
		if err == nil {
			t.Errorf("Expected error for missing config file, got nil")
		}
	})
}
