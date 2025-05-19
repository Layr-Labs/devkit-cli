package commands

import (
	"bytes"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEditorDetection tests the logic of detecting available editors
func TestEditorDetection(t *testing.T) {
	// Test with environment variable set
	os.Setenv("EDITOR", "test-editor")
	editor := os.Getenv("EDITOR")
	if editor != "test-editor" {
		t.Errorf("Failed to set EDITOR environment variable")
	}

	// Test editor detection logic
	commonEditors := []string{"nano", "vi", "vim"}
	found := false

	for _, ed := range commonEditors {
		if path, err := exec.LookPath(ed); err == nil {
			found = true
			t.Logf("Found editor: %s at %s", ed, path)
			break
		}
	}

	// This is informational, not a failure condition
	if !found {
		t.Logf("No common editors found on this system")
	}
}

// TestBackupAndRestore tests the logic of backing up and restoring files
func TestBackupAndRestoreYAML(t *testing.T) {
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, common.BaseConfig)

	originalContent := `
version: 0.1.0
config:
  project:
    name: "my-avs"
    version: "0.1.0"
    context: "devnet"
`
	err := os.WriteFile(testConfigPath, []byte(originalContent), 0644)
	require.NoError(t, err)

	// Backup
	backupData, err := os.ReadFile(testConfigPath)
	require.NoError(t, err)

	// Modify
	modifiedContent := strings.ReplaceAll(originalContent, "my-avs", "updated-avs")
	require.NoError(t, os.WriteFile(testConfigPath, []byte(modifiedContent), 0644))

	// Restore
	require.NoError(t, os.WriteFile(testConfigPath, backupData, 0644))

	restoredData, err := os.ReadFile(testConfigPath)
	require.NoError(t, err)
	require.Contains(t, string(restoredData), "my-avs")
}

// TestYAMLValidation tests the YAML validation logic
func TestValidateYAML(t *testing.T) {
	tempDir := t.TempDir()

	validYAML := `
version: 0.1.0
config:
  project:
    name: "valid-avs"
    version: "0.1.0"
    context: "devnet"
`
	invalidYAML := `
config:
  project:
    name: "broken-avs
    version: "0.1.0"
`

	validPath := filepath.Join(tempDir, "valid.yaml")
	invalidPath := filepath.Join(tempDir, "invalid.yaml")

	require.NoError(t, os.WriteFile(validPath, []byte(validYAML), 0644))
	require.NoError(t, os.WriteFile(invalidPath, []byte(invalidYAML), 0644))

	_, err := validateConfig(validPath)
	require.NoError(t, err)

	_, err = validateConfig(invalidPath)
	require.Error(t, err)
	t.Logf("Expected YAML parse error: %v", err)
}

// TestEditorLaunching tests the logic of launching an editor
func TestEditorLaunching(t *testing.T) {
	// Test with a mock editor (echo)
	editor := "echo"
	if _, err := exec.LookPath(editor); err != nil {
		t.Skip("echo command not available, skipping test")
	}

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-file.txt")

	// Create test file
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a command that would simulate an editor (echo appends to the file)
	cmd := exec.Command(editor, "edited content", ">", testFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Test with shell to handle redirection
	shellCmd := exec.Command("bash", "-c", editor+" 'edited content' > "+testFile)
	err = shellCmd.Run()
	if err != nil {
		t.Errorf("Failed to run mock editor command: %v", err)
		t.Logf("Stderr: %s", stderr.String())
		return
	}

	// Check if the file was modified (this doesn't test waiting for editor to close)
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read test file after edit: %v", err)
	}

	if strings.TrimSpace(string(content)) != "edited content" {
		t.Errorf("Editor didn't modify file as expected. Got: %s", string(content))
	}
}
