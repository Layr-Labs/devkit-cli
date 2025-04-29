package commands

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// helper to create a temp AVS project dir with eigen.toml copied
func createTempAVSProject(defaultEigenPath string) (string, error) {
	tempDir, err := os.MkdirTemp("", "devkit-test-avs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	destEigen := filepath.Join(tempDir, "eigen.toml")

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
	defaultEigenPath := filepath.Join("..", "..", "default.eigen.toml")

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
