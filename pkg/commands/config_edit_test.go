package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
func TestBackupAndRestore(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	testConfigPath := filepath.Join(tempDir, "config.toml")

	// Write test content
	originalContent := "original content"
	err := os.WriteFile(testConfigPath, []byte(originalContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create backup
	backupData, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	// Modify the file
	modifiedContent := "modified content"
	err = os.WriteFile(testConfigPath, []byte(modifiedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Restore from backup
	err = os.WriteFile(testConfigPath, backupData, 0644)
	if err != nil {
		t.Fatalf("Failed to restore test file: %v", err)
	}

	// Verify restoration
	restoredData, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("Failed to read restored file: %v", err)
	}

	if string(restoredData) != originalContent {
		t.Errorf("Restore failed. Expected: %s, Got: %s", originalContent, string(restoredData))
	}
}

// TestTomlValidation tests the TOML validation logic
func TestTomlValidation(t *testing.T) {
	// Valid TOML
	validToml := `
[section]
key = "value"
`
	// Invalid TOML
	invalidToml := `
[section
key = "value"
`
	// Create test files
	tempDir := t.TempDir()
	validPath := filepath.Join(tempDir, "valid.toml")
	invalidPath := filepath.Join(tempDir, "invalid.toml")

	err := os.WriteFile(validPath, []byte(validToml), 0644)
	if err != nil {
		t.Fatalf("Failed to create valid TOML file: %v", err)
	}

	err = os.WriteFile(invalidPath, []byte(invalidToml), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid TOML file: %v", err)
	}

	// Test validation with external command
	cmd := exec.Command("bash", "-c", "type toml &>/dev/null && toml validate "+validPath+" &>/dev/null")
	err = cmd.Run()
	if err != nil {
		// Skip test if toml command is not available
		t.Logf("Skipping TOML validation test: toml command not available")
	} else {
		// Test valid TOML
		validCmd := exec.Command("toml", "validate", validPath)
		err = validCmd.Run()
		if err != nil {
			t.Errorf("Failed to validate valid TOML: %v", err)
		}

		// Test invalid TOML
		invalidCmd := exec.Command("toml", "validate", invalidPath)
		err = invalidCmd.Run()
		if err == nil {
			t.Errorf("Invalid TOML was incorrectly validated as valid")
		}
	}
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
