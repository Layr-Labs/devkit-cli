package commands

import (
	"os"
	"path/filepath"
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
		Commands: []*cli.Command{WithTestConfig(&tmpCmd)},
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

	// Test 4: Test build after project creation
	projectPath := filepath.Join(tmpDir, "test-project")

	// Create a mock Makefile.Devkit in the project directory
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(projectPath, "Makefile.Devkit"), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to project directory to test build
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectPath); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	buildApp := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{WithTestConfig(BuildCommand)},
	}

	if err := buildApp.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}
