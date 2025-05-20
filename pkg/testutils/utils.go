package testutils

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

type ctxKey string

// ConfigContextKey identifies the ConfigWithContextConfig in context
const ConfigContextKey ctxKey = "ConfigWithContextConfig"

func WithTestConfig(cmd *cli.Command) *cli.Command {
	cmd.Before = func(cCtx *cli.Context) error {
		cfg := &common.ConfigWithContextConfig{
			Config: common.ConfigBlock{
				Project: common.ProjectConfig{
					Name: "test-avs",
				},
			},
		}
		ctx := context.WithValue(cCtx.Context, ConfigContextKey, cfg)
		cCtx.Context = ctx
		return nil
	}
	return cmd
}

// helper to create a temp AVS project dir with config.yaml copied
func CreateTempAVSProject(t *testing.T) (string, error) {
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
	err = os.WriteFile(destConfigFile, []byte(configs.ConfigYamls[configs.LatestVersion]), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to copy %s: %w", common.BaseConfig, err)
	}

	// Create config/contexts directory
	destContextsDir := filepath.Join(destConfigDir, "contexts")
	if err := os.MkdirAll(destContextsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config/contexts dir: %w", err)
	}

	// Set fork_urls as envs
	os.Setenv("L1_FORK_URL", "https://eth.llamarpc.com")
	os.Setenv("L2_FORK_URL", "https://eth.llamarpc.com")

	// Copy devnet.yaml context file
	destDevnetFile := filepath.Join(destContextsDir, "devnet.yaml")
	err = os.WriteFile(destDevnetFile, []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644)
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
	run := `#!/bin/bash
echo '{"mock": "run"}'`
	if err := os.WriteFile(filepath.Join(scriptsDir, "run"), []byte(run), 0755); err != nil {
		t.Fatal(err)
	}
	call := `#!/bin/bash
echo '{"mock": "call"}'`
	if err := os.WriteFile(filepath.Join(scriptsDir, "call"), []byte(call), 0755); err != nil {
		t.Fatal(err)
	}

	return tempDir, nil
}

func FindSubcommandByName(name string, commands []*cli.Command) *cli.Command {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}

func CaptureOutput(fn func()) (stdout string, stderr string) {
	// Get the logger
	log, _ := common.GetLogger()

	// Capture stdout
	origStdout := os.Stdout
	origStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rOut); err != nil {
			log.Warn("failed to read stdout: %v", err)
		}
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rErr); err != nil {
			log.Warn("failed to read stdout: %v", err)
		}
		errC <- buf.String()
	}()

	// Run target code
	fn()

	// Restore
	wOut.Close()
	wErr.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	stdout = <-outC
	stderr = <-errC

	return stdout, stderr
}
