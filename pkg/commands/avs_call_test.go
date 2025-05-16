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

func TestCallCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: call
test:
	@echo "Mock call executed"
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
		Name:     "call",
		Commands: []*cli.Command{CallCommand},
	}

	if err := app.Run([]string{"app", "call"}); err != nil {
		t.Errorf("Failed to execute run command: %v", err)
	}
}

func TestCancelledCallCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: call
run:
	@echo "Mock call executed"
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
		Name:     "call",
		Commands: []*cli.Command{CallCommand},
	}

	result := make(chan error)
	go func() {
		result <- app.RunContext(ctx, []string{"app", "call"})
	}()
	cancel()

	select {
	case err = <-result:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("Call exited cleanly after context cancellation")
		} else {
			t.Errorf("Call returned with error after context cancellation: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Call did not exit after context cancellation")
	}
}
