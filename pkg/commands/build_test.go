package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"

	"github.com/urfave/cli/v2"
)

func TestBuildCommand(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory and devnet.yaml
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextsDir := filepath.Join(configDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configs.ConfigYamls[configs.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build script
	scriptsDir := filepath.Join(tmpDir, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildScript := `#!/bin/bash
echo "Mock build executed"`
	if err := os.WriteFile(filepath.Join(scriptsDir, "build"), []byte(buildScript), 0755); err != nil {
		t.Fatal(err)
	}

	// Create contracts directory and its Makefile
	contractsDir := filepath.Join(tmpDir, common.ContractsDir)
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatal(err)
	}

	mockContractsMakefile := `
.PHONY: build
build:
	@echo "Mock contracts build executed"
	`
	if err := os.WriteFile(filepath.Join(contractsDir, common.ContractsMakefile), []byte(mockContractsMakefile), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	if err := app.Run([]string{"app", "build", "--context", devnet.DEVNET_CONTEXT}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

// Test the case where contracts directory doesn't exist
func TestBuildCommand_NoContracts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory and devnet.yaml
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextsDir := filepath.Join(configDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configs.ConfigYamls[configs.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build script
	scriptsDir := filepath.Join(tmpDir, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildScript := `#!/bin/bash
echo "Mock build executed"`
	if err := os.WriteFile(filepath.Join(scriptsDir, "build"), []byte(buildScript), 0755); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	if err := app.Run([]string{"app", "build", "--context", devnet.DEVNET_CONTEXT}); err != nil {
		t.Errorf("Failed to execute build command: %v", err)
	}
}

func TestBuildCommand_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory and devnet.yaml
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextsDir := filepath.Join(configDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configs.ConfigYamls[configs.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}

	// Create build script
	scriptsDir := filepath.Join(tmpDir, ".devkit", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		t.Fatal(err)
	}
	buildScript := `#!/bin/bash
echo "Mock build executed"`
	if err := os.WriteFile(filepath.Join(scriptsDir, "build"), []byte(buildScript), 0755); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	parentCtx, cancel := context.WithCancel(context.Background())
	ctx := common.WithShutdown(parentCtx)

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	done := make(chan error, 1)
	go func() {
		done <- app.RunContext(ctx, []string{"app", "build", "--context", devnet.DEVNET_CONTEXT})
	}()

	cancel()

	select {
	case err = <-done:
		if err != nil && errors.Is(err, context.Canceled) {
			t.Logf("Build command exited with error (expected due to context cancel): %v", err)
		} else {
			t.Errorf("Expected context cancellation but received: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Build command did not exit after context cancellation")
	}
}

func TestBuildCommand_CustomTarget(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory and devnet.yaml
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextsDir := filepath.Join(configDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configs.ConfigYamls[configs.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}

	// Create custom userland build script
	userlandDir := filepath.Join(tmpDir, "userland")
	if err := os.MkdirAll(userlandDir, 0755); err != nil {
		t.Fatal(err)
	}
	capturePath := filepath.Join(tmpDir, "captured_args.txt")
	customScript := fmt.Sprintf(`#!/bin/bash
echo "$@" > "%s"
cat <<'EOF'
{"artifact":{"artifactId":"custom-artifact","component":"custom-component"}}
EOF
`, capturePath)
	customScriptPath := filepath.Join(userlandDir, "build")
	if err := os.WriteFile(customScriptPath, []byte(customScript), 0755); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	args := []string{
		"app", "build",
		"--context", devnet.DEVNET_CONTEXT,
		"--target", "userland/build",
		"--",
		"--dockerfile", "path/to/Dockerfile",
		"--context", "custom-performer",
	}

	if err := app.Run(args); err != nil {
		t.Fatalf("Failed to execute build command with custom target: %v", err)
	}

	// Ensure custom script captured the forwarded arguments
	captured, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatalf("Failed to read captured args: %v", err)
	}
	capturedArgs := strings.TrimSpace(string(captured))
	if !strings.Contains(capturedArgs, "--dockerfile path/to/Dockerfile") {
		t.Fatalf("Expected forwarded args to include dockerfile flag, got: %s", capturedArgs)
	}
	if !strings.Contains(capturedArgs, "--context custom-performer") {
		t.Fatalf("Expected forwarded args to include performer context flag, got: %s", capturedArgs)
	}

	// Context file should have been updated with custom artifact info
	contextBytes, err := os.ReadFile(filepath.Join(configDir, "contexts", "devnet.yaml"))
	if err != nil {
		t.Fatalf("Failed to read context file: %v", err)
	}
	contextStr := string(contextBytes)
	if !strings.Contains(contextStr, "artifactId: custom-artifact") {
		t.Fatalf("Expected context to include custom artifactId, got: %s", contextStr)
	}
	if !strings.Contains(contextStr, "component: custom-component") {
		t.Fatalf("Expected context to include custom component, got: %s", contextStr)
	}
}

func TestBuildCommand_CustomTargetMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config directory and devnet.yaml
	configDir := filepath.Join(tmpDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	contextsDir := filepath.Join(configDir, "contexts")
	if err := os.MkdirAll(contextsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configs.ConfigYamls[configs.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Failed to restore original directory: %v", err)
		}
	}()

	app := &cli.App{
		Name:     "test",
		Commands: []*cli.Command{testutils.WithTestConfigAndNoopLogger(BuildCommand)},
	}

	err = app.Run([]string{"app", "build", "--context", devnet.DEVNET_CONTEXT, "--target", "userland/build"})
	if err == nil {
		t.Fatal("Expected error due to missing custom target, got nil")
	}
	if !strings.Contains(err.Error(), "custom build target not found") {
		t.Fatalf("Expected missing target error, got: %v", err)
	}
}
