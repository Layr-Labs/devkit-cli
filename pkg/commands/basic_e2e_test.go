package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"devkit-cli/pkg/hooks"

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

	// Create eigen.toml (needed to identify project root)
	eigenContent := `[project]
name = "test-avs"
version = "0.1.0"
`
	if err := os.WriteFile(filepath.Join(dir, "eigen.toml"), []byte(eigenContent), 0644); err != nil {
		t.Fatalf("Failed to write eigen.toml: %v", err)
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
	// Clear env var first
	os.Unsetenv("DEVKIT_TEST_ENV")

	// 1. Test that the middleware loads .env
	action := func(c *cli.Context) error { return nil }
	ctx := cli.NewContext(cli.NewApp(), nil, nil)
	ctx.Command = &cli.Command{Name: "build"}

	if err := hooks.WithEnvLoader(action)(ctx); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Verify env var was loaded
	if val := os.Getenv("DEVKIT_TEST_ENV"); val != "test_value" {
		t.Errorf("Expected DEVKIT_TEST_ENV=test_value, got: %q", val)
	}

	// 2. Verify makefile works with loaded env
	cmd := exec.Command("make", "-f", "Makefile.Devkit", "build")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run make build: %v", err)
	}

	t.Logf("Make build output: %s", out)

	// 3. Verify makefile works with loaded env
	cmd = exec.Command("make", "-f", "Makefile.Devkit", "run")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run make run: %v", err)
	}

	t.Logf("Make run output: %s", out)
}
