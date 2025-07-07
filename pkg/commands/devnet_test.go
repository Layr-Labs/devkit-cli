package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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
	"sigs.k8s.io/yaml"
)

// Test modes
const (
	TestModeBasic       = "basic"       // Skip funding, basic container operations
	TestModeIntegration = "integration" // Full integration with funding
	TestModeMock        = "mock"        // Mock external dependencies
)

// Test configuration
type TestConfig struct {
	Mode             string
	SkipFunding      bool
	SkipContracts    bool
	SkipTransporter  bool
	SkipAVSRun       bool
	Timeout          time.Duration
	RetryAttempts    int
	L1NetworkForkURL string // L1 fork URL
	L2NetworkForkURL string // L2 fork URL
	UseRealNetworks  bool
}

// Default test configurations
var (
	BasicTestConfig = TestConfig{
		Mode:             TestModeBasic,
		SkipFunding:      true,
		SkipContracts:    true,
		SkipTransporter:  true,
		SkipAVSRun:       true,
		Timeout:          30 * time.Second,
		RetryAttempts:    3,
		L1NetworkForkURL: devnet.DEFAULT_L1_FORK_URL,
		L2NetworkForkURL: devnet.DEFAULT_L2_FORK_URL,
		UseRealNetworks:  false, // Basic tests skip integration features but still need network URLs
	}

	IntegrationTestConfig = TestConfig{
		Mode:             TestModeIntegration,
		SkipFunding:      false,
		SkipContracts:    false,
		SkipTransporter:  false,
		SkipAVSRun:       false,
		Timeout:          300 * time.Second,
		RetryAttempts:    2,
		L1NetworkForkURL: devnet.DEFAULT_L1_FORK_URL,
		L2NetworkForkURL: devnet.DEFAULT_L2_FORK_URL,
		UseRealNetworks:  true,
	}

	MockTestConfig = TestConfig{
		Mode:            TestModeMock,
		SkipFunding:     true,
		SkipContracts:   false,
		SkipTransporter: true,
		SkipAVSRun:      false,
		Timeout:         60 * time.Second,
		RetryAttempts:   2,
	}
)

// TestDevnetSuite
type TestDevnetSuite struct {
	t           *testing.T
	config      TestConfig
	projectDir  string
	originalCwd string
	cleanup     []func()
	l1Port      int
	l2Port      int
}

// NewTestDevnetSuite creates a new devnet test suite
func NewTestDevnetSuite(t *testing.T, config TestConfig) *TestDevnetSuite {
	suite := &TestDevnetSuite{
		t:       t,
		config:  config,
		cleanup: make([]func(), 0),
	}

	// Setup test environment
	suite.setupTestEnvironment()
	return suite
}

// setupTestEnvironment initializes the test environment
func (s *TestDevnetSuite) setupTestEnvironment() {
	// Save original working directory
	var err error
	s.originalCwd, err = os.Getwd()
	require.NoError(s.t, err)

	// Create temporary project directory
	s.projectDir, err = testutils.CreateTempAVSProject(s.t)
	require.NoError(s.t, err)

	// Change to project directory
	err = os.Chdir(s.projectDir)
	require.NoError(s.t, err)

	// Get free ports for testing
	s.l1Port, err = s.getFreePortInt()
	require.NoError(s.t, err)
	s.l2Port, err = s.getFreePortInt()
	require.NoError(s.t, err)

	// Setup environment variables based on test mode
	s.setupEnvironmentVariables()

	// Add cleanup functions
	s.addCleanup(func() {
		_ = os.Chdir(s.originalCwd)
		_ = os.RemoveAll(s.projectDir)
	})
}

// setupEnvironmentVariables configures environment variables for the test mode
func (s *TestDevnetSuite) setupEnvironmentVariables() {
	s.t.Logf("Test mode: %s, UseRealNetworks: %v, SkipFunding: %v", s.config.Mode, s.config.UseRealNetworks, s.config.SkipFunding)

	// For L1_FORK_URL: use environment variable if set, otherwise use config default
	if existingL1URL := os.Getenv("L1_FORK_URL"); existingL1URL != "" {
		s.t.Logf("Using existing L1_FORK_URL from environment: %s", existingL1URL)
	} else if s.config.L1NetworkForkURL != "" {
		os.Setenv("L1_FORK_URL", s.config.L1NetworkForkURL)
		s.t.Logf("Setting L1_FORK_URL to config default: %s", s.config.L1NetworkForkURL)
	}

	// For L2_FORK_URL: use environment variable if set, otherwise use config default
	if existingL2URL := os.Getenv("L2_FORK_URL"); existingL2URL != "" {
		s.t.Logf("Using existing L2_FORK_URL from environment: %s", existingL2URL)
	} else if s.config.L2NetworkForkURL != "" {
		os.Setenv("L2_FORK_URL", s.config.L2NetworkForkURL)
		s.t.Logf("Setting L2_FORK_URL to config default: %s", s.config.L2NetworkForkURL)
	}

	// Set funding-related environment variables
	if s.config.SkipFunding {
		os.Setenv("SKIP_DEVNET_FUNDING", "true")
		os.Setenv("SKIP_TOKEN_FUNDING", "true")
		s.t.Logf("Set SKIP_DEVNET_FUNDING and SKIP_TOKEN_FUNDING to true")
	} else {
		os.Unsetenv("SKIP_DEVNET_FUNDING")
		os.Unsetenv("SKIP_TOKEN_FUNDING")
		s.t.Logf("Unset SKIP_DEVNET_FUNDING and SKIP_TOKEN_FUNDING")
	}

	s.t.Logf("Current environment - L1_FORK_URL: %s, L2_FORK_URL: %s", os.Getenv("L1_FORK_URL"), os.Getenv("L2_FORK_URL"))
}

// addCleanup adds a cleanup function to be called when the test finishes
func (s *TestDevnetSuite) addCleanup(fn func()) {
	s.cleanup = append(s.cleanup, fn)
}

// Cleanup runs all cleanup functions
func (s *TestDevnetSuite) Cleanup() {
	for i := len(s.cleanup) - 1; i >= 0; i-- {
		s.cleanup[i]()
	}
}

// getFreePortInt finds an available TCP port and returns it as an integer
func (s *TestDevnetSuite) getFreePortInt() (int, error) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// waitForDevnetReady waits for the devnet to be ready by checking RPC endpoints
func (s *TestDevnetSuite) waitForDevnetReady() error {
	l1URL := fmt.Sprintf("http://localhost:%d", s.l1Port)
	l2URL := fmt.Sprintf("http://localhost:%d", s.l2Port)

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	// Check L1 endpoint
	if err := s.waitForRPCEndpoint(ctx, l1URL); err != nil {
		return fmt.Errorf("L1 endpoint not ready: %w", err)
	}

	// Check L2 endpoint
	if err := s.waitForRPCEndpoint(ctx, l2URL); err != nil {
		return fmt.Errorf("L2 endpoint not ready: %w", err)
	}

	return nil
}

// checkContainerStatus checks if containers failed to start due to port conflicts
func (s *TestDevnetSuite) checkContainerStatus() error {
	// Check for failed containers with port conflict errors
	cmd := exec.Command("docker", "ps", "-a", "--filter", "name=devkit-devnet", "--format", "{{.Names}} {{.Status}}")
	output, err := cmd.Output()
	if err != nil {
		return nil // Ignore errors, this is just a check
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Exited") {
			// Get container logs to check for port conflicts
			parts := strings.Fields(line)
			if len(parts) > 0 {
				containerName := parts[0]
				logCmd := exec.Command("docker", "logs", containerName)
				logOutput, logErr := logCmd.Output()
				if logErr == nil {
					logStr := string(logOutput)
					if strings.Contains(logStr, "bind: address already in use") ||
						strings.Contains(logStr, "port is already allocated") ||
						strings.Contains(logStr, "already in use") {
						return fmt.Errorf("port conflict detected: bind: address already in use")
					}
				}
			}
		}
	}
	return nil
}

// waitForRPCEndpoint waits for an RPC endpoint to be available
func (s *TestDevnetSuite) waitForRPCEndpoint(ctx context.Context, url string) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Try to make a simple RPC call
			payload := strings.NewReader(`{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`)
			req, err := http.NewRequestWithContext(ctx, "POST", url, payload)
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{Timeout: 2 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// startDevnet starts the devnet with specified configuration
func (s *TestDevnetSuite) startDevnet() error {
	// The StartDevnetAction will detect and report port conflicts properly

	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
		&cli.BoolFlag{Name: "skip-avs-run"},
		&cli.BoolFlag{Name: "skip-setup"},
	}, StartDevnetAction)

	args := []string{"devkit",
		"--l1-port", strconv.Itoa(s.l1Port),
		"--l2-port", strconv.Itoa(s.l2Port),
		"--verbose",
	}

	if s.config.SkipContracts {
		args = append(args, "--skip-deploy-contracts")
	}
	if s.config.SkipTransporter {
		args = append(args, "--skip-transporter")
	}
	if s.config.SkipAVSRun {
		args = append(args, "--skip-avs-run")
	}
	if s.config.SkipContracts {
		args = append(args, "--skip-setup")
	}

	// Start devnet in background
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- app.RunContext(ctx, args)
	}()

	// Wait  for containers to start, then check for port conflicts
	time.Sleep(2 * time.Second)

	// Check if containers failed due to port conflicts
	if err := s.checkContainerStatus(); err != nil {
		return err
	}

	// Wait for devnet to be ready
	if err := s.waitForDevnetReady(); err != nil {
		// Check if this is a port conflict masquerading as a timeout
		if strings.Contains(err.Error(), "context deadline exceeded") {
			if portErr := s.checkContainerStatus(); portErr != nil {
				return portErr
			}
		}
		return fmt.Errorf("devnet not ready: %w", err)
	}

	// cleanup to stop devnet
	s.addCleanup(func() {
		_ = s.stopDevnet()
	})

	return nil
}

// stopDevnet stops the devnet
func (s *TestDevnetSuite) stopDevnet() error {
	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
	}, StopDevnetAction)

	return app.Run([]string{"devkit", "--l1-port", strconv.Itoa(s.l1Port), "--l2-port", strconv.Itoa(s.l2Port)})
}

// verifyDevnetState verifies that the devnet is in the expected state
func (s *TestDevnetSuite) verifyDevnetState() error {
	// Check if containers are running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check containers: %w", err)
	}

	containerNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(containerNames) < 2 {
		return fmt.Errorf("expected at least 2 containers, got %d", len(containerNames))
	}

	// Verify L1 and L2 containers are running
	l1Found := false
	l2Found := false
	for _, name := range containerNames {
		if strings.Contains(name, "l1") {
			l1Found = true
		}
		if strings.Contains(name, "l2") {
			l2Found = true
		}
	}

	if !l1Found {
		return fmt.Errorf("L1 container not found")
	}
	if !l2Found {
		return fmt.Errorf("L2 container not found")
	}

	// Check if context file was created/updated
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	if _, err := os.Stat(yamlPath); err != nil {
		return fmt.Errorf("devnet context file not found: %w", err)
	}

	// Verify context content
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("failed to parse context file: %w", err)
	}

	// Check if context contains expected chains
	context, ok := parsed["context"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("context section not found")
	}

	chains, ok := context["chains"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("chains section not found")
	}

	if _, ok := chains["l1"]; !ok {
		return fmt.Errorf("L1 chain not found in context")
	}
	if _, ok := chains["l2"]; !ok {
		return fmt.Errorf("L2 chain not found in context")
	}

	return nil
}

// Test functions using the new test suite

func TestBasicDevnetOperations(t *testing.T) {
	suite := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite.Cleanup()

	t.Run("StartAndStopDevnet", func(t *testing.T) {
		err := suite.startDevnet()
		require.NoError(t, err, "Failed to start devnet")

		err = suite.verifyDevnetState()
		require.NoError(t, err, "Devnet state verification failed")

		err = suite.stopDevnet()
		require.NoError(t, err, "Failed to stop devnet")
	})
}

func TestDevnetPortConflicts(t *testing.T) {
	// Use fixed ports for testing
	l1Port := 18545
	l2Port := 19545

	// Create first temporary AVS project
	tempDir1, err := testutils.CreateTempAVSProject(t)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir1)

	// Create second temporary AVS project
	tempDir2, err := testutils.CreateTempAVSProject(t)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir2)

	// Save original directory
	originalCwd, err := os.Getwd()
	require.NoError(t, err)

	t.Run("PortConflictDetection", func(t *testing.T) {
		// Set up first project
		err = os.Chdir(tempDir1)
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalCwd) }()

		// Start first devnet
		app1, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "l1-port"},
			&cli.IntFlag{Name: "l2-port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
			&cli.BoolFlag{Name: "skip-transporter"},
			&cli.BoolFlag{Name: "skip-avs-run"},
		}, StartDevnetAction)

		args1 := []string{"devkit",
			"--l1-port", strconv.Itoa(l1Port),
			"--l2-port", strconv.Itoa(l2Port),
			"--verbose", "--skip-deploy-contracts", "--skip-transporter", "--skip-avs-run"}

		done1 := make(chan error, 1)
		go func() {
			done1 <- app1.Run(args1)
		}()

		// Wait for first devnet to start
		time.Sleep(4 * time.Second)

		// Check that containers are running
		cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
		_, err := cmd.Output()
		require.NoError(t, err)

		// Set up second project
		err = os.Chdir(tempDir2)
		require.NoError(t, err)

		// Try to start second devnet on same ports
		app2, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "l1-port"},
			&cli.IntFlag{Name: "l2-port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
			&cli.BoolFlag{Name: "skip-transporter"},
			&cli.BoolFlag{Name: "skip-avs-run"},
		}, StartDevnetAction)

		args2 := []string{"devkit",
			"--l1-port", strconv.Itoa(l1Port),
			"--l2-port", strconv.Itoa(l2Port),
			"--verbose", "--skip-deploy-contracts", "--skip-transporter", "--skip-avs-run"}

		err = app2.Run(args2)
		require.Error(t, err, "Second devnet should fail due to port conflict")
		require.True(t,
			strings.Contains(err.Error(), "already in use") ||
				strings.Contains(err.Error(), "port is already allocated") ||
				strings.Contains(err.Error(), "bind: address already in use"),
			"Expected port conflict error, got: %s", err.Error())

		// Clean up first devnet
		err = os.Chdir(tempDir1)
		require.NoError(t, err)

		stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "l1-port"},
			&cli.IntFlag{Name: "l2-port"},
		}, StopDevnetAction)

		_ = stopApp.Run([]string{"devkit", "--l1-port", strconv.Itoa(l1Port), "--l2-port", strconv.Itoa(l2Port)})
	})
}

func TestDevnetIntegrationWithFunding(t *testing.T) {

	suite := NewTestDevnetSuite(t, IntegrationTestConfig)
	defer suite.Cleanup()

	t.Run("FullIntegrationWithFunding", func(t *testing.T) {
		err := suite.startDevnet()
		require.NoError(t, err, "Failed to start devnet with funding")

		err = suite.verifyDevnetState()
		require.NoError(t, err, "Devnet state verification failed")

		// Additional verification for funding
		err = suite.verifyFundingState()
		require.NoError(t, err, "Funding verification failed")
	})
}

func TestDevnetMockMode(t *testing.T) {
	suite := NewTestDevnetSuite(t, MockTestConfig)
	defer suite.Cleanup()

	t.Run("MockModeExecution", func(t *testing.T) {
		err := suite.startDevnet()
		require.NoError(t, err, "Failed to start devnet in mock mode")

		err = suite.verifyDevnetState()
		require.NoError(t, err, "Mock devnet state verification failed")

		// Verify that contracts were deployed (mock mode)
		err = suite.verifyContractDeployment()
		require.NoError(t, err, "Contract deployment verification failed")
	})
}

func TestDevnetContextCancellation(t *testing.T) {
	suite := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite.Cleanup()

	t.Run("GracefulShutdownOnCancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "l1-port"},
			&cli.IntFlag{Name: "l2-port"},
			&cli.BoolFlag{Name: "verbose"},
			// Don't skip contracts/transporter so devnet takes longer to complete
		}, StartDevnetAction)

		done := make(chan error, 1)
		go func() {
			args := []string{"devkit",
				"--l1-port", strconv.Itoa(suite.l1Port),
				"--l2-port", strconv.Itoa(suite.l2Port),
				"--verbose"}
			done <- app.RunContext(ctx, args)
		}()

		// Cancel after giving devnet time to start containers but before completion
		time.Sleep(2 * time.Second)
		cancel()

		select {
		case err := <-done:
			assert.Error(t, err)
		case <-time.After(15 * time.Second):
			t.Error("StartDevnetAction did not exit after context cancellation")
		}
	})
}

// Helper methods for additional verifications

func (s *TestDevnetSuite) verifyFundingState() error {
	// This would check that operator wallets and stakers are properly funded
	// Implementation would depend on the specific funding requirements
	if s.config.SkipFunding {
		return nil
	}

	// Check if operator wallets have sufficient balance
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("failed to parse context file: %w", err)
	}

	// Additional funding verification logic would go here
	return nil
}

func (s *TestDevnetSuite) verifyContractDeployment() error {
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("failed to parse context file: %w", err)
	}

	context, ok := parsed["context"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("context section not found")
	}

	// Check if contracts were deployed
	if s.config.SkipContracts {
		return nil
	}

	eigenLayer, ok := context["eigenlayer"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("eigenlayer section not found")
	}

	l1, ok := eigenLayer["l1"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("L1 section not found")
	}

	// Verify essential contract addresses are present
	requiredContracts := []string{"allocation_manager", "delegation_manager", "strategy_manager"}
	for _, contract := range requiredContracts {
		if _, ok := l1[contract]; !ok {
			return fmt.Errorf("required contract %s not found", contract)
		}
	}

	return nil
}

func TestStartAndStopDevnet(t *testing.T) {
	suite := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite.Cleanup()

	err := suite.startDevnet()
	assert.NoError(t, err)

	err = suite.stopDevnet()
	assert.NoError(t, err)
}

func TestStartDevnetOnUsedPort_ShouldFail(t *testing.T) {
	// Use fixed ports for testing
	l1Port := 18545
	l2Port := 19545

	// Create first temporary AVS project
	tempDir1, err := testutils.CreateTempAVSProject(t)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir1)

	// Create second temporary AVS project
	tempDir2, err := testutils.CreateTempAVSProject(t)
	require.NoError(t, err)
	defer os.RemoveAll(tempDir2)

	// Save original directory
	originalCwd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalCwd) }()

	// Set environment variables for the test
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	os.Setenv("SKIP_TOKEN_FUNDING", "true")
	defer os.Unsetenv("SKIP_DEVNET_FUNDING")
	defer os.Unsetenv("SKIP_TOKEN_FUNDING")

	// Start first devnet
	err = os.Chdir(tempDir1)
	require.NoError(t, err)

	// Start first devnet in background
	ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel1()

	done1 := make(chan error, 1)
	go func() {
		app1, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "l1-port"},
			&cli.IntFlag{Name: "l2-port"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
			&cli.BoolFlag{Name: "skip-transporter"},
			&cli.BoolFlag{Name: "skip-avs-run"},
			&cli.BoolFlag{Name: "skip-setup"},
		}, StartDevnetAction)

		err := app1.RunContext(ctx1, []string{"devkit",
			"--l1-port", strconv.Itoa(l1Port),
			"--l2-port", strconv.Itoa(l2Port),
			"--skip-deploy-contracts",
			"--skip-transporter",
			"--skip-avs-run",
			"--skip-setup",
		})
		done1 <- err
	}()

	// Wait for first devnet to start
	time.Sleep(5 * time.Second)

	// Verify first devnet is running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	_, err = cmd.Output()
	assert.NoError(t, err)

	// Start second devnet (should fail)
	err = os.Chdir(tempDir2)
	require.NoError(t, err)

	app2, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "l1-port"},
		&cli.IntFlag{Name: "l2-port"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
		&cli.BoolFlag{Name: "skip-avs-run"},
		&cli.BoolFlag{Name: "skip-setup"},
	}, StartDevnetAction)

	err = app2.Run([]string{"devkit",
		"--l1-port", strconv.Itoa(l1Port),
		"--l2-port", strconv.Itoa(l2Port),
		"--skip-deploy-contracts",
		"--skip-transporter",
		"--skip-avs-run",
		"--skip-setup",
	})

	// Should get a port conflict error
	assert.Error(t, err)
	if err != nil {
		// Check for either "already in use" or "port is already allocated" messages
		assert.True(t,
			strings.Contains(err.Error(), "already in use") ||
				strings.Contains(err.Error(), "port is already allocated") ||
				strings.Contains(err.Error(), "bind: address already in use"),
			"Expected port conflict error, got: %s", err.Error())
	}

	// Cleanup: stop first devnet
	cancel1()

	// Stop containers manually
	_ = exec.Command("docker", "stop", "devkit-devnet-l1-my-avs").Run()
	_ = exec.Command("docker", "stop", "devkit-devnet-l2-my-avs").Run()
	_ = exec.Command("docker", "rm", "devkit-devnet-l1-my-avs").Run()
	_ = exec.Command("docker", "rm", "devkit-devnet-l2-my-avs").Run()
}

func TestStartDevnet_WithDeployContracts(t *testing.T) {
	config := BasicTestConfig
	config.SkipContracts = false
	config.SkipAVSRun = true

	suite := NewTestDevnetSuite(t, config)
	defer suite.Cleanup()

	err := suite.startDevnet()
	assert.NoError(t, err)

	err = suite.verifyContractDeployment()
	assert.NoError(t, err)
}

func TestStopDevnetAll(t *testing.T) {
	// Start multiple devnets
	suite1 := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite1.Cleanup()

	suite2 := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite2.Cleanup()

	err := suite1.startDevnet()
	assert.NoError(t, err)

	err = suite2.startDevnet()
	assert.NoError(t, err)

	// Stop all
	stopCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(testutils.FindSubcommandByName("stop", DevnetCommand.Subcommands))

	devkitApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					{
						Name: "devnet",
						Subcommands: []*cli.Command{
							stopCmdWithLogger,
						}},
				},
			},
		},
	}

	err = devkitApp.Run([]string{"devkit", "avs", "devnet", "stop", "--all"})
	assert.NoError(t, err)

	// Verify no containers running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := cmd.Output()
	assert.NoError(t, err)
	assert.NotContains(t, string(output), "devkit-devnet-")
}

func TestDeployContracts(t *testing.T) {
	suite := NewTestDevnetSuite(t, BasicTestConfig)
	defer suite.Cleanup()

	// Start devnet without contracts
	err := suite.startDevnet()
	assert.NoError(t, err)

	// Deploy contracts separately
	deployApp, logger := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{}, DeployContractsAction)
	err = deployApp.Run([]string{"devkit"})
	assert.NoError(t, err)

	// Verify deployment
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "getOperatorRegistrationMetadata", ctx["mock"], "deployContracts should run")
	assert.False(t, logger.Contains("Offchain AVS components started successfully"), "Should not start AVS run")
}

func TestDeployContracts_ExtractContractOutputs(t *testing.T) {
	type fixture struct {
		name     string
		setup    func(baseDir string) ([]DeployContractTransport, error)
		context  string
		wantErr  bool
		validate func(t *testing.T, baseDir string)
	}

	tests := []fixture{
		{
			name:    "successfully writes JSON output",
			context: "devnet",
			setup: func(baseDir string) ([]DeployContractTransport, error) {
				abiDir := filepath.Join(baseDir, "artifacts")
				require.NoError(t, os.MkdirAll(abiDir, 0o755))
				abiPath := filepath.Join(abiDir, "MyToken.json")
				rawABI := map[string]interface{}{
					"abi": []interface{}{
						map[string]interface{}{
							"type": "function",
							"name": "balanceOf",
						},
					},
				}
				data, err := json.Marshal(rawABI)
				if err != nil {
					return nil, err
				}
				require.NoError(t, os.WriteFile(abiPath, data, 0o644))

				return []DeployContractTransport{
					{
						Name:    "MyToken",
						Address: "0x1234ABCD",
						ABI:     abiPath,
					},
				}, nil
			},
			wantErr: false,
			validate: func(t *testing.T, baseDir string) {
				outPath := filepath.Join(baseDir, "contracts", "outputs", "devnet", "MyToken.json")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err, "output file must exist and be readable")

				var out DeployContractJson
				require.NoError(t, json.Unmarshal(b, &out), "output JSON must unmarshal")

				require.Equal(t, "MyToken", out.Name)
				require.Equal(t, "0x1234ABCD", out.Address)

				abiSlice, ok := out.ABI.([]interface{})
				require.True(t, ok, "ABI should be a slice")
				require.Len(t, abiSlice, 1)
				entry, ok := abiSlice[0].(map[string]interface{})
				require.True(t, ok)
				require.Equal(t, "balanceOf", entry["name"])
			},
		},
		{
			name:    "error when ABI file missing",
			context: "testnet",
			setup: func(baseDir string) ([]DeployContractTransport, error) {
				return []DeployContractTransport{
					{
						Name:    "NoAbiContract",
						Address: "0xDEADBEEF",
						ABI:     filepath.Join(baseDir, "no_such.json"),
					},
				}, nil
			},
			wantErr: true,
			validate: func(t *testing.T, baseDir string) {
				// no files should be written
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			originalCwd, err := os.Getwd()
			require.NoError(t, err)
			defer func() { _ = os.Chdir(originalCwd) }()

			baseDir := t.TempDir()
			require.NoError(t, os.Chdir(baseDir))

			contractsList, err := tc.setup(baseDir)
			require.NoError(t, err)

			app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("test", []cli.Flag{}, func(c *cli.Context) error { return nil })
			cCtx := cli.NewContext(app, nil, nil)

			if app.Before != nil {
				err := app.Before(cCtx)
				require.NoError(t, err)
			}

			err = extractContractOutputs(cCtx, tc.context, contractsList)
			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "read ABI")
			} else {
				require.NoError(t, err)
				tc.validate(t, baseDir)
			}
		})
	}
}
