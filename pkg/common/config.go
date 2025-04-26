package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/naoina/toml"
)

const EigenTomlPath = "eigen.toml"

type ProjectConfig struct {
	Name        string `toml:"name"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
}

type OperatorConfig struct {
	Image       string              `toml:"image"`
	Keys        []string            `toml:"keys"`
	TotalStake  string              `toml:"total_stake"`
	Allocations OperatorAllocations `toml:"allocations"`
}

type OperatorAllocations struct {
	Strategies    []string `toml:"strategies"`
	TaskExecutors []string `toml:"task-executors"`
	Aggregators   []string `toml:"aggregators"`
}

type EnvConfig struct {
	NemesisContractAddress string   `toml:"nemesis_contract_address"`
	ChainImage             string   `toml:"chain_image"`
	ChainArgs              []string `toml:"chain_args"`
}

type OperatorSet struct {
	OperatorSetID int                  `toml:"operator_set_id"`
	Description   string               `toml:"description"`
	RPCEndpoint   string               `toml:"rpc_endpoint"`
	AVS           string               `toml:"avs"`
	SubmitWallet  string               `toml:"submit_wallet"`
	Operators     OperatorSetOperators `toml:"operators"`
}

type OperatorSetOperators struct {
	OperatorKeys               []string `toml:"operator_keys"`
	MinimumRequiredStakeWeight []string `toml:"minimum_required_stake_weight"`
}

type OperatorSetsMap map[string]OperatorSet

type OperatorSetsAliases struct {
	TaskExecution string `toml:"task_execution"`
	Aggregation   string `toml:"aggregation"`
}

type ReleaseConfig struct {
	AVSLogicImageTag string `toml:"avs_logic_image_tag"`
	PushImage        bool   `toml:"push_image"`
}

type EigenConfig struct {
	Project      ProjectConfig        `toml:"project"`
	Operator     OperatorConfig       `toml:"operator"`
	Env          map[string]EnvConfig `toml:"env"`
	OperatorSets OperatorSetsMap      `toml:"operatorsets"`
	Aliases      OperatorSetsAliases  `toml:"operatorset_aliases"`
	Release      ReleaseConfig        `toml:"release"`
}

// LoadEigenConfig loads into structured Go structs
func LoadEigenConfig() (*EigenConfig, error) {
	f, err := os.Open(EigenTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open eigen.toml: %w", err)
	}
	defer f.Close()

	var config EigenConfig
	if err := toml.NewDecoder(f).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}
	return &config, nil
}

// LoadEigenTree loads eigen.toml into a mutable map
func LoadEigenTree() (map[string]interface{}, error) {
	f, err := os.Open(EigenTomlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open eigen.toml: %w", err)
	}
	defer f.Close()

	var data map[string]interface{}
	if err := toml.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode TOML: %w", err)
	}
	return data, nil
}

// SaveEigenTree saves the mutable map back to eigen.toml
func SaveEigenTree(tree map[string]interface{}) error {
	f, err := os.Create(EigenTomlPath)
	if err != nil {
		return fmt.Errorf("failed to open eigen.toml for writing: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(tree)
}

// SetKey sets a value in a nested TOML tree
func SetKey(tree map[string]interface{}, keyPath string, value interface{}) error {
	parts := strings.Split(keyPath, ".")
	last := len(parts) - 1

	current := tree
	for i, part := range parts {
		if i == last {
			current[part] = value
			return nil
		}
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			return fmt.Errorf("invalid path: %s", strings.Join(parts[:i+1], "."))
		}
	}
	return nil
}
