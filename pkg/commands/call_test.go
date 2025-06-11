package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func setupCallApp(t *testing.T) (tmpDir string, restore func(), app *cli.App, noopLogger *logger.NoopLogger) {
	tmpDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)

	oldWD, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tmpDir))

	restore = func() {
		_ = os.Chdir(oldWD)
		os.RemoveAll(tmpDir)
	}

	cmdWithLogger, noopLogger := testutils.WithTestConfigAndNoopLoggerAndAccess(CallCommand)
	app = &cli.App{
		Name:     "call",
		Commands: []*cli.Command{cmdWithLogger},
	}

	return tmpDir, restore, app, noopLogger
}

func TestCallCommand_ExecutesSuccessfully(t *testing.T) {
	_, restore, app, _ := setupCallApp(t)
	defer restore()

	err := app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.NoError(t, err)
}

func TestCallCommand_MissingDevnetYAML(t *testing.T) {
	tmpDir, restore, app, _ := setupCallApp(t)
	defer restore()

	os.Remove(filepath.Join(tmpDir, "config", "contexts", "devnet.yaml"))

	err := app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load context")
}

func TestCallCommand_MissingParams(t *testing.T) {
	_, restore, app, _ := setupCallApp(t)
	defer restore()

	err := app.Run([]string{"app", "call"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no parameters supplied")
}

func TestParseParams_MultipleParams(t *testing.T) {
	input := `signature="(uint256,string)" args='(5,"hello")'`
	m, err := parseParams(input)
	require.NoError(t, err)
	assert.Equal(t, "(uint256,string)", m["signature"])
	assert.Equal(t, `(5,"hello")`, m["args"])
}

func TestCallCommand_MalformedParams(t *testing.T) {
	_, restore, app, _ := setupCallApp(t)
	defer restore()

	err := app.Run([]string{"app", "call", "--", "badparam"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid param")
}

func TestCallCommand_MalformedYAML(t *testing.T) {
	tmpDir, restore, app, _ := setupCallApp(t)
	defer restore()

	yamlPath := filepath.Join(tmpDir, "config", "contexts", "devnet.yaml")
	err := os.WriteFile(yamlPath, []byte(":\n  - bad"), 0644)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load context")
}

func TestCallCommand_MissingScript(t *testing.T) {
	tmpDir, restore, app, _ := setupCallApp(t)
	defer restore()

	err := os.Remove(filepath.Join(tmpDir, ".devkit", "scripts", "call"))
	assert.NoError(t, err)

	err = app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCallCommand_ScriptReturnsNonZero(t *testing.T) {
	tmpDir, restore, app, _ := setupCallApp(t)
	defer restore()

	scriptPath := filepath.Join(tmpDir, ".devkit", "scripts", "call")
	failScript := "#!/bin/bash\nexit 1"
	err := os.WriteFile(scriptPath, []byte(failScript), 0755)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "call failed")
}

func TestCallCommand_ScriptOutputsInvalidJSON(t *testing.T) {
	tmpDir, restore, app, logger := setupCallApp(t)
	defer restore()

	scriptPath := filepath.Join(tmpDir, ".devkit", "scripts", "call")
	badJSON := "#!/bin/bash\necho 'not-json'\nexit 0"
	err := os.WriteFile(scriptPath, []byte(badJSON), 0755)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "call", "--", "payload=0x1"})
	assert.NoError(t, err, "Call command should succeed with non-JSON output")

	// Check that the output was logged
	assert.True(t, logger.Contains("not-json"), "Expected 'not-json' to be logged as output")
}

func TestCallCommand_Cancelled(t *testing.T) {
	_, restore, app, _ := setupCallApp(t)
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error)

	go func() {
		result <- app.RunContext(ctx, []string{"app", "call", "--", "payload=0x1"})
	}()
	cancel()

	select {
	case err := <-result:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("Call exited cleanly after context cancellation")
		} else {
			t.Errorf("Unexpected exit: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Call command did not exit after context cancellation")
	}
}
