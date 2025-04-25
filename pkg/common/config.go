package common

import (
	"errors"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
)

const DEFAULT_CONFIG_FILE = "eigen.toml"

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

func LoadEigenConfig() (*EigenConfig, error) {
	var config EigenConfig
	if _, err := toml.DecodeFile(DEFAULT_CONFIG_FILE, &config); err != nil {
		return nil, fmt.Errorf("%s not found. Are you running this command from your project directory?", DEFAULT_CONFIG_FILE)
	}

	// Validate the config after loading
	validationResult := ValidateEigenConfig(&config)
	if !validationResult.Valid {
		// Construct a meaningful error message with all validation issues
		var errMsg strings.Builder
		errMsg.WriteString("Configuration validation failed:\n")
		for _, err := range validationResult.Errors {
			errMsg.WriteString(fmt.Sprintf("- %s: %s\n", err.Field, err.Message))
		}
		return &config, errors.New(errMsg.String())
	}

	return &config, nil
}
