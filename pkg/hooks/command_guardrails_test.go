package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectStageOrdering(t *testing.T) {
	tests := []struct {
		current  ProjectStage
		required ProjectStage
		allowed  bool
	}{
		{StageUninitialized, StageUninitialized, true},
		{StageCreated, StageUninitialized, true},
		{StageCreated, StageCreated, true},
		{StageBuilt, StageCreated, true},
		{StageDevnetReady, StageBuilt, true},
		{StageRunning, StageDevnetReady, true},
		{StageCreated, StageBuilt, false},
		{StageUninitialized, StageCreated, false},
		{StageBuilt, StageDevnetReady, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.current)+"_to_"+string(tt.required), func(t *testing.T) {
			result := isStageAllowed(tt.current, tt.required)
			assert.Equal(t, tt.allowed, result)
		})
	}
}

func TestFindCommandDependency(t *testing.T) {
	tests := []struct {
		command       string
		shouldFind    bool
		requiredStage ProjectStage
	}{
		{"create", true, StageUninitialized},
		{"build", true, StageCreated},
		{"start", true, StageCreated},
		{"deploy-contracts", true, StageCreated},
		{"stop", true, StageCreated},
		{"call", true, StageRunning},
		{"run", true, StageDevnetReady},
		{"nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			dep := findCommandDependency(tt.command)
			if tt.shouldFind {
				require.NotNil(t, dep)
				assert.Equal(t, tt.requiredStage, dep.RequiredStage)
			} else {
				assert.Nil(t, dep)
			}
		})
	}
}

func TestGetCurrentProjectStage(t *testing.T) {
	// Test uninitialized state
	t.Run("uninitialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		require.NoError(t, os.Chdir(tmpDir))

		stage, err := getCurrentProjectStage()
		require.NoError(t, err)
		assert.Equal(t, StageUninitialized, stage)
	})

	// Test created state (config exists but no context)
	t.Run("created", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		// Create basic project structure
		configDir := filepath.Join(tmpDir, "config")
		require.NoError(t, os.MkdirAll(configDir, 0755))

		configContent := `version: "0.1.0"
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

		require.NoError(t, os.Chdir(tmpDir))

		stage, err := getCurrentProjectStage()
		require.NoError(t, err)
		assert.Equal(t, StageCreated, stage)
	})

	// Test built state (build artifacts exist)
	t.Run("built", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		// Create project structure with build artifacts
		configDir := filepath.Join(tmpDir, "config")
		contextsDir := filepath.Join(configDir, "contexts")
		contractsDir := filepath.Join(tmpDir, "contracts", "out")
		require.NoError(t, os.MkdirAll(contextsDir, 0755))
		require.NoError(t, os.MkdirAll(contractsDir, 0755))

		configContent := `version: "0.1.0"
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

		contextContent := `version: "0.1.0"
context:
  name: "devnet"
  chains:
    l1:
      chain_id: 1
      rpc_url: "http://localhost:8545"
`
		require.NoError(t, os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contextContent), 0644))

		require.NoError(t, os.Chdir(tmpDir))

		stage, err := getCurrentProjectStage()
		require.NoError(t, err)
		assert.Equal(t, StageBuilt, stage)
	})

	// Test devnet ready state (deployed contracts exist)
	t.Run("devnet_ready", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		// Create project structure with deployed contracts
		configDir := filepath.Join(tmpDir, "config")
		contextsDir := filepath.Join(configDir, "contexts")
		require.NoError(t, os.MkdirAll(contextsDir, 0755))

		configContent := `version: "0.1.0"
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

		contextContent := `version: "0.1.0"
context:
  name: "devnet"
  chains:
    l1:
      chain_id: 1
      rpc_url: "http://localhost:8545"
  deployed_contracts:
    - name: "TestContract"
      address: "0x1234567890123456789012345678901234567890"
`
		require.NoError(t, os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contextContent), 0644))

		require.NoError(t, os.Chdir(tmpDir))

		stage, err := getCurrentProjectStage()
		require.NoError(t, err)
		assert.Equal(t, StageDevnetReady, stage)
	})

	// Test explicit stage setting
	t.Run("explicit_stage", func(t *testing.T) {
		tmpDir := t.TempDir()
		origDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(origDir) }()

		// Create project structure with explicit stage
		configDir := filepath.Join(tmpDir, "config")
		contextsDir := filepath.Join(configDir, "contexts")
		require.NoError(t, os.MkdirAll(contextsDir, 0755))

		configContent := `version: "0.1.0"
config:
  project:
    name: "test-project"
    version: "0.1.0"
    context: "devnet"
`
		require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(configContent), 0644))

		contextContent := `version: "0.1.0"
context:
  name: "devnet"
  stage: "running"
  chains:
    l1:
      chain_id: 1
      rpc_url: "http://localhost:8545"
`
		require.NoError(t, os.WriteFile(filepath.Join(contextsDir, "devnet.yaml"), []byte(contextContent), 0644))

		require.NoError(t, os.Chdir(tmpDir))

		stage, err := getCurrentProjectStage()
		require.NoError(t, err)
		assert.Equal(t, StageRunning, stage)
	})
}
