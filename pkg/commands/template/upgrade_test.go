package template

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// MockGitClient is a mock implementation of template.GitClient for testing
type MockGitClient struct {
	// In-memory mock script content
	mockUpgradeScript string
}

func (m *MockGitClient) Clone(ctx context.Context, repoURL, dest string) error {
	// Create basic directory structure for a mock git repo
	return os.MkdirAll(filepath.Join(dest, ".devkit", "scripts"), 0755)
}

func (m *MockGitClient) Checkout(ctx context.Context, repoDir, commit string) error {
	// Create upgrade script in the target directory with mock content
	targetScript := filepath.Join(repoDir, ".devkit", "scripts", "upgrade")
	return os.WriteFile(targetScript, []byte(m.mockUpgradeScript), 0755)
}

// MockGitClientGetter implements the gitClientGetter interface for testing
type MockGitClientGetter struct {
	client template.GitClient
}

func (m *MockGitClientGetter) GetClient() template.GitClient {
	return m.client
}

// MockTemplateInfoGetter implements the templateInfoGetter interface for testing
type MockTemplateInfoGetter struct {
	projectName       string
	templateURL       string
	templateVersion   string
	shouldReturnError bool
}

func (m *MockTemplateInfoGetter) GetInfo() (string, string, string, error) {
	if m.shouldReturnError {
		return "", "", "", fmt.Errorf("config/config.yaml not found")
	}
	return m.projectName, m.templateURL, m.templateVersion, nil
}

func (m *MockTemplateInfoGetter) GetInfoDefault() (string, string, string, error) {
	if m.shouldReturnError {
		return "", "", "", fmt.Errorf("config/config.yaml not found")
	}
	return m.projectName, m.templateURL, m.templateVersion, nil
}

func (m *MockTemplateInfoGetter) GetTemplateVersionFromConfig(arch, lang string) (string, error) {
	if m.shouldReturnError {
		return "", fmt.Errorf("config/config.yaml not found")
	}
	return m.templateVersion, nil
}

func TestUpgradeCommand(t *testing.T) {
	// Create a temporary directory for testing
	testProjectsDir, err := filepath.Abs(filepath.Join(os.TempDir(), "devkit-template-upgrade-test"))
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}
	defer os.RemoveAll(testProjectsDir)

	// Ensure test directory is clean
	os.RemoveAll(testProjectsDir)

	// Create config directory and config.yaml
	configDir := filepath.Join(testProjectsDir, "config")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create config with template information - using inline yaml
	configContent := `config:
  project:
    name: template-upgrade-test
    templateBaseUrl: https://github.com/Layr-Labs/hourglass-avs-template
    templateVersion: v0.0.3
`
	configPath := filepath.Join(configDir, common.BaseConfig)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create mock template info getter
	mockTemplateInfoGetter := &MockTemplateInfoGetter{
		projectName:     "template-upgrade-test",
		templateURL:     "https://github.com/Layr-Labs/hourglass-avs-template",
		templateVersion: "v0.0.4",
	}

	// Create the test command with mocked dependencies
	testCmd := createUpgradeCommand(mockTemplateInfoGetter)

	// Create test context
	app := &cli.App{
		Name: "test-app",
		Commands: []*cli.Command{
			testCmd,
		},
	}

	// Change to the test directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	//nolint:errcheck
	defer os.Chdir(origDir)

	err = os.Chdir(testProjectsDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
	}

	// Test upgrade command with version flag
	t.Run("Upgrade command with version", func(t *testing.T) {
		// Create a flag set and context with no-op logger
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v0.0.4", "")

		// Create context with no-op logger and call Before hook
		cmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(testCmd)
		ctx := cli.NewContext(app, set, nil)

		// Execute the Before hook to set up the logger context
		if cmdWithLogger.Before != nil {
			err := cmdWithLogger.Before(ctx)
			if err != nil {
				t.Fatalf("Before hook failed: %v", err)
			}
		}

		// Run the upgrade command (which is our test command with mocks)
		err := cmdWithLogger.Action(ctx)
		if err != nil {
			t.Errorf("UpgradeCommand action returned error: %v", err)
		}

		// Verify config was updated with new version
		configData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file after upgrade: %v", err)
		}

		var configMap map[string]interface{}
		if err := yaml.Unmarshal(configData, &configMap); err != nil {
			t.Fatalf("Failed to parse config file after upgrade: %v", err)
		}

		var templateVersion string
		if configSection, ok := configMap["config"].(map[string]interface{}); ok {
			if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
				if version, ok := projectMap["templateVersion"].(string); ok {
					templateVersion = version
				}
			}
		}

		if templateVersion != "v0.0.4" {
			t.Errorf("Template version not updated. Expected 'v0.0.4', got '%s'", templateVersion)
		}
	})

	// Test upgrade command without version flag
	t.Run("Upgrade command without version", func(t *testing.T) {
		// Create a flag set and context without version flag, with no-op logger
		set := flag.NewFlagSet("test", 0)

		cmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(testCmd)
		ctx := cli.NewContext(app, set, nil)

		// Execute the Before hook to set up the logger context
		if cmdWithLogger.Before != nil {
			err := cmdWithLogger.Before(ctx)
			if err != nil {
				t.Fatalf("Before hook failed: %v", err)
			}
		}

		// Run the upgrade command (which is our test command with mocks)
		err := cmdWithLogger.Action(ctx)
		if err != nil {
			t.Errorf("UpgradeCommand action returned error: %v", err)
		}

		// Verify config was updated with new version
		configData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("Failed to read config file after upgrade: %v", err)
		}

		var configMap map[string]interface{}
		if err := yaml.Unmarshal(configData, &configMap); err != nil {
			t.Fatalf("Failed to parse config file after upgrade: %v", err)
		}

		var templateVersion string
		if configSection, ok := configMap["config"].(map[string]interface{}); ok {
			if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
				if version, ok := projectMap["templateVersion"].(string); ok {
					templateVersion = version
				}
			}
		}

		if templateVersion != "v0.0.4" {
			t.Errorf("Template version not updated. Expected 'v0.0.4', got '%s'", templateVersion)
		}
	})

	// Test upgrade command with incompatible to devkit version
	t.Run("Upgrade command with incompatible version", func(t *testing.T) {
		// Create a flag set and context with no-op logger
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v0.0.5", "")

		// Create context with no-op logger and call Before hook
		cmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(testCmd)
		ctx := cli.NewContext(app, set, nil)

		// Execute the Before hook to set up the logger context
		if cmdWithLogger.Before != nil {
			err := cmdWithLogger.Before(ctx)
			if err != nil {
				t.Fatalf("Before hook failed: %v", err)
			}
		}

		// Run the upgrade command
		err := cmdWithLogger.Action(ctx)
		if err == nil {
			t.Errorf("UpgradeCommand action should return error when using an incompatible version")
		}
	})

	// Test with missing config file
	t.Run("No config file", func(t *testing.T) {
		// Create a separate directory without a config file
		noConfigDir := filepath.Join(testProjectsDir, "no-config")
		err = os.MkdirAll(noConfigDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create no-config directory: %v", err)
		}

		// Change to the no-config directory
		err = os.Chdir(noConfigDir)
		if err != nil {
			t.Fatalf("Failed to change to no-config directory: %v", err)
		}

		// Create mock with error response for GetTemplateInfo
		errorInfoGetter := &MockTemplateInfoGetter{
			shouldReturnError: true,
		}

		// Create command with error getter and no-op logger
		errorCmd := createUpgradeCommand(errorInfoGetter)
		errorCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(errorCmd)

		errorApp := &cli.App{
			Name: "test-app",
			Commands: []*cli.Command{
				errorCmdWithLogger,
			},
		}

		// Create a flag set and context with no-op logger
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v2.0.0", "")
		ctx := cli.NewContext(errorApp, set, nil)

		// Execute the Before hook to set up the logger context
		if errorCmdWithLogger.Before != nil {
			err := errorCmdWithLogger.Before(ctx)
			if err != nil {
				t.Fatalf("Before hook failed: %v", err)
			}
		}

		// Run the upgrade command
		err := errorApp.Commands[0].Action(ctx)
		if err == nil {
			t.Errorf("UpgradeCommand action should return error when config file is missing")
		}

		// Change back to the test directory
		err = os.Chdir(testProjectsDir)
		if err != nil {
			t.Fatalf("Failed to change back to test directory: %v", err)
		}
	})
}
