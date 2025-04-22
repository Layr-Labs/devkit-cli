package commands

import (
	"os"
	"path/filepath"
	"testing"

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
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile.Devkit"), []byte(mockMakefile), 0644); err != nil {
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
	defer os.Chdir(oldWd)

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{RunCommand},
	}

	if err := app.Run([]string{"app", "run"}); err != nil {
		t.Errorf("Failed to execute run command: %v", err)
	}
}
