package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"devkit-cli/config"
	"devkit-cli/pkg/hooks"

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
	testEnvLoading(t)
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
	eigenContent := config.DefaultConfigYaml
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(eigenContent), 0644); err != nil {
		t.Fatalf("Failed to write config.yaml: %v", err)
	}

	// Create .env file
	envContent := `DEVKIT_TEST_ENV=test_value
`
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(envContent), 0644); err != nil {
		t.Fatalf("Failed to write .env: %v", err)
	}

	// Create Makefile.Devkit
	makefileContent := `
.PHONY: build run
build:
	@echo "Building with env: DEVKIT_TEST_ENV=$${DEVKIT_TEST_ENV:-not_set}"
run:
	@echo "Running with env: DEVKIT_TEST_ENV=$${DEVKIT_TEST_ENV:-not_set}"
`
	if err := os.WriteFile(filepath.Join(dir, "Makefile.Devkit"), []byte(makefileContent), 0644); err != nil {
		t.Fatalf("Failed to write Makefile.Devkit: %v", err)
	}
}

func testEnvLoading(t *testing.T) {
	// Backup and unset the original env var
	original := os.Getenv("DEVKIT_TEST_ENV")
	defer os.Setenv("DEVKIT_TEST_ENV", original)
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

	// 2. Run `make build` and verify output
	cmdOut := exec.Command("make", "-f", "Makefile.Devkit", "build")
	out, err := cmdOut.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'make build': %v\nOutput:\n%s", err, out)
	}
	t.Logf("Make build output:\n%s", out)

	// 3. Run `make run` and verify output
	cmdOut = exec.Command("make", "-f", "Makefile.Devkit", "run")
	out, err = cmdOut.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run 'make run': %v\nOutput:\n%s", err, out)
	}
	t.Logf("Make run output:\n%s", out)
}
