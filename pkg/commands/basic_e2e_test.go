package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/pkg/hooks"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// TODO: Enhance this test to cover other commands and more complex scenarios

func TestBasicE2E(t *testing.T) {
	// Create a temporary project directory
	tmpDir, err := os.MkdirTemp("", "e2e-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save current directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current dir: %v", err)
	}
	defer func() {
		if err := os.Chdir(currentDir); err != nil {
			t.Logf("Warning: failed to restore directory: %v", err)
		}
	}()

	// Setup test project
	projectDir := filepath.Join(tmpDir, "test-avs")
	setupBasicProject(t, projectDir)

	// Change to the project directory
	if err := os.Chdir(projectDir); err != nil {
		t.Fatalf("Failed to change to project dir: %v", err)
	}

	// Test env loading
	testEnvLoading(t, projectDir)
}

func setupBasicProject(t *testing.T, dir string) {
	// Create project directory and required files
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create config directory
	configDir := filepath.Join(dir, "config")
	err := os.MkdirAll(configDir, 0755)
	assert.NoError(t, err)

	// Create config.yaml (needed to identify project root)
	eigenContent := configs.ConfigYamls[configs.LatestVersion]
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(eigenContent), 0644); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Create .env file
	envContent := `DEVKIT_TEST_ENV=test_value
`
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env: %v", err)
	}

	// Create build script
	scriptsDir := filepath.Join(dir, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildScript := `#!/bin/bash
echo -e "Mock build executed ${DEVKIT_TEST_ENV}"`
	if err := os.WriteFile(filepath.Join(scriptsDir, "build"), []byte(buildScript), 0755); err != nil {
		t.Fatal(err)
	}
}

func testEnvLoading(t *testing.T, dir string) {
	// Backup and unset the original env var
	original := os.Getenv("DEVKIT_TEST_ENV")
	defer os.Setenv("DEVKIT_TEST_ENV", original)

	// Clear env var
	os.Unsetenv("DEVKIT_TEST_ENV")

	// 1. Simulate CLI context and run the Before hook
	app := cli.NewApp()
	cmd := &cli.Command{
		Name: "build",
		Before: func(ctx *cli.Context) error {
			return hooks.LoadEnvFile(ctx)
		},
		Action: func(ctx *cli.Context) error {
			// Verify that the env var is now set
			if val := os.Getenv("DEVKIT_TEST_ENV"); val != "test_value" {
				t.Errorf("Expected DEVKIT_TEST_ENV=test_value, got: %q", val)
			}
			return nil
		},
	}
	app.Commands = []*cli.Command{cmd}

	err := app.Run([]string{"cmd", "build"})
	if err != nil {
		t.Fatalf("CLI command failed: %v", err)
	}

	// Ref the scripts dir
	scriptsDir := filepath.Join(dir, ".devkit", "scripts")

	// 2. Run `bash -c ./build` and verify output
	cmdOut := exec.Command("bash", "-c", filepath.Join(scriptsDir, "build"))
	out, err := cmdOut.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'make build': %v\nOutput:\n%s", err, out)
	}
}
