package operator

import (
	"testing"
)

func TestAllocationAndDeallocation(t *testing.T) {
	op := Operator{ID: "op1"}
	sa := op.FindOrCreateStrategy("EIGEN")
	if sa.TotalMagnitude != INITIAL_TOTAL_MAGNITUDE {
		t.Fatalf("unexpected total magnitude: %d", sa.TotalMagnitude)
	}
	// Allocate
	err := sa.AllocateMagnitude("AVS_1_EIGEN", 3_000_000_000_000_000_000)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if sa.NonSlashable != INITIAL_TOTAL_MAGNITUDE-3_000_000_000_000_000_000 {
		t.Errorf("non-slashable wrong after allocation")
	}
	// Allocate more
	err = sa.AllocateMagnitude("AVS_2_EIGEN", 2_500_000_000_000_000_000)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	// Deallocate
	err = sa.DeallocateMagnitude("AVS_1_EIGEN", 1_000_000_000_000_000_000)
	if err != nil {
		t.Fatalf("deallocate: %v", err)
	}
	alloc := sa.FindAllocation("AVS_1_EIGEN")
	if alloc == nil || alloc.Magnitude != 2_000_000_000_000_000_000 {
		t.Errorf("deallocation did not update magnitude correctly")
	}
	if sa.NonSlashable != INITIAL_TOTAL_MAGNITUDE-2_000_000_000_000_000_000-2_500_000_000_000_000_000 {
		t.Errorf("non-slashable wrong after deallocation")
	}
	// Deallocate all
	err = sa.DeallocateMagnitude("AVS_1_EIGEN", 2_000_000_000_000_000_000)
	if err != nil {
		t.Fatalf("deallocate all: %v", err)
	}
	if sa.FindAllocation("AVS_1_EIGEN") != nil {
		t.Errorf("allocation not removed when zeroed")
	}
}

func TestDepositAndSummary(t *testing.T) {
	op := Operator{ID: "op2"}
	sa := op.FindOrCreateStrategy("EIGEN")
	err := sa.AllocateMagnitude("AVS_1_EIGEN", 3_000_000_000_000_000_000)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	err = sa.AllocateMagnitude("AVS_2_EIGEN", 2_500_000_000_000_000_000)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	err = sa.AllocateMagnitude("EigenDA_EIGEN", 2_000_000_000_000_000_000)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	// Deposit
	err = sa.Deposit(100)
	if err != nil {
		t.Fatalf("deposit: %v", err)
	}
	sum := sa.Summary()
	if sum.Delegated != 100 {
		t.Errorf("delegated wrong: %d", sum.Delegated)
	}
	// Proportions
	for _, a := range sum.Allocations {
		if a.OperatorSet == "AVS_1_EIGEN" && a.Proportion < 0.299 || a.Proportion > 0.301 {
			t.Errorf("proportion for AVS_1_EIGEN not ~0.3: %f", a.Proportion)
		}
	}
}

func TestErrors(t *testing.T) {
	op := Operator{ID: "op3"}
	sa := op.FindOrCreateStrategy("EIGEN")
	// Over-allocate
	err := sa.AllocateMagnitude("AVS_1_EIGEN", INITIAL_TOTAL_MAGNITUDE+1)
	if err == nil {
		t.Errorf("expected error on over-allocation")
	}
	// Negative allocation
	err = sa.AllocateMagnitude("AVS_1_EIGEN", -1)
	if err == nil {
		t.Errorf("expected error on negative allocation")
	}
	// Deallocate non-existent
	err = sa.DeallocateMagnitude("notfound", 1)
	if err == nil {
		t.Errorf("expected error on deallocating non-existent operator set")
	}
	// Deallocate too much
	err = sa.AllocateMagnitude("AVS_1_EIGEN", 100)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	err = sa.DeallocateMagnitude("AVS_1_EIGEN", 200)
	if err == nil {
		t.Errorf("expected error on deallocating too much")
	}
	// Deposit negative
	err = sa.Deposit(-1)
	if err == nil {
		t.Errorf("expected error on negative deposit")
	}
}
