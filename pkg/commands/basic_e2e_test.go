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

	scriptsDir := filepath.Join(dir, ".devkit", "scripts")

	// 2. Verify makefile works with loaded env
	cmd := exec.Command("bash", "-c", filepath.Join(scriptsDir, "build"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run make build: %v", err)
	}

	t.Logf("Make build output: %s", out)
}
