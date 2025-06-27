package commands

import (
	"context"

	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

const contractsBasePath = ".devkit/contracts"

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()
	logger := logger.NewNoopLogger()
	mockConfigYaml := configs.ConfigYamls[configs.LatestVersion]
	configDir := filepath.Join("config")
	err := os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)

	// Create config/config.yaml in current directory
	if err := os.WriteFile(filepath.Join(configDir, common.BaseConfig), []byte(mockConfigYaml), 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(filepath.Join("config", common.BaseConfig)); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	devnetYaml := contexts.ContextYamls[contexts.LatestVersion]
	contextsDir := filepath.Join(configDir, "contexts")
	err = os.MkdirAll(contextsDir, 0755)
	assert.NoError(t, err)

	// Create config/context/devnet.yaml in current directory
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(devnetYaml), 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(filepath.Join(contextsDir, "devnet.yaml")); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	// Override default directory
	origCmd := CreateCommand
	tmpCmd := *CreateCommand
	tmpCmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Value: tmpDir,
		},
		&cli.StringFlag{
			Name:  "template-url",
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

		// Create contracts directory for testing
		contractsDir := filepath.Join(targetDir, common.ContractsDir)
		if err := os.MkdirAll(contractsDir, 0755); err != nil {
			return err
		}

		// Load the current config
		config, err := template.LoadConfig()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Test template URL lookup
		mainBaseURL, mainVersion, err := template.GetTemplateURLs(config, "task", "go")
		if err != nil {
			t.Fatalf("Failed to get template URLs: %v", err)
		}

		// Create config.yaml
		return copyDefaultConfigToProject(logger, targetDir, projectName, "test-uuid", mainBaseURL, mainVersion, false)
	}

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(&tmpCmd)},
	}

	// Test cases
	if err := app.Run([]string{"app", "create"}); err == nil {
		t.Error("Expected error for missing project name, but got nil")
	}

	if err := app.Run([]string{"app", "create", "test-project"}); err != nil {
		t.Errorf("Failed to create project: %v", err)
	}

	// Verify file exists
	eigenTomlPath := filepath.Join(tmpDir, "test-project", "config", common.BaseConfig)
	if _, err := os.Stat(eigenTomlPath); os.IsNotExist(err) {
		t.Errorf("config/%s was not created properly", common.BaseConfig)
	}

	// Verify contracts directory exists
	contractsDir := filepath.Join(tmpDir, "test-project", common.ContractsDir)
	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		t.Error("contracts directory was not created properly")
	}

	// Test 3: Project exists (trying to create same project again)
	if err := app.Run([]string{"app", "create", "test-project"}); err == nil {
		t.Error("Expected error when creating existing project")
	}

	// Test 4: Test build after project creation
	projectPath := filepath.Join(tmpDir, "test-project")

	// Create a mock Devkit in the contracts directory
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	t.Logf("Creating makefile path: %s", filepath.Join(projectPath, contractsBasePath))
	if err := os.MkdirAll(filepath.Join(projectPath, contractsBasePath), 0775); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectPath, contractsBasePath, common.Makefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build script
	scriptsDir := filepath.Join(projectPath, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildScript := `#!/bin/bash
	echo "Mock create executed"`
	if err := os.WriteFile(filepath.Join(scriptsDir, "build"), []byte(buildScript), 0755); err != nil {
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
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	if err := buildApp.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

// Test creating a project with mock template URLs
func TestCreateCommand_WithTemplates(t *testing.T) {
	// Mock template URLs similar to what would be in the config
	mainTemplateURL := "https://github.com/example/avs-template"
	contractsTemplateURL := "https://github.com/example/contracts-template"

	tmpDir := t.TempDir()

	// Create project directory structure
	projectName := "test-avs-with-contracts"
	projectDir := filepath.Join(tmpDir, projectName)

	// Create main directory and contracts directory
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	contractsDir := filepath.Join(projectDir, common.ContractsDir)
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Verify the structure
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Fatal("Project directory was not created")
	}

	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		t.Fatal("Contracts directory was not created")
	}

	// Log (for test purposes only)
	t.Logf("Mock templates: main=%s, contracts=%s", mainTemplateURL, contractsTemplateURL)
}

func TestCreateCommand_ContextCancellation(t *testing.T) {
	mockYaml := configs.ConfigYamls[configs.LatestVersion]
	if err := os.WriteFile("config.yaml", []byte(mockYaml), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove("config.yaml")

	origCmd := CreateCommand
	origCmd.Action = func(cCtx *cli.Context) error {
		<-cCtx.Context.Done()
		return cCtx.Context.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(origCmd)},
	}

	done := make(chan error, 1)
	go func() {
		done <- app.RunContext(ctx, []string{"app", "create", "cancelled-avs"})
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Logf("Expected context cancellation received: %v", err)
		} else {
			t.Errorf("Expected context cancellation but received: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Create command did not exit after context cancellation")
	}
}
