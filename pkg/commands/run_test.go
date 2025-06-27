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
	"github.com/urfave/cli/v2"
)

func setupRunApp(t *testing.T) (tmpDir string, restoreWD func(), app *cli.App, noopLogger *logger.NoopLogger) {
	tmpDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)

	oldWD, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tmpDir))

	restore := func() {
		_ = os.Chdir(oldWD)
		os.RemoveAll(tmpDir)
	}

	cmdWithLogger, logger := testutils.WithTestConfigAndNoopLoggerAndAccess(RunCommand)
	app = &cli.App{
		Name:     "run",
		Commands: []*cli.Command{cmdWithLogger},
	}

	return tmpDir, restore, app, logger
}

func TestRunCommand_ExecutesSuccessfully(t *testing.T) {
	_, restore, app, logger := setupRunApp(t)
	defer restore()

	err := app.Run([]string{"app", "run", "--verbose"})
	assert.NoError(t, err)

	// Check that the expected message was logged
	assert.True(t, logger.Contains("Offchain AVS components started successfully"),
		"Expected 'Offchain AVS components started successfully' to be logged")
}

func TestRunCommand_MissingDevnetYAML(t *testing.T) {
	tmpDir, restore, app, _ := setupRunApp(t)
	defer restore()

	os.Remove(filepath.Join(tmpDir, "config", "contexts", "devnet.yaml"))

	err := app.Run([]string{"app", "run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load context")
}

func TestRunCommand_MalformedYAML(t *testing.T) {
	tmpDir, restore, app, _ := setupRunApp(t)
	defer restore()

	yamlPath := filepath.Join(tmpDir, "config", "contexts", "devnet.yaml")
	err := os.WriteFile(yamlPath, []byte(":\n  - bad"), 0644)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load context")
}

func TestRunCommand_MissingScript(t *testing.T) {
	tmpDir, restore, app, _ := setupRunApp(t)
	defer restore()

	os.Remove(filepath.Join(tmpDir, ".devkit", "scripts", "run"))

	err := app.Run([]string{"app", "run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestRunCommand_ScriptReturnsNonZero(t *testing.T) {
	tmpDir, restore, app, _ := setupRunApp(t)
	defer restore()

	scriptPath := filepath.Join(tmpDir, ".devkit", "scripts", "run")
	failScript := "#!/bin/bash\nexit 1"
	err := os.WriteFile(scriptPath, []byte(failScript), 0755)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "run"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "run failed")
}

func TestRunCommand_ScriptOutputsInvalidJSON(t *testing.T) {
	tmpDir, restore, app, logger := setupRunApp(t)
	defer restore()

	scriptPath := filepath.Join(tmpDir, ".devkit", "scripts", "run")
	badOutput := "#!/bin/bash\necho 'not-json'\n"
	err := os.WriteFile(scriptPath, []byte(badOutput), 0755)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "run"})
	assert.NoError(t, err, "Run command should succeed with non-JSON output")

	// Check that the output was logged
	assert.True(t, logger.Contains("not-json"), "Expected 'not-json' to be logged as output")
}

func TestRunCommand_Cancelled(t *testing.T) {
	_, restore, app, _ := setupRunApp(t)
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error)
	go func() {
		result <- app.RunContext(ctx, []string{"app", "run"})
	}()
	cancel()

	select {
	case err := <-result:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("Run exited cleanly after context cancellation")
		} else {
			t.Errorf("Unexpected exit result: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Run command did not exit after context cancellation")
	}
}
