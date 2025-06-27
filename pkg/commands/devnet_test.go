package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

func TestStartAndStopDevnet(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd)
	})

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start
	startApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	// Stop
	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
	}, StopDevnetAction)

	err = stopApp.Run([]string{"devkit", "--port", port, "--verbose"})
	assert.NoError(t, err)
}

func TestStartDevnetOnUsedPort_ShouldFail(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd) // Restore cwd after test
	})
	projectDir1, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir1)

	projectDir2, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir2)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start from dir1
	err = os.Chdir(projectDir1)
	assert.NoError(t, err)

	app1, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	err = app1.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	// Attempt from dir2
	err = os.Chdir(projectDir2)
	assert.NoError(t, err)

	app2, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	err = app2.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts", "--skip-transporter"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in use")

	// Cleanup from dir1
	err = os.Chdir(projectDir1)
	assert.NoError(t, err)

	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	_ = stopApp.Run([]string{"devkit", "--port", port})
}

func TestStartDevnet_WithDeployContracts(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	app, logger := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
		&cli.BoolFlag{Name: "skip-setup"},
	}, StartDevnetAction)

	// Use --skip-setup to avoid AVS setup steps while still deploying contracts
	err = app.Run([]string{"devkit", "--port", port, "--skip-setup", "--skip-transporter"})
	assert.NoError(t, err)

	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "getOperatorRegistrationMetadata", ctx["mock"], "deployContracts should run by default")
	assert.True(t, logger.Contains("Offchain AVS components started successfully"), "AVSRun should run by default")

	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	_ = stopApp.Run([]string{"devkit", "--port", port})
}

func TestStartDevnet_SkipDeployContracts(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	app, logger := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	err = app.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEqual(t, "run", ctx["mock"], "avs run should run by default")
	assert.NotEqual(t, "getOperatorRegistrationMetadata", ctx["mock"], "deployContracts should be skipped")
	assert.False(t, logger.Contains("Offchain AVS components started successfully"), "AVSRun should be skipped")

	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	_ = stopApp.Run([]string{"devkit", "--port", port})
}

func TestStartDevnet_SkipAVSRun(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	app, logger := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-setup"},
		&cli.BoolFlag{Name: "skip-transporter"},
		&cli.BoolFlag{Name: "skip-avs-run"},
	}, StartDevnetAction)

	err = app.Run([]string{"devkit", "--port", port, "--skip-setup", "--skip-transporter", "--skip-avs-run"})
	assert.NoError(t, err)

	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "getOperatorRegistrationMetadata", ctx["mock"], "deployContracts should not be skipped")
	assert.False(t, logger.Contains("Offchain AVS components started successfully"), "AVSRun should be skipped")

	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	_ = stopApp.Run([]string{"devkit", "--port", port})
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
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save original working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Prepare temp AVS project
	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start devnet
	startApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)
	err = startApp.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	// Capture output of list
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	listApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{}, ListDevnetContainersAction)
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
	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	err = stopApp.Run([]string{"devkit", "--port", port})
	assert.NoError(t, err)
}

func TestStopDevnetAll(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save cwd
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Start two different devnets in different directories
	for i := 0; i < 2; i++ {
		projectDir, err := testutils.CreateTempAVSProject(t)
		assert.NoError(t, err)
		defer os.RemoveAll(projectDir)

		err = os.Chdir(projectDir)
		assert.NoError(t, err)

		port, err := getFreePort()
		assert.NoError(t, err)

		startApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
			&cli.BoolFlag{Name: "skip-transporter"},
		}, StartDevnetAction)

		err = startApp.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts", "--skip-transporter"})
		assert.NoError(t, err)
	}

	// Create stop command with no-op logger
	stopCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(testutils.FindSubcommandByName("stop", DevnetCommand.Subcommands))

	// Top-level CLI app simulating full command: devkit avs devnet stop --all
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

	// Verify no devnet containers are running
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := cmd.Output()
	assert.NoError(t, err)

	assert.NotContains(t, string(output), "devkit-devnet-", "All devnet containers should be stopped")
}

func TestStopDevnetContainerFlag(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	startApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	err = startApp.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	// Create stop command with no-op logger
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
						},
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

func TestDeployContracts(t *testing.T) {
	os.Setenv("SKIP_DEVNET_FUNDING", "true")
	// Save working dir
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Setup temp project
	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start devnet first
	startApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)
	err = startApp.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts", "--skip-transporter"})
	assert.NoError(t, err)

	// Run deploy-contracts
	deployApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{}, DeployContractsAction)

	err = deployApp.Run([]string{"devkit", "avs", "devnet", "deploy-contracts"})
	assert.NoError(t, err)

	// Read and verify context output
	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "context")

	// Unmarshal the context file
	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	// Expect the context to be present
	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok, "expected context map in devnet.yaml")

	// Expect getOperatorRegistrationMetadata to be written to mock
	mockVal := ctx["mock"]
	assert.Equal(t, "getOperatorRegistrationMetadata", mockVal)

	// Cleanup
	stopApp, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
	}, StopDevnetAction)
	err = stopApp.Run([]string{"devkit", "--port", port})
	assert.NoError(t, err)
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

				// ABI should match what we wrote
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
				// return a DeployContractOutput with a non-existent ABIPath
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
			t.Cleanup(func() { _ = os.Chdir(originalCwd) })

			// each test in its own temp workspace
			baseDir := t.TempDir()
			require.NoError(t, os.Chdir(baseDir))

			contractsList, err := tc.setup(baseDir)
			require.NoError(t, err)

			// Create CLI context with no-op logger
			app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("test", []cli.Flag{}, func(c *cli.Context) error { return nil })
			cCtx := cli.NewContext(app, nil, nil)

			// Execute the Before hook to set up the logger context
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

func TestStartDevnet_ContextCancellation(t *testing.T) {
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := testutils.CreateTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	app, _ := testutils.CreateTestAppWithNoopLoggerAndAccess("devkit", []cli.Flag{
		&cli.IntFlag{Name: "port"},
		&cli.BoolFlag{Name: "verbose"},
		&cli.BoolFlag{Name: "skip-deploy-contracts"},
		&cli.BoolFlag{Name: "skip-transporter"},
	}, StartDevnetAction)

	done := make(chan error, 1)
	go func() {
		args := []string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts", "--skip-transporter"}
		done <- app.RunContext(ctx, args)
	}()

	cancel()

	select {
	case err = <-done:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Log("StartDevnetAction exited cleanly after context cancellation")
		} else {
			t.Errorf("StartDevnetAction returned with error after context cancellation: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("StartDevnetAction did not exit after context cancellation")
	}
}

// Zeus is not being used temporarily.
// func TestStartDevnet_UseZeus(t *testing.T) {
// 	os.Setenv("SKIP_DEVNET_FUNDING", "true")
// 	originalCwd, err := os.Getwd()
// 	assert.NoError(t, err)
// 	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

// 	projectDir, err := testutils.CreateTempAVSProject(t)
// 	assert.NoError(t, err)
// 	defer os.RemoveAll(projectDir)

// 	err = os.Chdir(projectDir)
// 	assert.NoError(t, err)

// 	port, err := getFreePort()
// 	assert.NoError(t, err)

// 	app := &cli.App{
// 		Name: "devkit",
// 		Flags: []cli.Flag{
// 			&cli.IntFlag{Name: "port"},
// 			&cli.BoolFlag{Name: "verbose"},
// 			&cli.BoolFlag{Name: "skip-deploy-contracts"},
// 			&cli.BoolFlag{Name: "skip-transporter"},
// 			&cli.BoolFlag{Name: "use-zeus"},
// 		},
// 		Action: StartDevnetAction,
// 	}

// 	var stdOut bytes.Buffer

// 	originalStderr := os.Stderr
// 	originalStdout := os.Stdout

// 	r, w, _ := os.Pipe()
// 	os.Stdout = w
// 	os.Stderr = w

// 	err = app.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts", "--skip-transporter", "--use-zeus"})
// 	// Check error is nil
// 	assert.NoError(t, err, "Running devnet with --use-zeus flag should not produce an error")

// 	w.Close()
// 	os.Stdout = originalStdout
// 	os.Stderr = originalStderr

// 	_, err = io.Copy(&stdOut, r)
// 	assert.NoError(t, err)

// 	// Check output
// 	output := stdOut.String()
// 	assert.Contains(t, output, "zeus", "Output should mention zeus when --use-zeus flag is used")
// 	assert.NotContains(t, output, "error", "Output should not contain error messages")

// 	stopApp := &cli.App{
// 		Name:   "devkit",
// 		Flags:  []cli.Flag{&cli.IntFlag{Name: "port"}},
// 		Action: StopDevnetAction,
// 	}
// 	_ = stopApp.Run([]string{"devkit", "--port", port})
// }
