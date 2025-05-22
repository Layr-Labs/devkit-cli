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
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// MockGitClient is a mock implementation of template.GitClient for testing
type MockGitClient struct {
	// In-memory mock script content
	mockUpgradeScript string
}

func (m *MockGitClient) SubmoduleInit(ctx context.Context, repoDir string, opts template.CloneOptions) error {
	return nil
}

func (m *MockGitClient) Clone(ctx context.Context, repoURL, dest string, opts template.CloneOptions) error {
	// Create basic directory structure for a mock git repo
	return os.MkdirAll(filepath.Join(dest, ".devkit", "scripts"), 0755)
}

func (m *MockGitClient) Checkout(ctx context.Context, repoDir, commit string) error {
	// Create upgrade script in the target directory with mock content
	targetScript := filepath.Join(repoDir, ".devkit", "scripts", "upgrade")
	return os.WriteFile(targetScript, []byte(m.mockUpgradeScript), 0755)
}

// Implement other required methods of GitClient with minimal functionality for testing
func (m *MockGitClient) WorktreeCheckout(ctx context.Context, mirrorPath, commit, worktreePath string) error {
	return nil
}

func (m *MockGitClient) SubmoduleList(ctx context.Context, repoDir string) ([]template.Submodule, error) {
	return nil, nil
}

func (m *MockGitClient) SubmoduleCommit(ctx context.Context, repoDir, path string) (string, error) {
	return "", nil
}

func (m *MockGitClient) ResolveRemoteCommit(ctx context.Context, repoURL, branch string) (string, error) {
	return "", nil
}

func (m *MockGitClient) RetryClone(ctx context.Context, repoURL, dest string, opts template.CloneOptions, maxRetries int) error {
	return nil
}

func (m *MockGitClient) SubmoduleClone(
	ctx context.Context,
	submodule template.Submodule,
	commit string,
	repoUrl string,
	targetDir string,
	repoDir string,
	opts template.CloneOptions,
) error {
	return nil
}

func (m *MockGitClient) CheckoutCommit(ctx context.Context, repoDir, commitHash string) error {
	return nil
}

func (m *MockGitClient) StageSubmodule(ctx context.Context, repoDir, path, sha string) error {
	return nil
}

func (m *MockGitClient) SetSubmoduleURL(ctx context.Context, repoDir, name, url string) error {
	return nil
}

func (m *MockGitClient) ActivateSubmodule(ctx context.Context, repoDir, name string) error {
	return nil
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
    templateVersion: v0.0.5
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
		templateVersion: "v0.0.5",
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
		// Create a flag set and context
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v0.0.6", "")
		ctx := cli.NewContext(app, set, nil)

		// Run the upgrade command (which is our test command with mocks)
		err := app.Commands[0].Action(ctx)
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

		if templateVersion != "v0.0.6" {
			t.Errorf("Template version not updated. Expected 'v0.0.6', got '%s'", templateVersion)
		}
	})

	// Test upgrade command without version flag
	t.Run("Upgrade command without version", func(t *testing.T) {
		// Create a flag set and context without version flag
		set := flag.NewFlagSet("test", 0)
		ctx := cli.NewContext(app, set, nil)

		// Run the upgrade command
		err := app.Commands[0].Action(ctx)
		if err == nil {
			t.Errorf("UpgradeCommand action should return error when version flag is missing")
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

		// Create command with error getter
		errorCmd := createUpgradeCommand(errorInfoGetter)

		errorApp := &cli.App{
			Name: "test-app",
			Commands: []*cli.Command{
				errorCmd,
			},
		}

		// Create a flag set and context
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v2.0.0", "")
		ctx := cli.NewContext(errorApp, set, nil)

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
