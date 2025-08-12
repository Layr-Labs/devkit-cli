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

	t.Run("l1 fork block updated to 8713384", func(t *testing.T) {
		l1Block := migration.ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block"})
		if l1Block == nil || l1Block.Value != "8713384" {
			t.Errorf("Expected L1 fork block to be updated to 8713384, got %v", l1Block.Value)
		}
	})

	t.Run("l2 fork block updated to 28069764", func(t *testing.T) {
		l2Block := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "fork", "block"})
		if l2Block == nil || l2Block.Value != "28069764" {
			t.Errorf("Expected L2 fork block to be updated to 28069764, got %v", l2Block.Value)
		}
	})
	t.Run("L2 chain id updated to 31338", func(t *testing.T) {
		l2ChainId := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "chain_id"})
		if l2ChainId == nil || l2ChainId.Value != "31338" {
			t.Errorf("Expected L2 chain id to be updated to 31338, got %v", l2ChainId.Value)
		}
	})
	t.Run("L2 rpc url updated to http://localhost:9545", func(t *testing.T) {
		l2RpcUrl := migration.ResolveNode(migratedNode, []string{"context", "chains", "l2", "rpc_url"})
		if l2RpcUrl == nil || l2RpcUrl.Value != "http://localhost:9545" {
			t.Errorf("Expected L2 rpc url to be updated to http://localhost:9545, got %v", l2RpcUrl.Value)
		}
	})
	t.Run("bn254_certificate_verifier updated to 0x998535833f3feE44ce720440E735554699f728a5", func(t *testing.T) {
		bn254CertVerifier := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"})
		if bn254CertVerifier == nil || bn254CertVerifier.Value != "0x998535833f3feE44ce720440E735554699f728a5" {
			t.Errorf("Expected bn254_certificate_verifier to be updated to 0x998535833f3feE44ce720440E735554699f728a5, got %v", bn254CertVerifier.Value)
		}
	})
	t.Run("operator_table_updater updated to 0xE12C4cebd680a917271145eDbFB091B1BdEFD74D", func(t *testing.T) {
		operatorTableUpdater := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l2", "operator_table_updater"})
		if operatorTableUpdater == nil || operatorTableUpdater.Value != "0xE12C4cebd680a917271145eDbFB091B1BdEFD74D" {
			t.Errorf("Expected operator_table_updater to be updated to 0xE12C4cebd680a917271145eDbFB091B1BdEFD74D, got %v", operatorTableUpdater.Value)
		}
	})
	t.Run("Added ecdsa_certificate_verifier with address 0xAD2F58A551bD0e77fa20b5531dA96eF440C392BF", func(t *testing.T) {
		ecdsaCertVerifier := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"})
		if ecdsaCertVerifier == nil || ecdsaCertVerifier.Value != "0xAD2F58A551bD0e77fa20b5531dA96eF440C392BF" {
			t.Errorf("Expected ecdsa_certificate_verifier to be added  with address 0xAD2F58A551bD0e77fa20b5531dA96eF440C392BF, got %v", ecdsaCertVerifier.Value)
		}
	})
	t.Run("deployed_l1_contracts section added", func(t *testing.T) {
		deployedL1Contracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_l1_contracts"})
		if deployedL1Contracts == nil {
			t.Error("Expected deployed_l1_contracts section to be added")
		}
	})
	t.Run("deployed_l2_contracts section added", func(t *testing.T) {
		deployedL2Contracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_l2_contracts"})
		if deployedL2Contracts == nil {
			t.Error("Expected deployed_l2_contracts section to be added")
		}
	})
	t.Run("deployed_contracts section removed", func(t *testing.T) {
		deployedContracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_contracts"})
		if deployedContracts != nil {
			t.Errorf("Expected deployed_contracts section to be removed, got %v", deployedContracts.Value)
		}
	})
	t.Run("allocation_manager updated to 0x42583067658071247ec8CE0A516A58f682002d07", func(t *testing.T) {
		allocationManager := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "allocation_manager"})
		if allocationManager == nil || allocationManager.Value != "0x42583067658071247ec8CE0A516A58f682002d07" {
			t.Errorf("Expected allocation_manager to be updated to 0x42583067658071247ec8CE0A516A58f682002d07, got %v", allocationManager.Value)
		}
	})
	t.Run("delegation_manager updated to 0xD4A7E1Bd8015057293f0D0A557088c286942e84b", func(t *testing.T) {
		delegationManager := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "delegation_manager"})
		if delegationManager == nil || delegationManager.Value != "0xD4A7E1Bd8015057293f0D0A557088c286942e84b" {
			t.Errorf("Expected delegation_manager to be updated to 0xD4A7E1Bd8015057293f0D0A557088c286942e84b, got %v", delegationManager.Value)
		}
	})
	t.Run("strategy_manager updated to 0x2E3D6c0744b10eb0A4e6F679F71554a39Ec47a5D", func(t *testing.T) {
		strategyManager := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "strategy_manager"})
		if strategyManager == nil || strategyManager.Value != "0x2E3D6c0744b10eb0A4e6F679F71554a39Ec47a5D" {
			t.Errorf("Expected strategy_manager to be updated to 0x2E3D6c0744b10eb0A4e6F679F71554a39Ec47a5D, got %v", strategyManager.Value)
		}
	})
	t.Run("bn254_table_calculator updated to 0xc2c0bc13571aC5115709C332dc7AE666606b08E8", func(t *testing.T) {
		bn254TableCalculator := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "bn254_table_calculator"})
		if bn254TableCalculator == nil || bn254TableCalculator.Value != "0xc2c0bc13571aC5115709C332dc7AE666606b08E8" {
			t.Errorf("Expected bn254_table_calculator to be updated to 0xc2c0bc13571aC5115709C332dc7AE666606b08E8, got %v", bn254TableCalculator.Value)
		}
	})
	t.Run("cross_chain_registry updated to 0xe850D8A178777b483D37fD492a476e3E6004C816", func(t *testing.T) {
		crossChainRegistry := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "cross_chain_registry"})
		if crossChainRegistry == nil || crossChainRegistry.Value != "0xe850D8A178777b483D37fD492a476e3E6004C816" {
			t.Errorf("Expected cross_chain_registry to be updated to 0xe850D8A178777b483D37fD492a476e3E6004C816, got %v", crossChainRegistry.Value)
		}
	})
	t.Run("key_registrar updated to 0x78De554Ac8DfF368e3CAa73B3Df8AccCfD92928A", func(t *testing.T) {
		keyRegistrar := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "key_registrar"})
		if keyRegistrar == nil || keyRegistrar.Value != "0x78De554Ac8DfF368e3CAa73B3Df8AccCfD92928A" {
			t.Errorf("Expected key_registrar to be updated to 0x78De554Ac8DfF368e3CAa73B3Df8AccCfD92928A, got %v", keyRegistrar.Value)
		}
	})
	t.Run("release_manager updated to 0xd9Cb89F1993292dEC2F973934bC63B0f2A702776", func(t *testing.T) {
		releaseManager := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "release_manager"})
		if releaseManager == nil || releaseManager.Value != "0xd9Cb89F1993292dEC2F973934bC63B0f2A702776" {
			t.Errorf("Expected release_manager to be updated to 0xd9Cb89F1993292dEC2F973934bC63B0f2A702776, got %v", releaseManager.Value)
		}
	})
	t.Run("operator_table_updater updated to 0xE12C4cebd680a917271145eDbFB091B1BdEFD74D", func(t *testing.T) {
		operatorTableUpdater := migration.ResolveNode(migratedNode, []string{"context", "eigenlayer", "l1", "operator_table_updater"})
		if operatorTableUpdater == nil || operatorTableUpdater.Value != "0xE12C4cebd680a917271145eDbFB091B1BdEFD74D" {
			t.Errorf("Expected operator_table_updater to be updated to 0xE12C4cebd680a917271145eDbFB091B1BdEFD74D, got %v", operatorTableUpdater.Value)
		}
	})
}

// TestAVSContextMigration_0_0_7_to_0_0_8 tests the migration that adds ECDSA keystore support
func TestAVSContextMigration_0_0_7_to_0_0_8(t *testing.T) {
	// Use v0.0.7 content as starting point
	userYAML := string(contexts.ContextYamls["0.0.7"])
	userNode := testNode(t, userYAML)

	// Get the actual migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.7" && step.To == "0.0.8" {
			migrationStep = step
			break
		}
	}
	if migrationStep.Apply == nil {
		t.Fatal("Could not find 0.0.7 -> 0.0.8 migration step")
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.7", "0.0.8", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify results
	t.Run("version updated", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.8" {
			t.Errorf("Expected version to be updated to 0.0.8, got %v", version.Value)
		}
	})

	t.Run("ECDSA keystore fields added to operators", func(t *testing.T) {
		operators := migration.ResolveNode(migratedNode, []string{"context", "operators"})
		if operators == nil || operators.Kind != yaml.SequenceNode {
			t.Fatal("Expected operators to exist and be a sequence")
		}

		// Check first operator has ECDSA keystore fields
		if len(operators.Content) > 0 {
			firstOp := operators.Content[0]

			// Check for ECDSA keystore path
			ecdsaKeystorePath := migration.ResolveNode(firstOp, []string{"ecdsa_keystore_path"})
			if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator1.ecdsa.keystore.json" {
				t.Errorf("Expected ECDSA keystore path for operator1, got %v", ecdsaKeystorePath)
			}

			// Check for ECDSA keystore password
			ecdsaKeystorePassword := migration.ResolveNode(firstOp, []string{"ecdsa_keystore_password"})
			if ecdsaKeystorePassword == nil || ecdsaKeystorePassword.Value != "testpass" {
				t.Errorf("Expected ECDSA keystore password 'testpass', got %v", ecdsaKeystorePassword)
			}
		}

		// Check second operator
		if len(operators.Content) > 1 {
			secondOp := operators.Content[1]

			ecdsaKeystorePath := migration.ResolveNode(secondOp, []string{"ecdsa_keystore_path"})
			if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator2.ecdsa.keystore.json" {
				t.Errorf("Expected ECDSA keystore path for operator2, got %v", ecdsaKeystorePath)
			}
		}
	})

	t.Run("BLS keystore paths updated to new naming convention", func(t *testing.T) {
		operators := migration.ResolveNode(migratedNode, []string{"context", "operators"})
		if operators == nil || operators.Kind != yaml.SequenceNode {
			t.Fatal("Expected operators to exist and be a sequence")
		}

		// Check first operator's BLS keystore path
		if len(operators.Content) > 0 {
			firstOp := operators.Content[0]
			blsKeystorePath := migration.ResolveNode(firstOp, []string{"bls_keystore_path"})
			if blsKeystorePath == nil || blsKeystorePath.Value != "keystores/operator1.bls.keystore.json" {
				t.Errorf("Expected BLS keystore path to be updated, got %v", blsKeystorePath)
			}
		}
	})

	t.Run("existing fields preserved", func(t *testing.T) {
		// Check that existing operator fields are preserved
		firstOp := migration.ResolveNode(migratedNode, []string{"context", "operators", "0"})
		if firstOp == nil {
			t.Fatal("Expected first operator to exist")
		}

		// Check address is preserved
		address := migration.ResolveNode(firstOp, []string{"address"})
		if address == nil || address.Value != "0x90F79bf6EB2c4f870365E785982E1f101E93b906" {
			t.Errorf("Expected operator address to be preserved, got %v", address)
		}

		// Check ECDSA key is preserved
		ecdsaKey := migration.ResolveNode(firstOp, []string{"ecdsa_key"})
		if ecdsaKey == nil || ecdsaKey.Value != "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" {
			t.Errorf("Expected ECDSA key to be preserved, got %v", ecdsaKey)
		}

		// Check allocations are preserved
		allocations := migration.ResolveNode(firstOp, []string{"allocations"})
		if allocations == nil {
			t.Error("Expected allocations to be preserved")
		}
	})
}

// TestAVSContextMigration_0_0_7_to_0_0_8_WithCustomValues tests migration with custom operator values
func TestAVSContextMigration_0_0_7_to_0_0_8_WithCustomValues(t *testing.T) {
	// Create a custom v0.0.7 YAML with some custom values
	customYAML := `version: 0.0.7
context:
  name: "custom-context"
  operators:
    - address: "0xCUSTOM_ADDRESS_1"
      ecdsa_key: "0xCUSTOM_ECDSA_KEY_1"
      bls_keystore_path: "custom/path/operator1.keystore.json"
      bls_keystore_password: "custompass1"
      custom_field: "custom_value_1"
    - address: "0xCUSTOM_ADDRESS_2"
      ecdsa_key: "0xCUSTOM_ECDSA_KEY_2"
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
    - address: "0xCUSTOM_ADDRESS_3"
      ecdsa_key: "0xCUSTOM_ECDSA_KEY_3"
      bls_keystore_path: "keystores/custom.keystore.json"
      bls_keystore_password: "custompass3"
  artifact:
    registry: "custom-registry"
`

	userNode := testNode(t, customYAML)

	// Get the migration step
	var migrationStep migration.MigrationStep
	for _, step := range contexts.MigrationChain {
		if step.From == "0.0.7" && step.To == "0.0.8" {
			migrationStep = step
			break
		}
	}

	// Execute migration
	migrationChain := []migration.MigrationStep{migrationStep}
	migratedNode, err := migration.MigrateNode(userNode, "0.0.7", "0.0.8", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	t.Run("custom values preserved with ECDSA keystore fields added", func(t *testing.T) {
		// Check first operator with custom values
		firstOp := migration.ResolveNode(migratedNode, []string{"context", "operators", "0"})

		// Custom address preserved
		address := migration.ResolveNode(firstOp, []string{"address"})
		if address == nil || address.Value != "0xCUSTOM_ADDRESS_1" {
			t.Errorf("Expected custom address to be preserved, got %v", address)
		}

		// Custom ECDSA key preserved
		ecdsaKey := migration.ResolveNode(firstOp, []string{"ecdsa_key"})
		if ecdsaKey == nil || ecdsaKey.Value != "0xCUSTOM_ECDSA_KEY_1" {
			t.Errorf("Expected custom ECDSA key to be preserved, got %v", ecdsaKey)
		}

		// BLS keystore path updated even for custom paths if they match the pattern
		blsPath := migration.ResolveNode(firstOp, []string{"bls_keystore_path"})
		if blsPath == nil || blsPath.Value != "keystores/operator1.bls.keystore.json" {
			t.Errorf("Expected BLS path to be updated to new convention, got %v", blsPath)
		}

		// Custom field preserved
		customField := migration.ResolveNode(firstOp, []string{"custom_field"})
		if customField == nil || customField.Value != "custom_value_1" {
			t.Errorf("Expected custom field to be preserved, got %v", customField)
		}

		// ECDSA keystore fields added based on position
		ecdsaKeystorePath := migration.ResolveNode(firstOp, []string{"ecdsa_keystore_path"})
		if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator1.ecdsa.keystore.json" {
			t.Errorf("Expected ECDSA keystore path to be added, got %v", ecdsaKeystorePath)
		}
	})

	t.Run("standard operator paths updated", func(t *testing.T) {
		// Check second operator with standard path
		secondOp := migration.ResolveNode(migratedNode, []string{"context", "operators", "1"})

		// Custom values preserved
		address := migration.ResolveNode(secondOp, []string{"address"})
		if address == nil || address.Value != "0xCUSTOM_ADDRESS_2" {
			t.Errorf("Expected custom address to be preserved, got %v", address)
		}

		blsPath := migration.ResolveNode(secondOp, []string{"bls_keystore_path"})
		if blsPath == nil || blsPath.Value != "keystores/operator2.bls.keystore.json" {
			t.Errorf("Expected standard BLS path to be updated, got %v", blsPath)
		}

		// ECDSA keystore fields added
		ecdsaKeystorePath := migration.ResolveNode(secondOp, []string{"ecdsa_keystore_path"})
		if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator2.ecdsa.keystore.json" {
			t.Errorf("Expected ECDSA keystore path to be added, got %v", ecdsaKeystorePath)
		}
	})

	t.Run("third operator with custom values", func(t *testing.T) {
		// Check third operator
		thirdOp := migration.ResolveNode(migratedNode, []string{"context", "operators", "2"})

		// Custom address preserved
		address := migration.ResolveNode(thirdOp, []string{"address"})
		if address == nil || address.Value != "0xCUSTOM_ADDRESS_3" {
			t.Errorf("Expected custom address to be preserved, got %v", address)
		}

		// Has ECDSA keystore fields based on position
		ecdsaKeystorePath := migration.ResolveNode(thirdOp, []string{"ecdsa_keystore_path"})
		if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator3.ecdsa.keystore.json" {
			t.Errorf("Expected operator3 ECDSA keystore path, got %v", ecdsaKeystorePath)
		}
	})

	t.Run("custom context name preserved", func(t *testing.T) {
		name := migration.ResolveNode(migratedNode, []string{"context", "name"})
		if name == nil || name.Value != "custom-context" {
			t.Errorf("Expected custom context name to be preserved, got %v", name)
		}
	})
}

func TestAVSContextMigration_0_0_8_to_0_0_9_PatchesAndAddsSkipSetup(t *testing.T) {
	// v0.0.8 input with existing structure and old values
	const in = `version: 0.0.8
context:
  name: "pre-009"
  chains:
    l1:
      fork:
        block: "11111111"
    l2:
      fork:
        block: "22222222"
  eigenlayer:
    l1:
      cross_chain_registry: "0xOLD_L1_C_C_R"
      operator_table_updater: "0xOLD_L1_O_T_U"
      key_registrar: "0xOLD_L1_K_R"
      bn254_table_calculator: "0xOLD_L1_BN254_T_C"
      ecdsa_table_calculator: "0xOLD_L1_ECDSA_T_C"
    l2:
      task_mailbox: "0xOLD_L2_T_M"
      operator_table_updater: "0xOLD_L2_O_T_U"
      bn254_certificate_verifier: "0xOLD_L2_BN254_C_V"
      ecdsa_certificate_verifier: "0xOLD_L2_ECDSA_C_V"
  avs:
    address: "0xAVS"
`

	userNode := testNode(t, in)

	// find the step 0.0.8 -> 0.0.9
	var step migration.MigrationStep
	for _, s := range contexts.MigrationChain {
		if s.From == "0.0.8" && s.To == "0.0.9" {
			step = s
			break
		}
	}
	if step.Apply == nil {
		t.Fatalf("migration step 0.0.8 -> 0.0.9 not found")
	}

	out, err := migration.MigrateNode(userNode, "0.0.8", "0.0.9", []migration.MigrationStep{step})
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// print migrated YAML for debugging
	if b, err := yaml.Marshal(out); err == nil {
		t.Log("\n" + string(b))
	}

	t.Run("version bumped", func(t *testing.T) {
		v := migration.ResolveNode(out, []string{"version"})
		if v == nil || v.Value != "0.0.9" {
			t.Errorf("want version 0.0.9, got %v", v)
		}
	})

	t.Run("fork blocks updated", func(t *testing.T) {
		l1 := migration.ResolveNode(out, []string{"context", "chains", "l1", "fork", "block"})
		l2 := migration.ResolveNode(out, []string{"context", "chains", "l2", "fork", "block"})
		if l1 == nil || l1.Value != "8836193" {
			t.Errorf("l1.fork.block want 8836193, got %v", l1)
		}
		if l2 == nil || l2.Value != "28820370" {
			t.Errorf("l2.fork.block want 28820370, got %v", l2)
		}
	})

	t.Run("L1 addresses updated", func(t *testing.T) {
		C_C_R := migration.ResolveNode(out, []string{"context", "eigenlayer", "l1", "cross_chain_registry"})
		O_T_U := migration.ResolveNode(out, []string{"context", "eigenlayer", "l1", "operator_table_updater"})
		K_R := migration.ResolveNode(out, []string{"context", "eigenlayer", "l1", "key_registrar"})
		B_N := migration.ResolveNode(out, []string{"context", "eigenlayer", "l1", "bn254_table_calculator"})
		E_C := migration.ResolveNode(out, []string{"context", "eigenlayer", "l1", "ecdsa_table_calculator"})

		if C_C_R == nil || C_C_R.Value != "0x287381B1570d9048c4B4C7EC94d21dDb8Aa1352a" {
			t.Errorf("l1.cross_chain_registry updated wrong, got %v", C_C_R)
		}
		if O_T_U == nil || O_T_U.Value != "0xB02A15c6Bd0882b35e9936A9579f35FB26E11476" {
			t.Errorf("l1.operator_table_updater updated wrong, got %v", O_T_U)
		}
		if K_R == nil || K_R.Value != "0xA4dB30D08d8bbcA00D40600bee9F029984dB162a" {
			t.Errorf("l1.key_registrar updated wrong, got %v", K_R)
		}
		if B_N == nil || B_N.Value != "0xa19E3B00cf4aC46B5e6dc0Bbb0Fb0c86D0D65603" {
			t.Errorf("l1.bn254_table_calculator updated wrong, got %v", B_N)
		}
		if E_C == nil || E_C.Value != "0xaCB5DE6aa94a1908E6FA577C2ade65065333B450" {
			t.Errorf("l1.ecdsa_table_calculator updated wrong, got %v", E_C)
		}
	})

	t.Run("L2 addresses updated", func(t *testing.T) {
		T_M := migration.ResolveNode(out, []string{"context", "eigenlayer", "l2", "task_mailbox"})
		O_T_U := migration.ResolveNode(out, []string{"context", "eigenlayer", "l2", "operator_table_updater"})
		B_N := migration.ResolveNode(out, []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"})
		E_C := migration.ResolveNode(out, []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"})

		if T_M == nil || T_M.Value != "0xB99CC53e8db7018f557606C2a5B066527bF96b26" {
			t.Errorf("l2.task_mailbox updated wrong, got %v", T_M)
		}
		if O_T_U == nil || O_T_U.Value != "0xB02A15c6Bd0882b35e9936A9579f35FB26E11476" {
			t.Errorf("l2.operator_table_updater updated wrong, got %v", O_T_U)
		}
		if B_N == nil || B_N.Value != "0xff58A373c18268F483C1F5cA03Cf885c0C43373a" {
			t.Errorf("l2.bn254_certificate_verifier updated wrong, got %v", B_N)
		}
		if E_C == nil || E_C.Value != "0xb3Cd1A457dEa9A9A6F6406c6419B1c326670A96F" {
			t.Errorf("l2.ecdsa_certificate_verifier updated wrong, got %v", E_C)
		}
	})

	t.Run("unrelated fields preserved", func(t *testing.T) {
		name := migration.ResolveNode(out, []string{"context", "name"})
		if name == nil || name.Value != "pre-009" {
			t.Errorf("context.name mutated, got %v", name)
		}
	})
}

func TestAVSContextMigration_0_0_9_to_0_1_0_FixedOpSets(t *testing.T) {
	// v0.0.9 input with flat keystore fields
	customYAML := `version: 0.0.9
context:
  name: "custom-context"
  avs:
    address: "0xAVS_ADDR"
  operators:
    - address: "0xOP1"
      ecdsa_key: "0xECDSAKEY1"
      ecdsa_keystore_path: "keystores/operator1.ecdsa.keystore.json"
      ecdsa_keystore_password: "pass1"
      bls_keystore_path: "keystores/operator1.bls.keystore.json"
      bls_keystore_password: "bpass1"
      custom_field: "keepme1"
    - address: "0xOP2"
      ecdsa_key: "0xECDSAKEY2"
      ecdsa_keystore_path: "keystores/operator2.ecdsa.keystore.json"
      ecdsa_keystore_password: "pass2"
      bls_keystore_path: "keystores/operator2.bls.keystore.json"
      bls_keystore_password: "bpass2"
  allocations:
    - strategy_address: "0xSTRAT"
      name: "stETH_Strategy"
      operator_set_allocations:
        - operator_set: "0"
          allocation_in_wads: "500000000000000000"
        - operator_set: "1"
          allocation_in_wads: "500000000000000000"
`

	userNode := testNode(t, customYAML)

	// locate the 0.0.9 -> 0.1.0 step from the chain
	var step migration.MigrationStep
	for _, s := range contexts.MigrationChain {
		if s.From == "0.0.9" && s.To == "0.1.0" {
			step = s
			break
		}
	}
	if step.Apply == nil {
		t.Fatalf("migration step 0.0.9 -> 0.1.0 not found")
	}

	migrated, err := migration.MigrateNode(userNode, "0.0.9", "0.1.0", []migration.MigrationStep{step})
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	t.Run("version bumped", func(t *testing.T) {
		v := migration.ResolveNode(migrated, []string{"version"})
		if v == nil || v.Value != "0.1.0" {
			t.Errorf("expected version 0.1.0, got %v", v)
		}
	})

	t.Run("operator 1 keystores created and fields preserved", func(t *testing.T) {
		op := migration.ResolveNode(migrated, []string{"context", "operators", "0"})
		if op == nil {
			t.Fatal("operator[0] missing")
		}

		// preserved fields
		if got := migration.ResolveNode(op, []string{"address"}); got == nil || got.Value != "0xOP1" {
			t.Errorf("address not preserved, got %v", got)
		}
		if got := migration.ResolveNode(op, []string{"ecdsa_key"}); got == nil || got.Value != "0xECDSAKEY1" {
			t.Errorf("ecdsa_key not preserved, got %v", got)
		}
		if got := migration.ResolveNode(op, []string{"custom_field"}); got == nil || got.Value != "keepme1" {
			t.Errorf("custom_field not preserved, got %v", got)
		}

		// old flat fields removed
		for _, k := range []string{"ecdsa_keystore_path", "ecdsa_keystore_password", "bls_keystore_path", "bls_keystore_password", "keystore"} {
			if n := migration.ResolveNode(op, []string{k}); n != nil {
				t.Errorf("expected %s to be removed, found %v", k, n)
			}
		}

		// new keystores
		ks := migration.ResolveNode(op, []string{"keystores"})
		if ks == nil || ks.Kind != 2 /* yaml.SequenceNode */ || len(ks.Content) != 2 {
			t.Fatalf("expected keystores sequence with 2 entries, got %#v", ks)
		}

		// entry 1: operatorSet 0
		ks0 := ks.Content[0]
		check := func(path []string, want string) {
			n := migration.ResolveNode(ks0, path)
			if n == nil || n.Value != want {
				t.Errorf("expected %v == %q, got %v", path, want, n)
			}
		}
		check([]string{"avs"}, "0xAVS_ADDR")
		check([]string{"operatorSet"}, "0")
		check([]string{"ecdsa_keystore_path"}, "keystores/operator1.ecdsa.keystore.json")
		check([]string{"ecdsa_keystore_password"}, "pass1")
		check([]string{"bls_keystore_path"}, "keystores/operator1.bls.keystore.json")
		check([]string{"bls_keystore_password"}, "bpass1")

		// entry 2: operatorSet 1, suffix .2
		ks1 := ks.Content[1]
		check2 := func(path []string, want string) {
			n := migration.ResolveNode(ks1, path)
			if n == nil || n.Value != want {
				t.Errorf("expected %v == %q, got %v", path, want, n)
			}
		}
		check2([]string{"avs"}, "0xAVS_ADDR")
		check2([]string{"operatorSet"}, "1")
		check2([]string{"ecdsa_keystore_path"}, "keystores/operator1.ecdsa.keystore.json")
		check2([]string{"ecdsa_keystore_password"}, "pass1")
		check2([]string{"bls_keystore_path"}, "keystores/operator1.bls.keystore.json")
		check2([]string{"bls_keystore_password"}, "bpass1")
	})

	t.Run("operator 2 keystores created", func(t *testing.T) {
		op := migration.ResolveNode(migrated, []string{"context", "operators", "1"})
		if op == nil {
			t.Fatal("operator[1] missing")
		}
		if got := migration.ResolveNode(op, []string{"address"}); got == nil || got.Value != "0xOP2" {
			t.Errorf("address not preserved, got %v", got)
		}

		ks := migration.ResolveNode(op, []string{"keystores"})
		if ks == nil || ks.Kind != 2 || len(ks.Content) != 2 {
			t.Fatalf("expected keystores sequence with 2 entries, got %#v", ks)
		}

		ks0 := migration.ResolveNode(ks, []string{"0"})
		ks1 := migration.ResolveNode(ks, []string{"1"})

		if n := migration.ResolveNode(ks0, []string{"operatorSet"}); n == nil || n.Value != "0" {
			t.Errorf("operatorSet[0] expected 0, got %v", n)
		}
		if n := migration.ResolveNode(ks1, []string{"operatorSet"}); n == nil || n.Value != "1" {
			t.Errorf("operatorSet[1] expected 1, got %v", n)
		}

		// verify suffixing on operator2
		if n := migration.ResolveNode(ks0, []string{"ecdsa_keystore_path"}); n == nil || n.Value != "keystores/operator2.ecdsa.keystore.json" {
			t.Errorf("unexpected ecdsa path[0]: %v", n)
		}
		if n := migration.ResolveNode(ks1, []string{"ecdsa_keystore_path"}); n == nil || n.Value != "keystores/operator2.ecdsa.keystore.json" {
			t.Errorf("unexpected ecdsa path[1]: %v", n)
		}
		if n := migration.ResolveNode(ks0, []string{"bls_keystore_path"}); n == nil || n.Value != "keystores/operator2.bls.keystore.json" {
			t.Errorf("unexpected bls path[0]: %v", n)
		}
		if n := migration.ResolveNode(ks1, []string{"bls_keystore_path"}); n == nil || n.Value != "keystores/operator2.bls.keystore.json" {
			t.Errorf("unexpected bls path[1]: %v", n)
		}
	})

	t.Run("allocations and context name preserved", func(t *testing.T) {
		name := migration.ResolveNode(migrated, []string{"context", "name"})
		if name == nil || name.Value != "custom-context" {
			t.Errorf("context name not preserved, got %v", name)
		}
		allocs := migration.ResolveNode(migrated, []string{"context", "allocations"})
		if allocs == nil || allocs.Kind != 2 || len(allocs.Content) != 1 {
			t.Errorf("allocations mutated, got %#v", allocs)
		}
	})
}

// TestAVSContextMigration_FullChain tests migrating through the entire chain from 0.0.1 to 0.0.8
func TestAVSContextMigration_FullChain(t *testing.T) {
	// Use the embedded v0.0.1 content as our starting point
	userYAML := string(contexts.ContextYamls["0.0.1"])

	userNode := testNode(t, userYAML)

	// Execute migration through the entire chain to 0.0.8 (latest version with ECDSA support)
	migratedNode, err := migration.MigrateNode(userNode, "0.0.1", "0.0.8", contexts.MigrationChain)
	if err != nil {
		t.Fatalf("Full chain migration failed: %v", err)
	}

	// Verify final state
	t.Run("final version is 0.0.8", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.8" {
			t.Errorf("Expected final version to be 0.0.8, got %v", version.Value)
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

		// Check that tracking sections were added and evolved (from 0.0.4→0.0.5→0.0.7)
		// deployed_contracts was added in 0.0.4→0.0.5 but removed in 0.0.6→0.0.7
		deployedContracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_contracts"})
		if deployedContracts != nil {
			t.Error("Expected deployed_contracts section to be removed in favor of L1/L2 specific sections")
		}

		// Check that L1/L2 specific tracking sections were added (from 0.0.6→0.0.7)
		deployedL1Contracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_l1_contracts"})
		if deployedL1Contracts == nil {
			t.Error("Expected deployed_l1_contracts section to be added")
		}

		deployedL2Contracts := migration.ResolveNode(migratedNode, []string{"context", "deployed_l2_contracts"})
		if deployedL2Contracts == nil {
			t.Error("Expected deployed_l2_contracts section to be added")
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

	t.Run("ECDSA keystore fields added", func(t *testing.T) {
		// Check that ECDSA keystore fields were added (from 0.0.7→0.0.8)
		firstOp := migration.ResolveNode(migratedNode, []string{"context", "operators", "0"})
		if firstOp == nil {
			t.Fatal("Expected first operator to exist")
		}

		ecdsaKeystorePath := migration.ResolveNode(firstOp, []string{"ecdsa_keystore_path"})
		if ecdsaKeystorePath == nil || ecdsaKeystorePath.Value != "keystores/operator1.ecdsa.keystore.json" {
			t.Errorf("Expected ECDSA keystore path to be added through full chain, got %v", ecdsaKeystorePath)
		}

		ecdsaKeystorePassword := migration.ResolveNode(firstOp, []string{"ecdsa_keystore_password"})
		if ecdsaKeystorePassword == nil || ecdsaKeystorePassword.Value != "testpass" {
			t.Errorf("Expected ECDSA keystore password to be added through full chain, got %v", ecdsaKeystorePassword)
		}

		// Check BLS keystore path updated to new convention
		blsKeystorePath := migration.ResolveNode(firstOp, []string{"bls_keystore_path"})
		if blsKeystorePath == nil || blsKeystorePath.Value != "keystores/operator1.bls.keystore.json" {
			t.Errorf("Expected BLS keystore path to be updated through full chain, got %v", blsKeystorePath)
		}
	})
}
