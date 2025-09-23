package migration_test

import (
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
)

func TestMigration_0_1_0_to_0_1_1(t *testing.T) {
	// Test YAML with old block heights
	oldYAML := `
version: 0.1.0
context:
  name: "devnet"
  chains:
    l1:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 9085290
        url: ""
        block_time: 3
    l2:
      chain_id: 31338
      rpc_url: "http://localhost:9545"
      fork:
        block: 30327360
        url: ""
        block_time: 3
`

	userNode := testNode(t, oldYAML)

	// locate the 0.1.0 -> 0.1.1 step from the chain
	var step migration.MigrationStep
	for _, s := range contexts.MigrationChain {
		if s.From == "0.1.0" && s.To == "0.1.1" {
			step = s
			break
		}
	}
	if step.Apply == nil {
		t.Fatalf("migration step 0.1.0 -> 0.1.1 not found")
	}

	migrated, err := migration.MigrateNode(userNode, "0.1.0", "0.1.1", []migration.MigrationStep{step})
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	t.Run("version bumped", func(t *testing.T) {
		v := migration.ResolveNode(migrated, []string{"version"})
		if v == nil || v.Value != "0.1.1" {
			t.Errorf("expected version 0.1.1, got %v", v)
		}
	})

	t.Run("L1 block height updated", func(t *testing.T) {
		blockNode := migration.ResolveNode(migrated, []string{"context", "chains", "l1", "fork", "block"})
		if blockNode == nil || blockNode.Value != "9259079" {
			t.Errorf("expected L1 block 9259079, got %v", blockNode)
		}
	})

	t.Run("L2 block height updated", func(t *testing.T) {
		blockNode := migration.ResolveNode(migrated, []string{"context", "chains", "l2", "fork", "block"})
		if blockNode == nil || blockNode.Value != "31408197" {
			t.Errorf("expected L2 block 31408197, got %v", blockNode)
		}
	})

	t.Run("other fields preserved", func(t *testing.T) {
		// Check that other fields are not modified
		nameNode := migration.ResolveNode(migrated, []string{"context", "name"})
		if nameNode == nil || nameNode.Value != "devnet" {
			t.Errorf("expected name to be preserved as 'devnet', got %v", nameNode)
		}

		l1ChainIdNode := migration.ResolveNode(migrated, []string{"context", "chains", "l1", "chain_id"})
		if l1ChainIdNode == nil || l1ChainIdNode.Value != "31337" {
			t.Errorf("expected L1 chain_id to be preserved as 31337, got %v", l1ChainIdNode)
		}

		l2ChainIdNode := migration.ResolveNode(migrated, []string{"context", "chains", "l2", "chain_id"})
		if l2ChainIdNode == nil || l2ChainIdNode.Value != "31338" {
			t.Errorf("expected L2 chain_id to be preserved as 31338, got %v", l2ChainIdNode)
		}
	})
}
