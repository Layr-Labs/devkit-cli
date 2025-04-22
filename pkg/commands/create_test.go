package commands

import (
	"testing"

	"github.com/urfave/cli/v2"
)

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Override default directory
	origCmd := CreateCommand
	tmpCmd := *CreateCommand
	tmpCmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Value: tmpDir,
		},
		&cli.StringFlag{
			Name:  "template-path",
			Value: "https://github.com/Layr-Labs/teal",
		},
	}
	CreateCommand = &tmpCmd
	defer func() { CreateCommand = origCmd }()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{&tmpCmd},
	}

	// Test 1: Missing project name
	if err := app.Run([]string{"app", "create"}); err == nil {
		t.Error("Expected error for missing project name")
	}

	// Test 2: Basic project creation
	if err := app.Run([]string{"app", "create", "test-project"}); err != nil {
		t.Errorf("Failed to create project: %v", err)
	}

	// Test 3: Project exists (trying to create same project again)
	if err := app.Run([]string{"app", "create", "test-project"}); err == nil {
		t.Error("Expected error when creating existing project")
	}
}
