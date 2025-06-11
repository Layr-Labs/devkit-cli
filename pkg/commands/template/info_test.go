package template

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"
	"github.com/urfave/cli/v2"
)

func TestInfoCommand(t *testing.T) {
	// Create a temporary directory for testing
	testProjectsDir := filepath.Join("../../..", "test-projects", "template-info-test")
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
    name: template-info-test
    templateBaseUrl: https://github.com/Layr-Labs/hourglass-avs-template
    templateVersion: v0.0.3
`
	configPath := filepath.Join(configDir, common.BaseConfig)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create test context with no-op logger
	infoCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(InfoCommand)
	app := &cli.App{
		Name: "test-app",
		Commands: []*cli.Command{
			infoCmdWithLogger,
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

	// Test info command
	t.Run("Info command", func(t *testing.T) {
		// Create a flag set and context with no-op logger
		set := flag.NewFlagSet("test", 0)
		ctx := cli.NewContext(app, set, nil)

		// Execute the Before hook to set up the logger context
		if infoCmdWithLogger.Before != nil {
			err := infoCmdWithLogger.Before(ctx)
			if err != nil {
				t.Fatalf("Before hook failed: %v", err)
			}
		}

		// Run the info command
		err := infoCmdWithLogger.Action(ctx)
		if err != nil {
			t.Errorf("InfoCommand action returned error: %v", err)
		}
	})
}
