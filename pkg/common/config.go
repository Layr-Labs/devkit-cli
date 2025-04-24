package common

import (
	"github.com/BurntSushi/toml"
)

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

type LogConfig struct {
	Level string `toml:"level"` // Expected values: "debug", "info", "warn", "error"
}

type EigenConfig struct {
	Project      ProjectConfig        `toml:"project"`
	Operator     OperatorConfig       `toml:"operator"`
	Env          map[string]EnvConfig `toml:"env"`
	OperatorSets OperatorSetsMap      `toml:"operatorsets"`
	Aliases      OperatorSetsAliases  `toml:"operatorset_aliases"`
	Release      ReleaseConfig        `toml:"release"`
	Log          LogConfig            `toml:"log"`
}

func LoadEigenConfig() (*EigenConfig, error) {
	const defaultPath = "eigen.toml"

	var config EigenConfig
	if _, err := toml.DecodeFile(defaultPath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
