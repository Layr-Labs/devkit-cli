package config

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestConfigCommand_ListOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.yaml with embedded telemetry settings
	configContent := `version: 0.0.2
config:
  project:
    name: "my-avs"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "d7598c91-2ec4-4751-b0ab-bc848f73d58e"
    telemetry_enabled: true`

	defaultDevnetConfigFile := contexts.ContextYamls[contexts.LatestVersion]

	configPath := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configPath, common.BaseConfig), []byte(configContent), 0644))
	contextsPath := filepath.Join(configPath, "contexts")
	require.NoError(t, os.MkdirAll(contextsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(contextsPath, "devnet.yaml"), defaultDevnetConfigFile, 0644))

	// 🔁 Change into the test directory
	originalWD, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to return to original directory: %v", err)
		}
	}()
	require.NoError(t, os.Chdir(tmpDir))

	// 🧪 Capture os.Stdout
	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// ⚙️ Run the CLI app with nested subcommands and no-op logger
	cmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(Command)
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					cmdWithLogger,
				},
				Before: func(cCtx *cli.Context) error {
					// Execute the command's Before hook to set up logger context
					if cmdWithLogger.Before != nil {
						return cmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}
	err := app.Run([]string{"devkit", "avs", "config", "--list"})
	require.NoError(t, err)

	// 📤 Finish capturing output
	w.Close()
	os.Stdout = stdout
	_, _ = buf.ReadFrom(r)
	// output := stripANSI(buf.String())

	// ✅ Validating output
	// require.Contains(t, output, "[project]")
	// require.Contains(t, output, "[operator]")
	// require.Contains(t, output, "[env]")
}

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
version: 0.0.2
config:
  project:
    name: "valid-avs"
    version: "0.1.0"
    context: "devnet"
    project_uuid: "test-uuid"
    telemetry_enabled: true
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

	_, err := ValidateConfig(validPath, Config)
	require.NoError(t, err)

	_, err = ValidateConfig(invalidPath, Config)
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
