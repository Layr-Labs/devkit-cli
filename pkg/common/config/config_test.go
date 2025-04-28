package config_test

import (
	"devkit-cli/pkg/common/config"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
)

func TestLoadEigenConfig_FromCopiedTempFile(t *testing.T) {
	// Setup temp file
	tempDir := t.TempDir()
	tempTomlPath := filepath.Join(tempDir, "eigen.toml")

	srcPath := filepath.Join("..", "..", "..", "default.eigen.toml")
	src, err := os.Open(srcPath)
	assert.NoError(t, err)
	defer func() {
		err := src.Close()
		assert.NoError(t, err)
	}()

	dest, err := os.Create(tempTomlPath)
	assert.NoError(t, err)
	defer func() {
		err := dest.Close()
		assert.NoError(t, err)
	}()

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
	assert.Equal(t, []string{"0xf951e335afb289353dc249e82926178eac7ded78"}, cfg.Operator.Allocations["strategies"])
	assert.Equal(t, []string{"300000000000000000"}, cfg.Operator.Allocations["task-executors"])
	assert.Equal(t, []string{"250000000000000000"}, cfg.Operator.Allocations["aggregators"])

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

func TestLoadEigenConfig_WithAdditionalOperatorSet(t *testing.T) {
	tempDir := t.TempDir()
	tempTomlPath := filepath.Join(tempDir, "eigen.toml")

	// Load default template
	srcPath := filepath.Join("..", "..", "default.eigen.toml")
	srcBytes, err := os.ReadFile(srcPath)
	assert.NoError(t, err)

	tomlStr := string(srcBytes)

	// Append new allocation into existing [operator.allocations]
	tomlStr = strings.Replace(tomlStr, "[operator.allocations]",
		`[operator.allocations]
additional-set = ["200000000000000000"]`, 1)

	// Append new operator set at the end
	tomlStr += `
[operatorsets.additional-set]
operator_set_id = 2
description = "Handles fallback tasks"
rpc_endpoint = "http://localhost:8548"
avs = "0xAVS_EXTRA"
submit_wallet = "0xWalletExtra"

  [operatorsets.additional-set.operators]
  operator_keys = ["0xkey3"]
  minimum_required_stake_weight = ["750ETH"]
`

	// Write to temp path
	assert.NoError(t, os.WriteFile(tempTomlPath, []byte(tomlStr), 0644))

	// Load config and validate
	cfg, err := LoadEigenConfigFromPath(tempTomlPath)
	assert.NoError(t, err)

	// Check allocations
	assert.Contains(t, cfg.Operator.Allocations, "additional-set")
	assert.Equal(t, []string{"200000000000000000"}, cfg.Operator.Allocations["additional-set"])

	// Check new operator set
	addSet := cfg.OperatorSets["additional-set"]
	assert.Equal(t, 2, addSet.OperatorSetID)
	assert.Equal(t, "Handles fallback tasks", addSet.Description)
	assert.Equal(t, "http://localhost:8548", addSet.RPCEndpoint)
	assert.Equal(t, "0xAVS_EXTRA", addSet.AVS)
	assert.Equal(t, "0xWalletExtra", addSet.SubmitWallet)
	assert.Equal(t, []string{"0xkey3"}, addSet.Operators.OperatorKeys)
	assert.Equal(t, []string{"750ETH"}, addSet.Operators.MinimumRequiredStakeWeight)
}

func LoadEigenConfigFromPath(path string) (*config.EigenConfig, error) {
	var config config.EigenConfig
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
