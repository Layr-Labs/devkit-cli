package migration_test

import (
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	"gopkg.in/yaml.v3"
)

// assumes testNode(t, yamlStr) helper exists in this package like in your other test
func TestMigration_0_1_1_to_0_1_2(t *testing.T) {
	oldYAML := `
version: 0.1.1
context:
  name: "devnet"
  chains:
    l1:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 9259079
        url: ""
        block_time: 3
    l2:
      chain_id: 31338
      rpc_url: "http://localhost:9545"
      fork:
        block: 31408197
        url: ""
        block_time: 3
`

	userNode := testNode(t, oldYAML)

	// locate the 0.1.1 -> 0.1.2 step from the chain
	var step migration.MigrationStep
	for _, s := range contexts.MigrationChain {
		if s.From == "0.1.1" && s.To == "0.1.2" {
			step = s
			break
		}
	}
	if step.Apply == nil {
		t.Fatalf("migration step 0.1.1 -> 0.1.2 not found")
	}

	migrated, err := migration.MigrateNode(userNode, "0.1.1", "0.1.2", []migration.MigrationStep{step})
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	t.Run("version bumped", func(t *testing.T) {
		v := migration.ResolveNode(migrated, []string{"version"})
		if v == nil || v.Value != "0.1.2" {
			t.Errorf("expected version 0.1.2, got %v", v)
		}
	})

	t.Run("mnemonic inserted with value", func(t *testing.T) {
		// value node
		val := migration.ResolveNode(migrated, []string{"context", "mnemonic"})
		if val == nil {
			t.Fatalf("mnemonic key missing")
		}
		want := "test test test test test test test test test test test junk"
		if val.Value != want {
			t.Errorf("expected mnemonic value %q, got %q", want, val.Value)
		}
	})

	t.Run("inserted after context.chains", func(t *testing.T) {
		// inspect ordering within the context mapping
		ctx := migration.ResolveNode(migrated, []string{"context"})
		if ctx == nil || ctx.Kind != 4 /* yaml.ScalarNode */ {
			t.Fatalf("context mapping missing or wrong kind %d", ctx.Kind)
		}
		chainsIdx := -1
		mnemonicIdx := -1
		for i := 0; i < len(ctx.Content)-1; i += 2 {
			k := ctx.Content[i]
			switch k.Value {
			case "chains":
				chainsIdx = i
			case "mnemonic":
				mnemonicIdx = i
			}
		}
		if chainsIdx < 0 {
			t.Fatalf("context.chains key not found")
		}
		if mnemonicIdx < 0 {
			t.Fatalf("context.mnemonic key not found")
		}
		if mnemonicIdx != chainsIdx+2 {
			t.Errorf("mnemonic not inserted immediately after chains: chainsIdx=%d mnemonicIdx=%d", chainsIdx, mnemonicIdx)
		}
	})

	t.Run("comment attached to mnemonic key", func(t *testing.T) {
		ctx := migration.ResolveNode(migrated, []string{"context"})
		var keyNode *yaml.Node
		for i := 0; i < len(ctx.Content)-1; i += 2 {
			if ctx.Content[i].Value == "mnemonic" {
				keyNode = ctx.Content[i]
				break
			}
		}
		if keyNode == nil {
			t.Fatalf("mnemonic key node not found")
		}
		wantComment := "Devnet mnemonic for unlocked accounts"
		if keyNode.HeadComment != wantComment {
			t.Errorf("expected head comment %q, got %q", wantComment, keyNode.HeadComment)
		}
	})

	t.Run("other fields preserved", func(t *testing.T) {
		nameNode := migration.ResolveNode(migrated, []string{"context", "name"})
		if nameNode == nil || nameNode.Value != "devnet" {
			t.Errorf("expected name preserved as devnet, got %v", nameNode)
		}
		l1ChainId := migration.ResolveNode(migrated, []string{"context", "chains", "l1", "chain_id"})
		if l1ChainId == nil || l1ChainId.Value != "31337" {
			t.Errorf("expected L1 chain_id 31337, got %v", l1ChainId)
		}
		l2ChainId := migration.ResolveNode(migrated, []string{"context", "chains", "l2", "chain_id"})
		if l2ChainId == nil || l2ChainId.Value != "31338" {
			t.Errorf("expected L2 chain_id 31338, got %v", l2ChainId)
		}
	})
}
