package migration_test

import (
	"fmt"
	"testing"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	configMigrations "github.com/Layr-Labs/devkit-cli/config/configs/migrations"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	"gopkg.in/yaml.v3"
)

// helper to parse YAML into *yaml.Node
func testNode(t *testing.T, input string) *yaml.Node {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// unwrap DocumentNode
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}

func TestConfigMigration_0_0_1_to_0_0_2(t *testing.T) {
	// Use the embedded v0.0.1 content as our starting point and upgrade to v0.0.2
	user := testNode(t, string(configs.ConfigYamls["0.0.1"]))
	old := testNode(t, string(configs.ConfigYamls["0.0.1"]))
	new := testNode(t, string(configs.ConfigYamls["0.0.2"]))

	migrated, err := configMigrations.Migration_0_0_1_to_0_0_2(user, old, new)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	t.Run("version bumped", func(t *testing.T) {
		version := migration.ResolveNode(migrated, []string{"version"})
		if version == nil || version.Value != "0.0.2" {
			t.Errorf("Expected version to be '0.0.2', got: %v", version.Value)
		}
	})

	t.Run("project_uuid added", func(t *testing.T) {
		val := migration.ResolveNode(migrated, []string{"config", "project", "project_uuid"})
		if val == nil || val.Value != "" {
			t.Errorf("Expected empty project_uuid, got: %v", val)
		}
	})

	t.Run("telemetry_enabled added", func(t *testing.T) {
		val := migration.ResolveNode(migrated, []string{"config", "project", "telemetry_enabled"})
		if val == nil || val.Value != "false" {
			t.Errorf("Expected telemetry_enabled to be false, got: %v", val)
		}
	})

	t.Run("templateBaseUrl added", func(t *testing.T) {
		val := migration.ResolveNode(migrated, []string{"config", "project", "templateBaseUrl"})
		expected := "https://github.com/Layr-Labs/hourglass-avs-template"
		if val == nil || val.Value != expected {
			t.Errorf("Expected templateBaseUrl to be '%s', got: %v", expected, val)
		}
	})

	t.Run("templateVersion added", func(t *testing.T) {
		val := migration.ResolveNode(migrated, []string{"config", "project", "templateVersion"})
		if val == nil || val.Value != "v0.0.10" {
			t.Errorf("Expected templateVersion to be 'v0.0.10', got: %v", val)
		}
	})
}

// TestAVSContextMigration_0_0_1_to_0_0_2 tests the specific migration from version 0.0.1 to 0.0.2
// using the actual migration files from config/contexts/
func TestAVSContextMigration_0_0_1_to_0_0_2(t *testing.T) {
	// Use the embedded v0.0.1 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.1"])

	// Parse YAML nodes
	userNode := testNode(t, userYAML)

	// Get the actual migration step from the contexts package
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.1" && step.To == "0.0.2" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.1 -> 0.0.2 migration step in contexts.MigrationChain")
	}

	// Execute migration using the actual migration chain
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.1", "0.0.2", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the migration results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.2" {
			t.Errorf("Expected version to be updated to 0.0.2, got %v", version.Value)
		}
	})

	t.Run("L1 fork URL updated", func(t *testing.T) {
		l1ForkUrl := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "url"})
		if l1ForkUrl == nil || l1ForkUrl.Value != "" {
			t.Errorf("Expected L1 fork URL to be empty, got %v", l1ForkUrl.Value)
		}
	})

	t.Run("L2 fork URL updated", func(t *testing.T) {
		l2ForkUrl := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "fork", "url"})
		if l2ForkUrl == nil || l2ForkUrl.Value != "" {
			t.Errorf("Expected L2 fork URL to be empty, got %v", l2ForkUrl.Value)
		}
	})

	t.Run("app_private_key updated", func(t *testing.T) {
		appKey := migration.ResolveNode(migratedNode, []string{"context", "app_private_key"})
		if appKey == nil || appKey.Value != "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" {
			t.Errorf("Expected app_private_key to be updated to new value, got %v", appKey.Value)
		}
	})

	t.Run("operator details preserved", func(t *testing.T) {
		// Since the user's operator 0 values match the old default values,
		// the migration will update them to the new default values (this is correct behavior)
		opAddress := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "address"})
		if opAddress == nil || opAddress.Value != "0x90F79bf6EB2c4f870365E785982E1f101E93b906" {
			t.Errorf("Expected operator address to be updated to new default value, got %v", opAddress.Value)
		}

		opKey := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "ecdsa_key"})
		if opKey == nil || opKey.Value != "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" {
			t.Errorf("Expected operator ECDSA key to be updated to new default value, got %v", opKey.Value)
		}

		opStake := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "stake"})
		if opStake == nil || opStake.Value != "1000ETH" {
			t.Errorf("Expected operator stake to be preserved, got %v", opStake.Value)
		}
	})

	t.Run("AVS details preserved", func(t *testing.T) {
		// Since the user's AVS values match the old default values,
		// the migration will update them to the new default values (this is correct behavior)
		avsAddress := migration.ResolveNode(migratedNode, []string{"context", "avs", "address"})
		if avsAddress == nil || avsAddress.Value != "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" {
			t.Errorf("Expected AVS address to be updated to new default value, got %v", avsAddress.Value)
		}

		// AVS private key should be updated to new default value
		avsKey := migration.ResolveNode(migratedNode, []string{"context", "avs", "avs_private_key"})
		if avsKey == nil || avsKey.Value != "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" {
			t.Errorf("Expected AVS private key to be updated to new default value, got %v", avsKey.Value)
		}

		avsMetadata := migration.ResolveNode(migratedNode, []string{"context", "avs", "metadata_url"})
		if avsMetadata == nil || avsMetadata.Value != "https://my-org.com/avs/metadata.json" {
			t.Errorf("Expected AVS metadata URL to be preserved, got %v", avsMetadata.Value)
		}
	})

	t.Run("chain configuration preserved", func(t *testing.T) {
		// Chain IDs
		l1ChainId := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "chain_id"})
		if l1ChainId == nil || l1ChainId.Value != "31337" {
			t.Errorf("Expected L1 chain ID to be preserved, got %v", l1ChainId.Value)
		}

		// RPC URLs
		l1RpcUrl := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "rpc_url"})
		if l1RpcUrl == nil || l1RpcUrl.Value != "http://localhost:8545" {
			t.Errorf("Expected L1 RPC URL to be preserved, got %v", l1RpcUrl.Value)
		}

		// Fork block
		l1ForkBlock := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block"})
		if l1ForkBlock == nil || l1ForkBlock.Value != "22475020" {
			t.Errorf("Expected L1 fork block to be preserved, got %v", l1ForkBlock.Value)
		}
	})
}

// TestAVSContextMigration_0_0_1_to_0_0_2_CustomValues tests migration when user has custom values
// that differ from defaults - these should be preserved
func TestAVSContextMigration_0_0_1_to_0_0_2_CustomValues(t *testing.T) {
	// This represents a user's devnet.yaml file with CUSTOM values (different from defaults)
	userYAML := `version: 0.0.1
context:
  chains:
    l1: 
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: "https://eth.llamarpc.com"
    l2:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: "https://eth.llamarpc.com"
  app_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  operators:
    - address: "0x1234567890123456789012345678901234567890" # CUSTOM address (different from default)
      ecdsa_key: "0x1111111111111111111111111111111111111111111111111111111111111111" # CUSTOM key
      stake: "2000ETH"
    - address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
      ecdsa_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
      stake: "1500ETH"
  avs:
    address: "0x9999999999999999999999999999999999999999" # CUSTOM AVS address
    avs_private_key: "0x2222222222222222222222222222222222222222222222222222222222222222" # CUSTOM key
    metadata_url: "https://custom-org.com/avs/metadata.json"`

	// Parse YAML nodes
	userNode := testNode(t, userYAML)

	// Get the actual migration step from the contexts package
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.1" && step.To == "0.0.2" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.1 -> 0.0.2 migration step in contexts.MigrationChain")
	}

	// Execute migration using the actual migration chain
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.1", "0.0.2", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the migration results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.2" {
			t.Errorf("Expected version to be updated to 0.0.2, got %v", version.Value)
		}
	})

	t.Run("fork URLs updated", func(t *testing.T) {
		l1ForkUrl := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "url"})
		if l1ForkUrl == nil || l1ForkUrl.Value != "" {
			t.Errorf("Expected L1 fork URL to be empty, got %v", l1ForkUrl.Value)
		}
	})

	t.Run("app_private_key updated", func(t *testing.T) {
		appKey := migration.ResolveNode(migratedNode, []string{"context", "app_private_key"})
		if appKey == nil || appKey.Value != "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" {
			t.Errorf("Expected app_private_key to be updated to new value, got %v", appKey.Value)
		}
	})

	t.Run("custom operator values preserved", func(t *testing.T) {
		// Custom operator 0 values should be preserved (they differ from old defaults)
		opAddress := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "address"})
		if opAddress == nil || opAddress.Value != "0x1234567890123456789012345678901234567890" {
			t.Errorf("Expected custom operator address to be preserved, got %v", opAddress.Value)
		}

		opKey := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "ecdsa_key"})
		if opKey == nil || opKey.Value != "0x1111111111111111111111111111111111111111111111111111111111111111" {
			t.Errorf("Expected custom operator ECDSA key to be preserved, got %v", opKey.Value)
		}

		opStake := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "stake"})
		if opStake == nil || opStake.Value != "2000ETH" {
			t.Errorf("Expected custom operator stake to be preserved, got %v", opStake.Value)
		}
	})

	t.Run("custom AVS values preserved", func(t *testing.T) {
		// Custom AVS values should be preserved (they differ from old defaults)
		avsAddress := migration.ResolveNode(migratedNode, []string{"context", "avs", "address"})
		if avsAddress == nil || avsAddress.Value != "0x9999999999999999999999999999999999999999" {
			t.Errorf("Expected custom AVS address to be preserved, got %v", avsAddress.Value)
		}

		avsKey := migration.ResolveNode(migratedNode, []string{"context", "avs", "avs_private_key"})
		if avsKey == nil || avsKey.Value != "0x2222222222222222222222222222222222222222222222222222222222222222" {
			t.Errorf("Expected custom AVS private key to be preserved, got %v", avsKey.Value)
		}

		avsMetadata := migration.ResolveNode(migratedNode, []string{"context", "avs", "metadata_url"})
		if avsMetadata == nil || avsMetadata.Value != "https://custom-org.com/avs/metadata.json" {
			t.Errorf("Expected custom AVS metadata URL to be preserved, got %v", avsMetadata.Value)
		}
	})
}

// TestAVSContextMigration_0_0_2_to_0_0_3 tests the migration from version 0.0.2 to 0.0.3
func TestAVSContextMigration_0_0_2_to_0_0_3(t *testing.T) {
	// Use the embedded v0.0.2 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.2"])

	userNode := testNode(t, userYAML)

	// Get the actual migration step from the contexts package
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.2" && step.To == "0.0.3" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.2 -> 0.0.3 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.2", "0.0.3", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.3" {
			t.Errorf("Expected version to be updated to 0.0.3, got %v", version.Value)
		}
	})

	t.Run("block_time added to L1 fork", func(t *testing.T) {
		blockTime := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block_time"})
		if blockTime == nil || blockTime.Value != "3" {
			t.Errorf("Expected L1 fork block_time to be added with value 3, got %v", blockTime.Value)
		}
	})

	t.Run("block_time added to L2 fork", func(t *testing.T) {
		blockTime := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "fork", "block_time"})
		if blockTime == nil || blockTime.Value != "3" {
			t.Errorf("Expected L2 fork block_time to be added with value 3, got %v", blockTime.Value)
		}
	})

	t.Run("existing fork values preserved", func(t *testing.T) {
		l1Block := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block"})
		if l1Block == nil || l1Block.Value != "22475020" {
			t.Errorf("Expected L1 fork block to be preserved, got %v", l1Block.Value)
		}

		l1Url := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "url"})
		if l1Url == nil || l1Url.Value != "" {
			t.Errorf("Expected L1 fork URL to be preserved as empty, got %v", l1Url.Value)
		}
	})
}

// TestAVSContextMigration_0_0_3_to_0_0_4 tests the migration from version 0.0.3 to 0.0.4
// which adds the eigenlayer section with contract addresses
func TestAVSContextMigration_0_0_3_to_0_0_4(t *testing.T) {
	// Use the embedded v0.0.3 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.3"])

	userNode := testNode(t, userYAML)

	// Get the actual migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.3" && step.To == "0.0.4" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.3 -> 0.0.4 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.3", "0.0.4", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.4" {
			t.Errorf("Expected version to be updated to 0.0.4, got %v", version.Value)
		}
	})

	t.Run("eigenlayer section added", func(t *testing.T) {
		eigenlayer := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer"})
		if eigenlayer == nil {
			t.Error("Expected eigenlayer section to be added")
			return
		}

		// Check specific contract addresses
		allocMgr := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "allocation_manager"})
		if allocMgr == nil || allocMgr.Value != "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39" {
			t.Errorf("Expected allocation_manager address, got %v", allocMgr.Value)
		}

		delegMgr := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "delegation_manager"})
		if delegMgr == nil || delegMgr.Value != "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A" {
			t.Errorf("Expected delegation_manager address, got %v", delegMgr.Value)
		}
	})

	t.Run("existing configuration preserved", func(t *testing.T) {
		// Ensure existing configs aren't affected
		blockTime := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block_time"})
		if blockTime == nil || blockTime.Value != "3" {
			t.Errorf("Expected existing block_time to be preserved, got %v", blockTime.Value)
		}
	})
}

// TestAVSContextMigration_0_0_4_to_0_0_5 tests the migration from version 0.0.4 to 0.0.5
// which adds deployed_contracts, operator_sets, and operator_registrations sections
func TestAVSContextMigration_0_0_4_to_0_0_5(t *testing.T) {
	// Use the embedded v0.0.4 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.4"])

	userNode := testNode(t, userYAML)

	// Get the actual migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.4" && step.To == "0.0.5" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.4 -> 0.0.5 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.4", "0.0.5", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.5" {
			t.Errorf("Expected version to be updated to 0.0.5, got %v", version.Value)
		}
	})

	t.Run("deployed_contracts section added", func(t *testing.T) {
		deployedContracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_contracts"})
		if deployedContracts == nil {
			t.Error("Expected deployed_contracts section to be added")
		}
	})

	t.Run("operator_sets section added", func(t *testing.T) {
		operatorSets := migration.ResolveNode(migratedNode, []string{"context", "operator_sets"})
		if operatorSets == nil {
			t.Error("Expected operator_sets section to be added")
		}
	})

	t.Run("operator_registrations section added", func(t *testing.T) {
		operatorRegs := migration.ResolveNode(migratedNode, []string{"context", "operator_registrations"})
		if operatorRegs == nil {
			t.Error("Expected operator_registrations section to be added")
		}
	})
}

// TestAVSContextMigration_0_0_5_to_0_0_6 tests the migration from version 0.0.5 to 0.0.6
// which updates fork blocks, adds strategy_manager, and converts stake to allocations
func TestAVSContextMigration_0_0_5_to_0_0_6(t *testing.T) {
	// Use the embedded v0.0.5 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.5"])

	userNode := testNode(t, userYAML)

	// Get the actual migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.5" && step.To == "0.0.6" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.5 -> 0.0.6 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.5", "0.0.6", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.6" {
			t.Errorf("Expected version to be updated to 0.0.6, got %v", version.Value)
		}
	})

	t.Run("fork blocks updated", func(t *testing.T) {
		l1Block := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block"})
		if l1Block == nil || l1Block.Value != "4017700" {
			t.Errorf("Expected L1 fork block to be updated to 4017700, got %v", l1Block.Value)
		}

		l2Block := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "fork", "block"})
		if l2Block == nil || l2Block.Value != "4017700" {
			t.Errorf("Expected L2 fork block to be updated to 4017700, got %v", l2Block.Value)
		}
	})

	t.Run("strategy_manager added to eigenlayer L1", func(t *testing.T) {
		strategyMgr := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "strategy_manager"})
		if strategyMgr == nil || strategyMgr.Value != "0xdfB5f6CE42aAA7830E94ECFCcAd411beF4d4D5b6" {
			t.Errorf("Expected strategy_manager to be added to L1, got %v", strategyMgr.Value)
		}
	})

	t.Run("operators converted from stake to allocations", func(t *testing.T) {
		// Check that operator 0 (first operator) has allocations structure
		allocations := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations"})
		if allocations == nil {
			t.Error("Expected operator 0 to have allocations structure")
			return
		}

		// Check first allocation details
		strategyAddr := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations", "0", "strategy_address"})
		if strategyAddr == nil || strategyAddr.Value != "0x7D704507b76571a51d9caE8AdDAbBFd0ba0e63d3" {
			t.Errorf("Expected stETH strategy address, got %v", strategyAddr.Value)
		}

		strategyName := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations", "0", "name"})
		if strategyName == nil || strategyName.Value != "stETH_Strategy" {
			t.Errorf("Expected strategy name to be stETH_Strategy, got %v", strategyName.Value)
		}
		// Check operator set allocation
		opSetAlloc := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations", "0", "operator_set_allocations", "0", "operator_set"})
		if opSetAlloc == nil || opSetAlloc.Value != "0" {
			t.Errorf("Expected operator set to be 0, got %v", opSetAlloc.Value)
		}

		allocationWads := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations", "0", "operator_set_allocations", "0", "allocation_in_wads"})
		if allocationWads == nil || allocationWads.Value != "500000000000000000" {
			t.Errorf("Expected allocation in wads to be 500000000000000000, got %v", allocationWads.Value)
		}
	})

	t.Run("stake field removed from operators", func(t *testing.T) {
		// The migration replaces entire operator structures, but may leave empty stake fields
		for i := 0; i < 5; i++ {
			stake := migration.ResolveNode(migratedNode, []string{"context", "operators", fmt.Sprintf("%d", i), "stake"})
			if stake != nil && stake.Value != "" {
				t.Errorf("Expected stake field to be removed or empty for operator %d, but got value %v", i, stake.Value)
			}
		}
	})

	t.Run("operator 1 has stETH allocation", func(t *testing.T) {
		// Check that operator 1 also has stETH strategy allocation (same as operator 0)
		strategyAddr := migration.ResolveNode(migratedNode, []string{"context", "operators", "1", "allocations", "0", "strategy_address"})
		if strategyAddr == nil || strategyAddr.Value != "0x7D704507b76571a51d9caE8AdDAbBFd0ba0e63d3" {
			t.Errorf("Expected stETH strategy address for operator 1, got %v", strategyAddr.Value)
		}

		strategyName := migration.ResolveNode(migratedNode, []string{"context", "operators", "1", "allocations", "0", "name"})
		if strategyName == nil || strategyName.Value != "stETH_Strategy" {
			t.Errorf("Expected strategy name to be stETH_Strategy for operator 1, got %v", strategyName.Value)
		}
	})

	t.Run("operators 2-4 have no meaningful allocations", func(t *testing.T) {
		// Operators 2, 3, 4 should have no meaningful allocations
		for i := 2; i < 5; i++ {
			allocations := migration.ResolveNode(migratedNode, []string{"context", "operators", fmt.Sprintf("%d", i), "allocations"})
			if allocations != nil {
				// If allocations exist, check that they're empty (no items in the sequence)
				if allocations.Kind == yaml.SequenceNode && len(allocations.Content) > 0 {
					t.Errorf("Expected operator %d to have empty allocations, but got %d items", i, len(allocations.Content))
				}
			}

			// But they should still be there as operator objects
			operator := migration.ResolveNode(migratedNode, []string{"context", "operators", fmt.Sprintf("%d", i)})
			if operator == nil {
				t.Errorf("Expected operator %d to still exist", i)
			}
		}
	})

	t.Run("eigenlayer converted to L1/L2 structure", func(t *testing.T) {
		// Check that eigenlayer now has L1/L2 structure
		allocationMgr := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "allocation_manager"})
		if allocationMgr == nil || allocationMgr.Value != "0xFdD5749e11977D60850E06bF5B13221Ad95eb6B4" {
			t.Errorf("Expected allocation_manager in L1 structure, got %v", allocationMgr.Value)
		}

		delegationMgr := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "delegation_manager"})
		if delegationMgr == nil || delegationMgr.Value != "0x75dfE5B44C2E530568001400D3f704bC8AE350CC" {
			t.Errorf("Expected delegation_manager in L1 structure, got %v", delegationMgr.Value)
		}

		// Check L2 contracts exist
		certVerifier := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"})
		if certVerifier == nil || certVerifier.Value != "0xf462d03A82C1F3496B0DFe27E978318eD1720E1f" {
			t.Errorf("Expected bn254_certificate_verifier in L2 structure, got %v", certVerifier.Value)
		}

		// Check that operator sets are preserved
		operatorSets := migration.ResolveNode(migratedNode, []string{"context", "operator_sets"})
		if operatorSets == nil {
			t.Error("Expected operator_sets section to be preserved")
		}
	})

	t.Run("transporter section added with expected keys", func(t *testing.T) {
		schedule := migration.ResolveNode(migratedNode, []string{"context", "transporter", "schedule"})
		if schedule == nil || schedule.Value != "0 */2 * * *" {
			t.Errorf("Expected schedule '0 */2 * * *', got %v", schedule.Value)
		}
		privKey := migration.ResolveNode(migratedNode, []string{"context", "transporter", "private_key"})
		if privKey == nil {
			t.Error("Expected private_key field to be present")
		}
		blsPrivKey := migration.ResolveNode(migratedNode, []string{"context", "transporter", "bls_private_key"})
		if blsPrivKey == nil {
			t.Error("Expected bls_private_key field to be present")
		}
	})

	t.Run("transporter inserted after chains", func(t *testing.T) {
		ctxNode := migration.ResolveNode(migratedNode, []string{"context"})
		if ctxNode == nil || ctxNode.Kind != yaml.MappingNode {
			t.Fatal("context node not found or invalid")
		}
		var keys []string
		for i := 0; i < len(ctxNode.Content)-1; i += 2 {
			keys = append(keys, ctxNode.Content[i].Value)
		}
		chainsIdx, transpIdx := -1, -1
		for i, key := range keys {
			if key == "chains" {
				chainsIdx = i
			}
			if key == "transporter" {
				transpIdx = i
			}
		}
		if chainsIdx == -1 || transpIdx == -1 {
			t.Fatal("chains or transporter key missing in context")
		}
		if transpIdx <= chainsIdx {
			t.Errorf("Expected transporter to appear after chains, got chains at %d, transporter at %d", chainsIdx, transpIdx)
		}
	})
}

// TestAVSContextMigration_0_0_6_to_0_0_7 tests the migration from version 0.0.6 to 0.0.7
// which adds the artifact section
func TestAVSContextMigration_0_0_6_to_0_0_7(t *testing.T) {
	// Use the embedded v0.0.6 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.6"])

	userNode := testNode(t, userYAML)

	// Get the actual migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.6" && step.To == "0.0.7" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.6 -> 0.0.7 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.6", "0.0.7", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.7" {
			t.Errorf("Expected version to be updated to 0.0.7, got %v", version.Value)
		}
	})

	t.Run("artifact section added", func(t *testing.T) {
		artifacts := migration.ResolveNode(migratedNode, []string{"context", "artifact"})
		if artifacts == nil {
			t.Error("Expected artifacts section to be added")
		}
	})
}

// TestAVSContextMigration_FullChain tests migrating through the entire chain from 0.0.1 to 0.0.6
func TestAVSContextMigration_FullChain(t *testing.T) {
	// Use the embedded v0.0.1 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.1"])

	userNode := testNode(t, userYAML)

	// Execute migration through the entire chain to 0.0.6 (where stake conversion happens)
	migratedNode, err := migration.MigrateNode(userNode, "0.0.1", "0.0.6", contexts.MigrationChain)
	if err != nil {
		t.Fatalf("Full chain migration failed: %v", err)
	}

	// Verify final state
	t.Run("final version is 0.0.6", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.6" {
			t.Errorf("Expected final version to be 0.0.6, got %v", version.Value)
		}
	})

	t.Run("all features added through chain", func(t *testing.T) {
		// Check that block_time was added (from 0.0.2→0.0.3)
		blockTime := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block_time"})
		if blockTime == nil || blockTime.Value != "3" {
			t.Errorf("Expected block_time to be added, got %v", blockTime.Value)
		}

		// Check that eigenlayer was added (from 0.0.3→0.0.4)
		eigenlayer := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer"})
		if eigenlayer == nil {
			t.Error("Expected eigenlayer section to be added")
		}

		// Check that tracking sections were added (from 0.0.4→0.0.5)
		deployedContracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_contracts"})
		if deployedContracts == nil {
			t.Error("Expected deployed_contracts section to be added")
		}

		// Check that strategy_manager was added (from 0.0.5→0.0.6)
		strategyManager := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "strategy_manager"})
		if strategyManager == nil {
			t.Error("Expected strategy_manager to be added to L1 structure")
		}
	})

	t.Run("stake converted to allocations", func(t *testing.T) {
		// Check that the original stake was converted to allocations structure
		allocations := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "allocations"})
		if allocations == nil {
			t.Error("Expected operator to have allocations structure after full migration")
			return
		}

		// Verify stake field is completely removed or empty
		stake := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "stake"})
		if stake != nil && stake.Value != "" {
			t.Errorf("Expected stake field to be removed or empty after migration, but got %v", stake.Value)
		}
	})
}
