package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestBuildCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mock Makefile.Devkit
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
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
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{BuildCommand},
	}

	if err := app.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}
