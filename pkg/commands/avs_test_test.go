package commands

import (
	"context"
	"devkit-cli/pkg/common"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

func TestTestCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: test
test:
	@echo "Mock test executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Run from temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{TestCommand},
	}

	if err := app.Run([]string{"app", "test"}); err != nil {
		t.Errorf("Failed to execute run command: %v", err)
	}
}

func TestCancelledTestCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: test
run:
	@echo "Mock test executed"
	`
	if err := os.WriteFile(filepath.Join(tmpDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Run from temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{TestCommand},
	}

	result := make(chan error)
	go func() {
		result <- app.RunContext(ctx, []string{"app", "test"})
	}()
	cancel()

	select {
	case err = <-result:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("Test exited cleanly after context cancellation")
		} else {
			t.Errorf("Test returned with error after context cancellation: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Test did not exit after context cancellation")
	}
}
