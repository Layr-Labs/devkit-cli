package common

import (
	"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
	"strings"
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

func LoadEigenConfig() (*EigenConfig, error) {
	const defaultPath = "eigen.toml"

	var config EigenConfig
	if _, err := toml.DecodeFile(defaultPath, &config); err != nil {
		return nil, fmt.Errorf("eigen.toml not found. Are you running this command from your project directory?")
	}
	return &config, nil
}

func PrintStyledConfig(tomlOutput string) {
	sectionColor := color.New(color.FgHiBlue).SprintFunc()
	keyColor := color.New(color.FgHiWhite).SprintFunc()
	valueColor := color.New(color.FgHiCyan).SprintFunc()

	lines := strings.Split(tomlOutput, "\n")
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]"):
			// Section headers
			fmt.Println(sectionColor(line))

		case strings.Contains(trim, "="):
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := keyColor(strings.TrimSpace(parts[0]))
				value := valueColor(strings.TrimSpace(parts[1]))
				fmt.Printf("%s = %s\n", key, value)
			} else {
				fmt.Println(line)
			}

		default:
			fmt.Println(line)
		}
	}
}

// StructToMap converts a struct to a map[string]interface{}
func StructToMap(cfg interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}

	tmp, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}

	if err := json.Unmarshal(tmp, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal into map: %w", err)
	}

	return result, nil
}
