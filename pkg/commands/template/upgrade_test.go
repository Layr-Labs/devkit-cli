package template

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func TestUpgradeCommand(t *testing.T) {
	// Create a temporary directory for testing
	testProjectsDir, err := filepath.Abs(filepath.Join("../../..", "test-projects", "template-upgrade-test"))
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

	// Create config with template information
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

	// Create a mock template directory
	mockTemplate := filepath.Join(testProjectsDir, "template-repo")
	err = os.MkdirAll(filepath.Join(mockTemplate, ".devkit", "scripts"), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock template directory: %v", err)
	}

	// Create a mock upgrade script in the mock template
	upgradeScript := `#!/bin/bash
echo "Running upgrade script for project at: $1"
exit 0
`
	upgradeScriptPath := filepath.Join(mockTemplate, ".devkit", "scripts", "upgrade")
	err = os.WriteFile(upgradeScriptPath, []byte(upgradeScript), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock upgrade script: %v", err)
	}

	// Make sure the script is executable
	err = os.Chmod(upgradeScriptPath, 0755)
	if err != nil {
		t.Fatalf("Failed to make upgrade script executable: %v", err)
	}

	// Create test context
	app := &cli.App{
		Name: "test-app",
		Commands: []*cli.Command{
			UpgradeCommand,
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

	// Override the Git operations for testing
	origClone := gitCloneRepo
	origFetch := gitFetch
	origCheckout := gitCheckout
	defer func() {
		gitCloneRepo = origClone
		gitFetch = origFetch
		gitCheckout = origCheckout
	}()

	// Mock git clone
	gitCloneRepo = func(ctx context.Context, repoURL, targetDir string) error {
		// Create basic directory structure for a mock git repo
		err := os.MkdirAll(filepath.Join(targetDir, ".devkit", "scripts"), 0755)
		if err != nil {
			return err
		}
		return nil
	}

	// Mock git fetch - no-op for tests
	gitFetch = func(ctx context.Context, repoDir string) error {
		return nil
	}

	// Mock git checkout
	gitCheckout = func(ctx context.Context, repoDir, version string) error {
		// Copy the upgrade script to simulate successful checkout
		scriptData, err := os.ReadFile(upgradeScriptPath)
		if err != nil {
			return err
		}

		targetScript := filepath.Join(repoDir, ".devkit", "scripts", "upgrade")
		err = os.WriteFile(targetScript, scriptData, 0755)
		if err != nil {
			return err
		}

		return nil
	}

	// Test upgrade command with version flag
	t.Run("Upgrade command with version", func(t *testing.T) {
		// Create a flag set and context
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v2.0.0", "")
		ctx := cli.NewContext(app, set, nil)

		// Run the upgrade command
		err := UpgradeCommand.Action(ctx)
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

		if templateVersion != "v2.0.0" {
			t.Errorf("Template version not updated. Expected 'v2.0.0', got '%s'", templateVersion)
		}
	})

	// Test upgrade command without version flag
	t.Run("Upgrade command without version", func(t *testing.T) {
		// Create a flag set and context without version flag
		set := flag.NewFlagSet("test", 0)
		ctx := cli.NewContext(app, set, nil)

		// Run the upgrade command
		err := UpgradeCommand.Action(ctx)
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

		// Create a flag set and context
		set := flag.NewFlagSet("test", 0)
		set.String("version", "v2.0.0", "")
		ctx := cli.NewContext(app, set, nil)

		// Run the upgrade command
		err := UpgradeCommand.Action(ctx)
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
