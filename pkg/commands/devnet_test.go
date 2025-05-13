package commands

import (
	"bytes"
	"devkit-cli/pkg/common"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

func createTempAVSProject(t *testing.T, defaultConfigDir string) (string, error) {
	tempDir, err := os.MkdirTemp("", "devkit-test-avs-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Create config/ directory
	destConfigDir := filepath.Join(tempDir, "config")
	if err := os.MkdirAll(destConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}

	// Copy config.yaml
	srcConfigFile := filepath.Join(defaultConfigDir, "config.yaml")
	destConfigFile := filepath.Join(destConfigDir, "config.yaml")

	common.CopyFileTesting(t, srcConfigFile, destConfigFile)

	// Create config/contexts directory
	destContextsDir := filepath.Join(destConfigDir, "contexts")
	if err := os.MkdirAll(destContextsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config/contexts dir: %w", err)
	}

	// Copy devnet.yaml context file
	srcDevnetFile := filepath.Join(defaultConfigDir, "contexts", "devnet.yaml")
	destDevnetFile := filepath.Join(destContextsDir, "devnet.yaml")

	common.CopyFileTesting(t, srcDevnetFile, destDevnetFile)

	return tempDir, nil
}

func TestStartAndStopDevnet(t *testing.T) {
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd)
	})
	defaultBaseConfigPath := filepath.Join("..", "..", "config")

	projectDir, err := createTempAVSProject(t, defaultBaseConfigPath)
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

	defaultBaseConfigPath := filepath.Join("..", "..", "config")

	projectDir1, err := createTempAVSProject(t, defaultBaseConfigPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir1)

	projectDir2, err := createTempAVSProject(t, defaultBaseConfigPath)
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
	defaultBaseConfigPath := filepath.Join("..", "..", "config")
	projectDir, err := createTempAVSProject(t, defaultBaseConfigPath)
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

func TestStopDevnetAll(t *testing.T) {
	// Save working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Prepare and start multiple devnets
	defaultBaseConfigPath, _ := filepath.Abs(filepath.Join("..", "..", "config"))

	for i := 0; i < 2; i++ {
		projectDir, err := createTempAVSProject(t, defaultBaseConfigPath)
		assert.NoError(t, err)
		defer os.RemoveAll(projectDir)

		err = os.Chdir(projectDir)
		assert.NoError(t, err)

		port, err := getFreePort()
		assert.NoError(t, err)

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
	}

	// Top-level CLI app simulating full command: devkit avs devnet stop --all
	devkitApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					{
						Name:        "devnet",
						Subcommands: []*cli.Command{DevnetCommand.Subcommands[1]}, // stop
					},
				},
			},
		},
	}

	err = devkitApp.Run([]string{"devkit", "avs", "devnet", "stop", "--all"})
	assert.NoError(t, err)

	// Verify no devnet containers are running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := cmd.Output()
	assert.NoError(t, err)

	assert.NotContains(t, string(output), "devkit-devnet-", "All devnet containers should be stopped")
}

func TestStopDevnetContainerFlag(t *testing.T) {
	// Save working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Prepare and start multiple devnets
	defaultBaseConfigPath, _ := filepath.Abs(filepath.Join("..", "..", "config"))

	projectDir, err := createTempAVSProject(t, defaultBaseConfigPath)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

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

	devkitApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					{
						Name:        "devnet",
						Subcommands: []*cli.Command{DevnetCommand.Subcommands[1]}, // stop
					},
				},
			},
		},
	}

	err = devkitApp.Run([]string{"devkit", "avs", "devnet", "stop", "--project.name", "my-avs"})
	assert.NoError(t, err)

	// Verify no devnet containers are running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := cmd.Output()
	assert.NoError(t, err)
	assert.NotContains(t, string(output), "devkit-devnet-", "The devnet container should be stopped")
}
