package commands

import (
	"context"
	"devkit-cli/pkg/common"
	devcontext "devkit-cli/pkg/context"
	"devkit-cli/pkg/testutils"
	"errors"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock Makefile.Devkit in main directory
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create contracts directory and its Makefile.Devkit
	contractsDir := filepath.Join(tmpDir, common.ContractsDir)
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatal(err)
	}

	mockContractsMakefile := `
.PHONY: build
build:
	@echo "Mock contracts build executed"
	`
	if err := os.WriteFile(filepath.Join(contractsDir, common.DevkitMakefile), []byte(mockContractsMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfig(BuildCommand)},
	}

	if err := app.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

// Test the case where contracts directory doesn't exist
func TestBuildCommand_NoContracts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only main Makefile.Devkit
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfig(BuildCommand)},
	}

	if err := app.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

// Test the case where contracts directory exists but has no Makefile
func TestBuildCommand_ContractsNoMakefile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock Makefile.Devkit in main directory
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create contracts directory but no Makefile
	contractsDir := filepath.Join(tmpDir, common.ContractsDir)
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfig(BuildCommand)},
	}

	// This should fail because contracts dir exists but has no Makefile
	if err := app.Run([]string{"app", "build"}); err == nil {
		t.Errorf("Expected build to fail due to missing contracts Makefile, but it succeeded")
	}
}

func TestBuildCommand_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Set up a Makefile.Devkit
	mockMakefile := `
.PHONY: build
build:
	@sleep 2
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	parentCtx, cancel := context.WithCancel(context.Background())
	ctx := devcontext.WithShutdown(parentCtx)

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfig(BuildCommand)},
	}

	done := make(chan error, 1)
	go func() {
		done <- app.RunContext(ctx, []string{"app", "build"})
	}()

	cancel()

	select {
	case err = <-done:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Logf("Build command exited with error (expected due to context cancel): %v", err)
		} else {
			t.Errorf("Expected context cancellation but received: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Build command did not exit after context cancellation")
	}
}
