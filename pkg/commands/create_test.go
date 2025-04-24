package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal default.eigen.toml
	mockToml := `
[project]
name = "my-avs"
version = "0.1.0"
`
	// Create default.eigen.toml in current directory
	if err := os.WriteFile("default.eigen.toml", []byte(mockToml), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("default.eigen.toml")

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

	// Override Action for testing
	tmpCmd.Action = func(cCtx *cli.Context) error {
		if cCtx.NArg() == 0 {
			return fmt.Errorf("project name is required")
		}
		projectName := cCtx.Args().First()
		targetDir := filepath.Join(cCtx.String("dir"), projectName)

		// Check if directory exists
		if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
			return fmt.Errorf("directory %s already exists", targetDir)
		}

		// Create project dir
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		// Create eigen.toml
		return copyDefaultTomlToProject(targetDir, projectName, false)
	}

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

	// Verify file exists
	eigenTomlPath := filepath.Join(tmpDir, "test-project", "eigen.toml")
	if _, err := os.Stat(eigenTomlPath); os.IsNotExist(err) {
		t.Error("eigen.toml was not created properly")
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
		Commands: []*cli.Command{BuildCommand},
	}

	if err := buildApp.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}
