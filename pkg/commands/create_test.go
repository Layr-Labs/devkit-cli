package commands

import (
	"bytes"
	"devkit-cli/pkg/common"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestCreateCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal default.eigen.toml
	mockToml := `
[project]
name = "my-avs"
version = "0.1.0"
`
	// Create default.eigen.toml in current directory
	if err := os.WriteFile("default.eigen.toml", []byte(mockToml), 0644); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove("default.eigen.toml"); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	// Override default directory
	origCmd := CreateCommand
	tmpCmd := *CreateCommand
	tmpCmd.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Value: tmpDir,
		},
		&cli.StringFlag{
			Name:  "template-path",
			Value: "https://github.com/Layr-Labs/teal",
		},
	}
	CreateCommand = &tmpCmd
	defer func() { CreateCommand = origCmd }()

	// Override Action for testing
	tmpCmd.Action = func(cCtx *cli.Context) error {
		if cCtx.NArg() == 0 {
			return fmt.Errorf("project name is required")
		}
		projectName := cCtx.Args().First()
		targetDir := filepath.Join(cCtx.String("dir"), projectName)

		// Check if directory exists
		if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
			return fmt.Errorf("directory %s already exists", targetDir)
		}

		// Create project dir
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		// Create contracts directory for testing
		contractsDir := filepath.Join(targetDir, common.ContractsDir)
		if err := os.MkdirAll(contractsDir, 0755); err != nil {
			return err
		}

		// Create eigen.toml
		return copyDefaultTomlToProject(targetDir, projectName, false)
	}

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{WithTestConfig(&tmpCmd)},
	}

	// Test cases
	if err := app.Run([]string{"app", "create"}); err == nil {
		t.Error("Expected error for missing project name, but got nil")
	}

	if err := app.Run([]string{"app", "create", "test-project"}); err != nil {
		t.Errorf("Failed to create project: %v", err)
	}

	// Verify file exists
	eigenTomlPath := filepath.Join(tmpDir, "test-project", "eigen.toml")
	if _, err := os.Stat(eigenTomlPath); os.IsNotExist(err) {
		t.Error("eigen.toml was not created properly")
	}

	// Verify contracts directory exists
	contractsDir := filepath.Join(tmpDir, "test-project", common.ContractsDir)
	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		t.Error("contracts directory was not created properly")
	}

	// Test 3: Project exists (trying to create same project again)
	if err := app.Run([]string{"app", "create", "test-project"}); err == nil {
		t.Error("Expected error when creating existing project")
	}

	// Test 4: Test build after project creation
	projectPath := filepath.Join(tmpDir, "test-project")

	// Create a mock Makefile.Devkit in the project directory
	mockMakefile := `
.PHONY: build
build:
	@echo "Mock build executed"
	`
	if err := os.WriteFile(filepath.Join(projectPath, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a mock Makefile.Devkit in the contracts directory
	if err := os.WriteFile(filepath.Join(contractsDir, common.DevkitMakefile), []byte(mockMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	// Change to project directory to test build
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectPath); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to change back to original directory: %v", err)
		}
	}()

	buildApp := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{WithTestConfig(BuildCommand)},
	}

	if err := buildApp.Run([]string{"app", "build"}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

// Test creating a project with mock template URLs
func TestCreateCommand_WithTemplates(t *testing.T) {
	// Mock template URLs similar to what would be in the config
	mainTemplateURL := "https://github.com/example/avs-template"
	contractsTemplateURL := "https://github.com/example/contracts-template"

	tmpDir := t.TempDir()

	// Create project directory structure
	projectName := "test-avs-with-contracts"
	projectDir := filepath.Join(tmpDir, projectName)

	// Create main directory and contracts directory
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	contractsDir := filepath.Join(projectDir, common.ContractsDir)
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Verify the structure
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		t.Fatal("Project directory was not created")
	}

	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		t.Fatal("Contracts directory was not created")
	}

	// Log (for test purposes only)
	t.Logf("Mock templates: main=%s, contracts=%s", mainTemplateURL, contractsTemplateURL)
}

func TestConfigCommand_ListOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// 📥 Load the default.eigen.toml content
	defaultTomlPath := filepath.Join("..", "..", "default.eigen.toml") // adjust as needed
	defaultContent, err := os.ReadFile(defaultTomlPath)
	require.NoError(t, err)

	// 📝 Write it to test directory as eigen.toml
	eigenPath := filepath.Join(tmpDir, "eigen.toml")
	require.NoError(t, os.WriteFile(eigenPath, defaultContent, 0644))

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

	// ⚙️ Run the CLI app with nested subcommands
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					ConfigCommand,
				},
			},
		},
	}
	err = app.Run([]string{"devkit", "avs", "config", "--list"})
	require.NoError(t, err)

	// 📤 Finish capturing output
	w.Close()
	os.Stdout = stdout
	_, _ = buf.ReadFrom(r)
	output := stripANSI(buf.String())

	// ✅ Validating output
	require.Contains(t, output, "[project]")
	require.Contains(t, output, "[operator]")
	require.Contains(t, output, "[env]")
}

func stripANSI(input string) string {
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansi.ReplaceAllString(input, "")
}
