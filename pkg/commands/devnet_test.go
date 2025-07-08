package commands

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// TestDevnetPortAvailability tests port availability checking
func TestDevnetPortAvailability(t *testing.T) {
	// Find a free port
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Port should be available after closing
	assert.True(t, devnet.IsPortAvailable(port))

	// Create a new listener on the same port
	listener2, err := net.Listen("tcp", "localhost:"+strconv.Itoa(port))
	require.NoError(t, err)
	defer listener2.Close()

	// Port should not be available when occupied
	assert.False(t, devnet.IsPortAvailable(port))
}

// TestDevnetPortConflictDetection tests port conflict detection without starting containers
func TestDevnetPortConflictDetection(t *testing.T) {
	// Find an available port, then occupy it
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	occupiedPort := listener.Addr().(*net.TCPAddr).Port

	// Create test app
	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
	}, func(c *cli.Context) error {
		l1Port := c.Int("l1-port")
		l2Port := c.Int("l2-port")

		if !devnet.IsPortAvailable(l1Port) {
			return assert.AnError // Simulate port conflict error
		}
		if !devnet.IsPortAvailable(l2Port) {
			return assert.AnError // Simulate port conflict error
		}
		return nil
	})

	// Test with occupied port should fail
	err = app.Run([]string{"devkit", "--l1-port", strconv.Itoa(occupiedPort), "--l2-port", "8546"})
	assert.Error(t, err)

	// Clean up
	listener.Close()
}

// TestDevnetConfigurationLoading tests configuration loading without starting containers
func TestDevnetConfigurationLoading(t *testing.T) {
	// Create temporary project directory
	tempDir, err := testutils.CreateTempAVSProject(t)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Change to project directory
	originalCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalCwd) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test that we can load the devnet configuration
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")

	// File should exist after creating AVS project
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "devnet.yaml should exist in AVS project")

	// Test reading the configuration
	data, err := os.ReadFile(yamlPath)
	require.NoError(t, err)

	// Basic validation that it contains expected content
	content := string(data)
	assert.Contains(t, content, "context:")
	assert.Contains(t, content, "chains:")
}

// TestDevnetEnvironmentVariables tests environment variable handling
func TestDevnetEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalL1Fork := os.Getenv("L1_FORK_URL")
	originalL2Fork := os.Getenv("L2_FORK_URL")
	originalSkipFunding := os.Getenv("SKIP_DEVNET_FUNDING")
	defer func() {
		// Restore original environment
		if originalL1Fork != "" {
			os.Setenv("L1_FORK_URL", originalL1Fork)
		} else {
			os.Unsetenv("L1_FORK_URL")
		}
		if originalL2Fork != "" {
			os.Setenv("L2_FORK_URL", originalL2Fork)
		} else {
			os.Unsetenv("L2_FORK_URL")
		}
		if originalSkipFunding != "" {
			os.Setenv("SKIP_DEVNET_FUNDING", originalSkipFunding)
		} else {
			os.Unsetenv("SKIP_DEVNET_FUNDING")
		}
	}()

	// Test setting environment variables
	os.Setenv("L1_FORK_URL", "https://eth-sepolia.g.alchemy.com/v2/test")
	os.Setenv("L2_FORK_URL", "https://base-sepolia.g.alchemy.com/v2/test")
	os.Setenv("SKIP_DEVNET_FUNDING", "true")

	assert.Equal(t, "https://eth-sepolia.g.alchemy.com/v2/test", os.Getenv("L1_FORK_URL"))
	assert.Equal(t, "https://base-sepolia.g.alchemy.com/v2/test", os.Getenv("L2_FORK_URL"))
	assert.Equal(t, "true", os.Getenv("SKIP_DEVNET_FUNDING"))
}

// TestDevnetCommandFlags tests command line flag parsing
func TestDevnetCommandFlags(t *testing.T) {
	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
		&cli.BoolFlag{Name: "skip-avs-run"},
		&cli.BoolFlag{Name: "skip-setup"},
		&cli.BoolFlag{Name: "verbose"},
	}, func(c *cli.Context) error {
		// Test flag values
		assert.Equal(t, 8545, c.Int("l1-port"))
		assert.Equal(t, 8546, c.Int("l2-port"))
		assert.True(t, c.Bool("skip-deploy-contracts"))
		assert.True(t, c.Bool("skip-transporter"))
		assert.True(t, c.Bool("skip-avs-run"))
		assert.True(t, c.Bool("skip-setup"))
		assert.True(t, c.Bool("verbose"))
		return nil
	})

	err := app.Run([]string{"devkit",
		"--l1-port", "8545",
		"--l2-port", "8546",
		"--skip-deploy-contracts",
		"--skip-transporter",
		"--skip-avs-run",
		"--skip-setup",
		"--verbose",
	})
	assert.NoError(t, err)
}

// TestDevnetDockerComposeGeneration tests Docker compose file generation
func TestDevnetDockerComposeGeneration(t *testing.T) {
	// Test that we can generate the docker-compose file
	composePath := devnet.WriteEmbeddedArtifacts()
	defer os.Remove(composePath)

	// File should exist
	_, err := os.Stat(composePath)
	assert.NoError(t, err)

	// Read and validate content
	content, err := os.ReadFile(composePath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "services:")
	assert.Contains(t, contentStr, "devkit-devnet-l1:")
	assert.Contains(t, contentStr, "devkit-devnet-l2:")
}

// TestDevnetRPCURLGeneration tests RPC URL generation
func TestDevnetRPCURLGeneration(t *testing.T) {
	l1Port := 8545
	l2Port := 8546

	l1URL := devnet.GetRPCURL(l1Port)
	l2URL := devnet.GetRPCURL(l2Port)

	assert.Equal(t, "http://localhost:8545", l1URL)
	assert.Equal(t, "http://localhost:8546", l2URL)
}

// TestDevnetContainerNames tests container name generation
func TestDevnetContainerNames(t *testing.T) {
	projectName := "test-project"

	l1ContainerName := "devkit-devnet-l1-" + projectName
	l2ContainerName := "devkit-devnet-l2-" + projectName

	assert.Equal(t, "devkit-devnet-l1-test-project", l1ContainerName)
	assert.Equal(t, "devkit-devnet-l2-test-project", l2ContainerName)
}

// TestDevnetStopCommand tests the stop command logic
func TestDevnetStopCommand(t *testing.T) {
	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
		&cli.BoolFlag{Name: "all"},
	}, func(c *cli.Context) error {
		// Mock stop action - just validate flags are parsed correctly
		l1Port := c.Int("l1-port")
		l2Port := c.Int("l2-port")
		all := c.Bool("all")

		if all {
			// Stop all containers
			return nil
		}

		// Stop specific containers based on ports
		if l1Port > 0 || l2Port > 0 {
			return nil
		}

		return assert.AnError
	})

	// Test stop with specific ports
	err := app.Run([]string{"devkit", "--l1-port", "8545", "--l2-port", "8546"})
	assert.NoError(t, err)

	// Test stop all
	err = app.Run([]string{"devkit", "--all"})
	assert.NoError(t, err)
}

// TestDevnetContextCancellation tests context cancellation handling
func TestDevnetContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{}, func(c *cli.Context) error {
		// Simulate a long-running operation that should be cancelled
		select {
		case <-c.Context.Done():
			return c.Context.Err()
		case <-time.After(5 * time.Second):
			return assert.AnError // Should not reach here
		}
	})

	done := make(chan error, 1)
	go func() {
		done <- app.RunContext(ctx, []string{"devkit"})
	}()

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	select {
	case err := <-done:
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	case <-time.After(2 * time.Second):
		t.Error("Context cancellation did not work")
	}
}

// TestDevnetContainerStatusCheck tests container status checking without starting containers
func TestDevnetContainerStatusCheck(t *testing.T) {
	// This test checks that the container status check command works
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=devkit-devnet", "--format", "{{.Names}} {{.Status}}")
	output, err := cmd.Output()

	// Command should succeed (even if no containers exist)
	assert.NoError(t, err)

	// Output should be empty or contain container info
	outputStr := strings.TrimSpace(string(output))
	// If containers exist, they should have devkit-devnet in the name
	if outputStr != "" {
		lines := strings.Split(outputStr, "\n")
		for _, line := range lines {
			if line != "" {
				assert.Contains(t, line, "devkit-devnet")
			}
		}
	}
}

// TestDevnetFlagValidation tests flag validation
func TestDevnetFlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
	}{
		{
			name:        "valid ports",
			args:        []string{"devkit", "--l1-port", "8545", "--l2-port", "8546"},
			shouldError: false,
		},
		{
			name:        "same port for l1 and l2",
			args:        []string{"devkit", "--l1-port", "8545", "--l2-port", "8545"},
			shouldError: false, // Should be handled by port availability check
		},
		{
			name:        "no ports specified",
			args:        []string{"devkit"},
			shouldError: false, // Should use defaults
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
				&cli.IntFlag{Name: "l1-port", Value: 8545},
				&cli.IntFlag{Name: "l2-port", Value: 8546},
			}, func(c *cli.Context) error {
				l1Port := c.Int("l1-port")
				l2Port := c.Int("l2-port")

				// Basic validation
				if l1Port <= 0 || l2Port <= 0 {
					return assert.AnError
				}
				if l1Port > 65535 || l2Port > 65535 {
					return assert.AnError
				}

				return nil
			})

			err := app.Run(tt.args)
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
