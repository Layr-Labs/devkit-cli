package operator

import (
	"errors"
	"fmt"
)

const INITIAL_TOTAL_MAGNITUDE = 1_000_000_000_000_000_000 // 1e18

// FindOperator returns a pointer to the operator with the given ID.
func (data *OperatorData) FindOperator(operatorID string) (*Operator, error) {
	for i := range data.Operators {
		if data.Operators[i].ID == operatorID {
			return &data.Operators[i], nil
		}
	}
	return nil, fmt.Errorf("operator %s not found", operatorID)
}

// FindOrCreateStrategy returns a pointer to the strategy allocation, creating it if needed.
func (op *Operator) FindOrCreateStrategy(strategy string) *StrategyAllocation {
	for i := range op.Strategies {
		if op.Strategies[i].Strategy == strategy {
			return &op.Strategies[i]
		}
	}
	// Create new
	sa := StrategyAllocation{
		Strategy:       strategy,
		TotalMagnitude: INITIAL_TOTAL_MAGNITUDE,
		NonSlashable:   INITIAL_TOTAL_MAGNITUDE,
	}
	op.Strategies = append(op.Strategies, sa)
	return &op.Strategies[len(op.Strategies)-1]
}

// FindStrategy returns a pointer to the strategy allocation, or nil if not found.
func (op *Operator) FindStrategy(strategy string) *StrategyAllocation {
	for i := range op.Strategies {
		if op.Strategies[i].Strategy == strategy {
			return &op.Strategies[i]
		}
	}
	return nil
}

// FindAllocation returns a pointer to the allocation for the given operator set, or nil if not found.
func (sa *StrategyAllocation) FindAllocation(operatorSet string) *MagnitudeAllocation {
	for i := range sa.Allocations {
		if sa.Allocations[i].OperatorSet == operatorSet {
			return &sa.Allocations[i]
		}
	}
	return nil
}

// AllocateMagnitude allocates magnitude to an operator set.
func (sa *StrategyAllocation) AllocateMagnitude(operatorSet string, magnitude int64) error {
	if magnitude <= 0 {
		return errors.New("magnitude must be positive")
	}
	if magnitude > sa.NonSlashable {
		return fmt.Errorf("not enough non-slashable magnitude (available: %d)", sa.NonSlashable)
	}
	alloc := sa.FindAllocation(operatorSet)
	if alloc == nil {
		sa.Allocations = append(sa.Allocations, MagnitudeAllocation{
			OperatorSet: operatorSet,
			Magnitude:   magnitude,
		})
	} else {
		alloc.Magnitude += magnitude
	}
	sa.NonSlashable -= magnitude
	return nil
}

// DeallocateMagnitude deallocates magnitude from an operator set.
func (sa *StrategyAllocation) DeallocateMagnitude(operatorSet string, magnitude int64) error {
	if magnitude <= 0 {
		return errors.New("magnitude must be positive")
	}
	for i := range sa.Allocations {
		if sa.Allocations[i].OperatorSet == operatorSet {
			if sa.Allocations[i].Magnitude < magnitude {
				return fmt.Errorf("cannot deallocate more than allocated (allocated: %d)", sa.Allocations[i].Magnitude)
			}
			sa.Allocations[i].Magnitude -= magnitude
			sa.NonSlashable += magnitude
			if sa.Allocations[i].Magnitude == 0 {
				// Remove allocation
				sa.Allocations = append(sa.Allocations[:i], sa.Allocations[i+1:]...)
			}
			return nil
		}
	}
	return fmt.Errorf("operator set %s not found", operatorSet)
}

// Deposit increases the total delegated for a strategy.
func (sa *StrategyAllocation) Deposit(amount int64) error {
	if amount <= 0 {
		return errors.New("deposit amount must be positive")
	}
	sa.TotalDelegated += amount
	return nil
}

// ListAllocations returns a summary of allocations, proportions, and EIGEN per set.
type AllocationSummary struct {
	OperatorSet string
	Magnitude   int64
	Proportion  float64 // 0..1
	Amount      int64   // EIGEN (proportion * TotalDelegated)
}

type StrategySummary struct {
	Strategy     string
	Total        int64
	NonSlashable int64
	Delegated    int64
	Allocations  []AllocationSummary
}

func (sa *StrategyAllocation) Summary() StrategySummary {
	allocs := make([]AllocationSummary, 0, len(sa.Allocations)+1)
	for _, a := range sa.Allocations {
		prop := float64(a.Magnitude) / float64(sa.TotalMagnitude)
		amt := int64(prop * float64(sa.TotalDelegated))
		allocs = append(allocs, AllocationSummary{
			OperatorSet: a.OperatorSet,
			Magnitude:   a.Magnitude,
			Proportion:  prop,
			Amount:      amt,
		})
	}
	// Non-slashable
	prop := float64(sa.NonSlashable) / float64(sa.TotalMagnitude)
	amt := int64(prop * float64(sa.TotalDelegated))
	allocs = append(allocs, AllocationSummary{
		OperatorSet: "Non-slashable",
		Magnitude:   sa.NonSlashable,
		Proportion:  prop,
		Amount:      amt,
	})
	return StrategySummary{
		Strategy:     sa.Strategy,
		Total:        sa.TotalMagnitude,
		NonSlashable: sa.NonSlashable,
		Delegated:    sa.TotalDelegated,
		Allocations:  allocs,
	}
}
