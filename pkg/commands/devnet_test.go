package commands

import (
	"bytes"
	"context"
	"devkit-cli/config"
	"devkit-cli/pkg/common"
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

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"sigs.k8s.io/yaml"
)

// helper to create a temp AVS project dir with config.yaml copied
func createTempAVSProject(t *testing.T) (string, error) {
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
	destConfigFile := filepath.Join(destConfigDir, common.BaseConfig)
	err = os.WriteFile(destConfigFile, []byte(config.DefaultConfigYaml), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to copy %s: %w", common.BaseConfig, err)
	}

	// Create config/contexts directory
	destContextsDir := filepath.Join(destConfigDir, "contexts")
	if err := os.MkdirAll(destContextsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config/contexts dir: %w", err)
	}

	// Copy devnet.yaml context file
	destDevnetFile := filepath.Join(destContextsDir, "devnet.yaml")
	err = os.WriteFile(destDevnetFile, []byte(config.ContextYamls["devnet"]), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create config/contexts/devnet.yaml: %w", err)
	}

	// Create build script
	scriptsDir := filepath.Join(tempDir, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	deployScript := `#!/bin/bash
echo '{"mock": "deployContracts"}'`
	if err := os.WriteFile(filepath.Join(scriptsDir, "deployContracts"), []byte(deployScript), 0755); err != nil {
		t.Fatal(err)
	}
	getOperatorSets := `#!/bin/bash
echo '{"mock": "getOperatorSets"}'`
	if err := os.WriteFile(filepath.Join(scriptsDir, "getOperatorSets"), []byte(getOperatorSets), 0755); err != nil {
		t.Fatal(err)
	}
	getOperatorRegistrationMetadata := `#!/bin/bash
echo '{"mock": "getOperatorRegistrationMetadata"}'`
	if err := os.WriteFile(filepath.Join(scriptsDir, "getOperatorRegistrationMetadata"), []byte(getOperatorRegistrationMetadata), 0755); err != nil {
		t.Fatal(err)
	}

	return tempDir, nil
}

func findSubcommandByName(name string, commands []*cli.Command) *cli.Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func TestStartAndStopDevnet(t *testing.T) {
	// Save current working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() {
		_ = os.Chdir(originalCwd)
	})

	projectDir, err := createTempAVSProject(t)
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
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}

	err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
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
	projectDir1, err := createTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir1)

	projectDir2, err := createTempAVSProject(t)
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
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}
	err = app1.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
	assert.NoError(t, err)

	// Attempt from dir2
	err = os.Chdir(projectDir2)
	assert.NoError(t, err)

	app2 := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}
	err = app2.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
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
func TestStartDevnet_WithDeployContracts(t *testing.T) {
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := createTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	app := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}

	err = app.Run([]string{"devkit", "--port", port})
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

	stopApp := &cli.App{
		Name:   "devkit",
		Flags:  []cli.Flag{&cli.IntFlag{Name: "port"}},
		Action: StopDevnetAction,
	}
	_ = stopApp.Run([]string{"devkit", "--port", port})
}

func TestStartDevnet_SkipDeployContracts(t *testing.T) {
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := createTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	app := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}

	err = app.Run([]string{"devkit", "--port", port, "--skip-deploy-contracts"})
	assert.NoError(t, err)

	yamlPath := filepath.Join("config", "contexts", "devnet.yaml")
	data, err := os.ReadFile(yamlPath)
	assert.NoError(t, err)

	var parsed map[string]interface{}
	err = yaml.Unmarshal(data, &parsed)
	assert.NoError(t, err)

	ctx, ok := parsed["context"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotEqual(t, "getOperatorRegistrationMetadata", ctx["mock"], "deployContracts should be skipped")

	stopApp := &cli.App{
		Name:   "devkit",
		Flags:  []cli.Flag{&cli.IntFlag{Name: "port"}},
		Action: StopDevnetAction,
	}
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
	// Save original working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Prepare temp AVS project
	projectDir, err := createTempAVSProject(t)
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
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}
	err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
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
	for i := 0; i < 2; i++ {
		projectDir, err := createTempAVSProject(t)
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
				&cli.BoolFlag{Name: "skip-deploy-contracts"},
			},
			Action: StartDevnetAction,
		}

		err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
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
						Name: "devnet",
						Subcommands: []*cli.Command{
							findSubcommandByName("stop", DevnetCommand.Subcommands),
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
	// Save working directory
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := createTempAVSProject(t)
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
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}

	err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
	assert.NoError(t, err)

	devkitApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					{
						Name: "devnet",
						Subcommands: []*cli.Command{
							findSubcommandByName("stop", DevnetCommand.Subcommands),
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
	// Save working dir
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	// Setup temp project
	projectDir, err := createTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	// Start devnet first
	startApp := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}
	err = startApp.Run([]string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"})
	assert.NoError(t, err)

	// Run deploy-contracts
	deployApp := &cli.App{
		Name:   "devkit",
		Action: DeployContractsAction,
	}
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

func TestStartDevnet_ContextCancellation(t *testing.T) {
	originalCwd, err := os.Getwd()
	assert.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(originalCwd) })

	projectDir, err := createTempAVSProject(t)
	assert.NoError(t, err)
	defer os.RemoveAll(projectDir)

	err = os.Chdir(projectDir)
	assert.NoError(t, err)

	port, err := getFreePort()
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	app := &cli.App{
		Name: "devkit",
		Flags: []cli.Flag{
			&cli.IntFlag{Name: "port"},
			&cli.BoolFlag{Name: "verbose"},
			&cli.BoolFlag{Name: "skip-deploy-contracts"},
		},
		Action: StartDevnetAction,
	}

	done := make(chan error, 1)
	go func() {
		args := []string{"devkit", "--port", port, "--verbose", "--skip-deploy-contracts"}
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