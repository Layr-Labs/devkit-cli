package common

import (
	"testing"
)

func TestValidateEigenConfig(t *testing.T) {
	// Test case 1: Valid configuration
	validConfig := &EigenConfig{
		Project: ProjectConfig{
			Name:        "test-avs",
			Version:     "0.1.0",
			Description: "Test AVS",
		},
		Operator: OperatorConfig{
			Image:      "eigen/ponos-client:v1.0",
			Keys:       []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"},
			TotalStake: "1000ETH",
			Allocations: map[string][]string{
				"strategies":     {"0xf951e335afb289353dc249e82926178eac7ded78"},
				"task-executors": {"300000000000000000"},
				"aggregators":    {"250000000000000000"},
			},
		},

		Env: map[string]EnvConfig{
			"devnet": {
				NemesisContractAddress: "0x123...",
				ChainImage:             "ghcr.io/foundry-rs/foundry:latest",
				ChainArgs:              []string{"--chain-id", "31337"},
			},
		},
		OperatorSets: map[string]OperatorSet{
			"task-executors": {
				OperatorSetID: 0,
				Description:   "Operators responsible for executing tasks.",
				RPCEndpoint:   "http://localhost:8546",
				AVS:           "0xAVS...",
				SubmitWallet:  "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
				Operators: OperatorSetOperators{
					OperatorKeys:               []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"},
					MinimumRequiredStakeWeight: []string{"1000ETH"},
				},
			},
		},
		Release: ReleaseConfig{
			AVSLogicImageTag: "some-org/avs-logic:v0.1",
			PushImage:        false,
		},
	}

	result := ValidateEigenConfig(validConfig)
	if !result.Valid {
		t.Errorf("Expected valid config but got invalid. Errors: %v", result.Errors)
	}

	// Test case 2: Missing required fields
	invalidConfig := &EigenConfig{
		Project: ProjectConfig{
			// Name is missing
			Version:     "0.1.0",
			Description: "Test AVS",
		},
		Operator: OperatorConfig{
			// Image is missing
			Keys:       []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"},
			TotalStake: "1000ETH",
			Allocations: map[string][]string{
				"strategies":     {}, // Empty strategies
				"task-executors": {}, // Empty task executors
				"aggregators":    {}, // Empty aggregators
			},
		},
		Env: map[string]EnvConfig{
			"devnet": {
				NemesisContractAddress: "0x123...",
				// ChainImage is missing
				ChainArgs: []string{"--chain-id", "31337"},
			},
		},
		OperatorSets: map[string]OperatorSet{
			"task-executors": {
				OperatorSetID: 0,
				Description:   "Operators responsible for executing tasks.",
				RPCEndpoint:   "invalid-url", // Invalid URL
				AVS:           "0xAVS...",
				// SubmitWallet is missing
				Operators: OperatorSetOperators{
					OperatorKeys:               []string{}, // Empty operator keys
					MinimumRequiredStakeWeight: []string{"1000ETH"},
				},
			},
		},
		Release: ReleaseConfig{
			// AVSLogicImageTag is missing
			PushImage: false,
		},
	}

	result = ValidateEigenConfig(invalidConfig)
	if result.Valid {
		t.Errorf("Expected invalid config but got valid")
	}

	// Check specific errors (a subset for brevity in the test)
	expectedErrors := map[string]bool{
		"project.name":                             false,
		"operator.image":                           false,
		"operator.allocations.strategies":          false,
		"operatorsets.task-executors.rpc_endpoint": false,
	}

	for _, err := range result.Errors {
		expectedErrors[err.Field] = true
	}

	for field, found := range expectedErrors {
		if !found {
			t.Errorf("Expected validation error for field %s but didn't find it", field)
		}
	}

	// Test case 3: Mismatched array lengths
	mismatchedConfig := &EigenConfig{
		Project: ProjectConfig{
			Name:        "test-avs",
			Version:     "0.1.0",
			Description: "Test AVS",
		},
		Operator: OperatorConfig{
			Image:      "eigen/ponos-client:v1.0",
			Keys:       []string{"ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"},
			TotalStake: "1000ETH",
			Allocations: map[string][]string{
				"strategies":     {"0xf951e335afb289353dc249e82926178eac7ded78"},
				"task-executors": {"300000000000000000"},
				"aggregators":    {"250000000000000000"},
			},
		},
		OperatorSets: map[string]OperatorSet{
			"task-executors": {
				OperatorSetID: 0,
				Description:   "Operators responsible for executing tasks.",
				RPCEndpoint:   "http://localhost:8546",
				AVS:           "0xAVS...",
				SubmitWallet:  "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
				Operators: OperatorSetOperators{
					OperatorKeys:               []string{"key1", "key2"}, // 2 keys
					MinimumRequiredStakeWeight: []string{"1000ETH"},      // 1 weight, mismatch!
				},
			},
		},
		Release: ReleaseConfig{
			AVSLogicImageTag: "some-org/avs-logic:v0.1",
			PushImage:        false,
		},
	}

	result = ValidateEigenConfig(mismatchedConfig)
	if result.Valid {
		t.Errorf("Expected invalid config due to mismatched array lengths but got valid")
	}

	foundMismatchError := false
	for _, err := range result.Errors {
		if err.Field == "operatorsets.task-executors.operators" {
			foundMismatchError = true
			break
		}
	}

	if !foundMismatchError {
		t.Errorf("Expected validation error for mismatched array lengths but didn't find it")
	}
}
