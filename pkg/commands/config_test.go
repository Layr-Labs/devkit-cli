package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestConfigCommand_Set(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a minimal eigen.toml
	mockToml := `
[project]
name = "my-avs"
version = "0.1.0"

[operator]
image = "eigen/ponos-client:v1.0"
keys = ["ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"] # Default Anvil key (index 0)
total_stake = "1000ETH" # keeping this as constant for all operators in above keys array
`
	if err := os.WriteFile(filepath.Join(tmpDir, "eigen.toml"), []byte(mockToml), 0644); err != nil {
		t.Fatal(err)
	}

	// Change working directory to temp dir
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to revert working directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{ConfigCommand},
	}

	// Test 1: Update a simple key
	if err := app.Run([]string{"app", "config", "--set", "project.name=new-avs-name"}); err != nil {
		t.Fatalf("Failed to update project name: %v", err)
	}

	// Verify updated eigen.toml
	data, err := os.ReadFile("eigen.toml")
	if err != nil {
		t.Fatalf("Failed to read eigen.toml: %v", err)
	}

	content := string(data)
	if !contains(content, `name = "new-avs-name"`) {
		t.Errorf("Expected updated project name in eigen.toml, got:\n%s", content)
	}

	// Test 2: Update array
	if err := app.Run([]string{"app", "config", "--set", "operator.keys=key1,key2,key3"}); err != nil {
		t.Fatalf("Failed to update operator keys: %v", err)
	}

	data, err = os.ReadFile("eigen.toml")
	if err != nil {
		t.Fatalf("Failed to re-read eigen.toml: %v", err)
	}

	content = string(data)
	if !contains(content, `keys = ["key1", "key2", "key3"]`) {
		t.Errorf("Expected updated operator keys array in eigen.toml, got:\n%s", content)
	}
}

// contains is a helper like strings.Contains but trims whitespace noise
func contains(content, substring string) bool {
	return strings.Contains(strings.ReplaceAll(content, " ", ""), strings.ReplaceAll(substring, " ", ""))
}
