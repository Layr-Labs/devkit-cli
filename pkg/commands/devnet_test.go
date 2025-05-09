package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

)

// helper to create a temp AVS project dir with eigen.toml copied
func createTempAVSProject(defaultEigenPath string) (string, error) {
	tempDir, err := os.MkdirTemp("", "devkit-test-avs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	destEigen := filepath.Join(tempDir, common.EigenTomlPath)

	// Copy default eigen.toml
	srcFile, err := os.Open(defaultEigenPath)
	if err != nil {
		return "", fmt.Errorf("failed to open default eigen.toml: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destEigen)
	if err != nil {
		return "", fmt.Errorf("failed to create destination eigen.toml: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to copy eigen.toml: %w", err)
	}

	return tempDir, nil
}

func TestStartAndStopDevnet(t *testing.T) {
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd) // Restore cwd after test
	})
	defaultEigenPath := filepath.Join("..", "..", "default.eigen.toml")

	projectDir, err := createTempAVSProject(defaultEigenPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start
	startApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StartDevnetAction,
	}

	err = startApp.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)

	// Stop
	stopApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StopDevnetAction,
	}

	err = stopApp.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)
}

func TestStartDevnetOnUsedPort_ShouldFail(t *testing.T) {
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd) // Restore cwd after test
	})

	defaultEigenPath, err := filepath.Abs(filepath.Join("..", "..", "default.eigen.toml"))
	assert.NoError(t, err, "failed to resolve default.eigen.toml path")
	assert.FileExists(t, defaultEigenPath, "eigen file does not exist at computed path")

	projectDir1, err := createTempAVSProject(defaultEigenPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir1)

	projectDir2, err := createTempAVSProject(defaultEigenPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir2)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start from dir1
	err = os.Chdir(projectDir1)
	assert.NoError(t, err)

	app1 := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StartDevnetAction,
	}
	err = app1.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)

	// Attempt from dir2
	err = os.Chdir(projectDir2)
	assert.NoError(t, err)

	app2 := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StartDevnetAction,
	}
	err = app2.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in use")

	// Cleanup from dir1
	err = os.Chdir(projectDir1)
	assert.NoError(t, err)

	stopApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StopDevnetAction,
	}
	_ = stopApp.Run([]string{"devkit", "--port", port, "--verbose"})
}

// getFreePort finds an available TCP port for testing
func getFreePort() (string, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	return strconv.Itoa(port), nil
}


func TestListRunningDevnets(t *testing.T) {
	// Save original working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Prepare temp AVS project
	defaultEigenPath := filepath.Join("..", "..", "default.eigen.toml")
	projectDir, err := createTempAVSProject(defaultEigenPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start devnet
	startApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StartDevnetAction,
	}
	err = startApp.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)

	// Capture output of list
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listApp := &cli.App{
		Name:   "devkit",
		Action: ListDevnetContainersAction,
	}
	err = listApp.Run([]string{"devkit", "avs", "devnet", "list"})
	assert.NoError(t, err)

	// Restore stdout and capture buffer
	w.Close()
	os.Stdout = originalStdout

	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "devkit-devnet-", "Expected container name in output")
	assert.Contains(t, output, fmt.Sprintf("http://localhost:%s", port), "Expected devnet URL in output")

	// Stop devnet
	stopApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: StopDevnetAction,
	}
	err = stopApp.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)
}
