package commands

import (
	"context"
	"devkit-cli/pkg/common"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
)

func TestRunCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: run
run:
	@echo "Mock run executed"
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
		Commands: []*cli.Command{RunCommand},
	}

	if err := app.Run([]string{"app", "run"}); err != nil {
		t.Errorf("Failed to execute run command: %v", err)
	}
}

func TestCancelledRunCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: run
run:
	@echo "Mock run executed"
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
		Commands: []*cli.Command{RunCommand},
	}

	result := make(chan error)
	go func() {
		result <- app.RunContext(ctx, []string{"app", "run"})
	}()
	cancel()

	select {
	case err = <-result:
		if err == nil {
			t.Log("Run exited cleanly after context cancellation")
		} else {
			t.Logf("Run returned with error after context cancellation: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Run did not exit after context cancellation")
	}
}
