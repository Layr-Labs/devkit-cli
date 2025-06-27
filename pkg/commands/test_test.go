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

func setupTestApp(t *testing.T) (tmpDir string, restoreWD func(), app *cli.App, noopLogger *logger.NoopLogger) {
	tmpDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)

	// Create the test script
	scriptsDir := filepath.Join(tmpDir, ".devkit", "scripts")
	testScript := `#!/bin/bash
echo "Running tests..."
exit 0`
	err = os.WriteFile(filepath.Join(scriptsDir, "test"), []byte(testScript), 0755)
	assert.NoError(t, err)

	oldWD, err := os.Getwd()
	assert.NoError(t, err)
	assert.NoError(t, os.Chdir(tmpDir))

	restore := func() {
		_ = os.Chdir(oldWD)
		os.RemoveAll(tmpDir)
	}

	cmdWithLogger, logger := testutils.WithTestConfigAndNoopLoggerAndAccess(TestCommand)
	app = &cli.App{
		Name:     "test",
		Commands: []*cli.Command{cmdWithLogger},
	}

	return tmpDir, restore, app, logger
}

func TestTestCommand_ExecutesSuccessfully(t *testing.T) {
	_, restore, app, l := setupTestApp(t)
	defer restore()

	err := app.Run([]string{"app", "test", "--verbose"})
	assert.NoError(t, err)

	// Check that the expected message was logged
	assert.True(t, l.Contains("AVS tests completed successfully"),
		"Expected 'AVS tests completed successfully' to be logged")
}

func TestTestCommand_MissingDevnetYAML(t *testing.T) {
	tmpDir, restore, app, _ := setupTestApp(t)
	defer restore()

	os.Remove(filepath.Join(tmpDir, "config", "contexts", "devnet.yaml"))

	err := app.Run([]string{"app", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load context")
}

func TestTestCommand_MissingScript(t *testing.T) {
	tmpDir, restore, app, _ := setupTestApp(t)
	defer restore()

	os.Remove(filepath.Join(tmpDir, ".devkit", "scripts", "test"))

	err := app.Run([]string{"app", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestTestCommand_ScriptReturnsNonZero(t *testing.T) {
	tmpDir, restore, app, _ := setupTestApp(t)
	defer restore()

	scriptPath := filepath.Join(tmpDir, ".devkit", "scripts", "test")
	failScript := "#!/bin/bash\nexit 1"
	err := os.WriteFile(scriptPath, []byte(failScript), 0755)
	assert.NoError(t, err)

	err = app.Run([]string{"app", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test failed")
}

func TestTestCommand_Cancelled(t *testing.T) {
	_, restore, app, _ := setupTestApp(t)
	defer restore()

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error)
	go func() {
		result <- app.RunContext(ctx, []string{"app", "test"})
	}()
	cancel()

	select {
	case err := <-result:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("Test exited cleanly after context cancellation")
		} else {
			t.Errorf("Unexpected exit result: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Test command did not exit after context cancellation")
	}
}
