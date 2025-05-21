package template

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

func TestUpgradeCommand(t *testing.T) {
	// Create a temporary directory for testing
	testProjectsDir := filepath.Join("../../..", "test-projects", "template-upgrade-test")
	defer os.RemoveAll(testProjectsDir)

	// Create config directory and config.yaml
	configDir := filepath.Join(testProjectsDir, "config")
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create config with template information
	configContent := `config:
  project:
    name: template-upgrade-test
    templateBaseUrl: https://github.com/Layr-Labs/custom-template
    templateVersion: v1.0.0
`
	configPath := filepath.Join(configDir, common.BaseConfig)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
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
	defer os.Chdir(origDir)

	err = os.Chdir(testProjectsDir)
	if err != nil {
		t.Fatalf("Failed to change to test directory: %v", err)
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
