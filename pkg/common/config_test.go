package common_test

import (
	"devkit-cli/pkg/common"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestLoadEigenConfig_FromCopiedTempFile(t *testing.T) {
	// Setup temp file
	tempDir := t.TempDir()
	tempTomlPath := filepath.Join(tempDir, "eigen.toml")

	srcPath := filepath.Join("..", "..", "default.eigen.toml")
	src, err := os.Open(srcPath)
	assert.NoError(t, err)
	defer src.Close()

	dest, err := os.Create(tempTomlPath)
	assert.NoError(t, err)
	defer dest.Close()

	_, err = io.Copy(dest, src)
	assert.NoError(t, err)
	err = dest.Sync()
	assert.NoError(t, err)

	// Load config
	cfg, err := LoadEigenConfigFromPath(tempTomlPath)
	assert.NoError(t, err)

	// Project
	assert.Equal(t, "my-avs", cfg.Project.Name)
	assert.Equal(t, "0.1.0", cfg.Project.Version)
	assert.Equal(t, "Default Hourglass AVS with minimal settings.", cfg.Project.Description)

	// Operator
	assert.Equal(t, "eigen/ponos-client:v1.0", cfg.Operator.Image)
	assert.Equal(t, []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"}, cfg.Operator.Keys)
	assert.Equal(t, "1000ETH", cfg.Operator.TotalStake)

	// Allocations
	assert.Equal(t, []string{"0xf951e335afb289353dc249e82926178eac7ded78"}, cfg.Operator.Allocations.Strategies)
	assert.Equal(t, []string{"300000000000000000"}, cfg.Operator.Allocations.TaskExecutors)
	assert.Equal(t, []string{"250000000000000000"}, cfg.Operator.Allocations.Aggregators)

	// Environment
	devnet := cfg.Env["devnet"]
	assert.Equal(t, "0x123...", devnet.NemesisContractAddress)
	assert.Equal(t, "ghcr.io/foundry-rs/foundry:latest", devnet.ChainImage)
	assert.Equal(t, []string{"--chain-id", "31337", "--block-time", "3", "--gas-price", "0", "--base-fee", "0"}, devnet.ChainArgs)

	// Operator sets
	taskSet := cfg.OperatorSets["task-executors"]
	assert.Equal(t, 0, taskSet.OperatorSetID)
	assert.Equal(t, "Operators responsible for executing tasks.", taskSet.Description)
	assert.Equal(t, "http://localhost:8546", taskSet.RPCEndpoint)
	assert.Equal(t, "0xAVS...", taskSet.AVS)
	assert.Equal(t, "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", taskSet.SubmitWallet)
	assert.Equal(t, []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"}, taskSet.Operators.OperatorKeys)
	assert.Equal(t, []string{"1000ETH"}, taskSet.Operators.MinimumRequiredStakeWeight)

	// Aliases
	assert.Equal(t, "task-executors", cfg.Aliases.TaskExecution)
	assert.Equal(t, "aggregators", cfg.Aliases.Aggregation)

	// Release
	assert.Equal(t, "some-org/avs-logic:v0.1", cfg.Release.AVSLogicImageTag)
	assert.False(t, cfg.Release.PushImage)
	// log
	assert.Equal(t, "debug", cfg.Log.Level)
}

func LoadEigenConfigFromPath(path string) (*common.EigenConfig, error) {
	var config common.EigenConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
