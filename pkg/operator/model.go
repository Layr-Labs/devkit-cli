package operator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// MagnitudeAllocation represents an allocation to an Operator Set
// for a specific strategy.
type MagnitudeAllocation struct {
	OperatorSet string `json:"operator_set"`
	Magnitude   int64  `json:"magnitude"`
}

// StrategyAllocation represents all allocations for a strategy.
type StrategyAllocation struct {
	Strategy       string                `json:"strategy"`
	TotalMagnitude int64                 `json:"total_magnitude"`
	Allocations    []MagnitudeAllocation `json:"allocations"`
	NonSlashable   int64                 `json:"non_slashable"`
	TotalDelegated int64                 `json:"total_delegated"`
}

// Operator represents an operator with strategies and allocations.
type Operator struct {
	ID         string               `json:"id"`
	Strategies []StrategyAllocation `json:"strategies"`
}

// OperatorData is the top-level structure for all operators.
type OperatorData struct {
	Operators []Operator `json:"operators"`
}

// LoadOperatorData loads operator data from a JSON file.
func LoadOperatorData(path string) (*OperatorData, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &OperatorData{}, nil // treat as empty if not found
		}
		return nil, fmt.Errorf("open operator data: %w", err)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read operator data: %w", err)
	}
	var data OperatorData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("unmarshal operator data: %w", err)
	}
	return &data, nil
}

// SaveOperatorData saves operator data to a JSON file.
func SaveOperatorData(path string, data *OperatorData) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal operator data: %w", err)
	}
	if err := ioutil.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("write operator data: %w", err)
	}
	return nil
}
