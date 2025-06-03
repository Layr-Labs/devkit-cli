package migration_test

import (
	"testing"

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

// TestAVSContextMigration_0_0_1_to_0_0_2 tests the specific migration from version 0.0.1 to 0.0.2
// using the actual migration files from config/contexts/
func TestAVSContextMigration_0_0_1_to_0_0_2(t *testing.T) {
	// This represents a user's devnet.yaml file at version 0.0.1
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.1
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
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
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  # List of Operators and their private keys / stake details
  operators:
    - address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
      ecdsa_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
      ecdsa_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
      ecdsa_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
    avs_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"`

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
	// User's devnet.yaml file at version 0.0.2
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.2
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
  chains:
    l1: 
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
    l2:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
  # List of Operators and their private keys / stake details
  operators:
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
      ecdsa_key: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba" # Anvil Private Key 5
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x976EA74026E726554dB657fA54763abd0C3a0aa9"
      ecdsa_key: "0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e" # Anvil Private Key 6
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"
      ecdsa_key: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356" # Anvil Private Key 7
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    avs_private_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"
`

	userNode := testNode(t, userYAML)

	// Get the actual migration step
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
	// User's devnet.yaml file at version 0.0.3 (without eigenlayer section)
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.3
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
  chains:
    l1:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
    l2:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
  # List of Operators and their private keys / stake details
  operators:
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
      ecdsa_key: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba" # Anvil Private Key 5
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x976EA74026E726554dB657fA54763abd0C3a0aa9"
      ecdsa_key: "0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e" # Anvil Private Key 6
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"
      ecdsa_key: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356" # Anvil Private Key 7
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    avs_private_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"
`

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
	// User's devnet.yaml file at version 0.0.4
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.4
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
  chains:
    l1: 
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
    l2:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
  # List of Operators and their private keys / stake details
  operators:
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
      ecdsa_key: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba" # Anvil Private Key 5
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x976EA74026E726554dB657fA54763abd0C3a0aa9"
      ecdsa_key: "0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e" # Anvil Private Key 6
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"
      ecdsa_key: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356" # Anvil Private Key 7
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    avs_private_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"
  # Core EigenLayer contract addresses
  eigenlayer:
    allocation_manager: "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39"
    delegation_manager: "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A" `

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
// which updates keystore files
func TestAVSContextMigration_0_0_5_to_0_0_6(t *testing.T) {
	// User's devnet.yaml file at version 0.0.5
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.5
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
  chains:
    l1:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
    l2:
      chain_id: 31337
      rpc_url: "http://localhost:8545"
      fork:
        block: 22475020
        url: ""
        block_time: 3
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
  # List of Operators and their private keys / stake details
  operators:
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
      ecdsa_key: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba" # Anvil Private Key 5
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x976EA74026E726554dB657fA54763abd0C3a0aa9"
      ecdsa_key: "0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e" # Anvil Private Key 6
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"
      ecdsa_key: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356" # Anvil Private Key 7
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    avs_private_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"
  # Core EigenLayer contract addresses
  eigenlayer:
    allocation_manager: "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39"
    delegation_manager: "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A" 
  # Contracts deployed on devnet start
  deployed_contracts: []
  # Operator Sets registered on devnet start
  operator_sets: []
  # Operators registered on devnet start
  operator_registrations: []`

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

	t.Run("configuration preserved", func(t *testing.T) {
		// Ensure existing configs aren't affected by the keystore file updates
		opAddr := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "address"})
		if opAddr == nil || opAddr.Value != "0x90F79bf6EB2c4f870365E785982E1f101E93b906" {
			t.Errorf("Expected operator address to be preserved, got %v", opAddr.Value)
		}

		keystorePath := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "bls_keystore_path"})
		if keystorePath == nil || keystorePath.Value != "keystores/operator1.keystore.json" {
			t.Errorf("Expected keystore path to be preserved, got %v", keystorePath.Value)
		}
	})
}

// TestAVSContextMigration_FullChain tests migrating through the entire chain from 0.0.1 to 0.0.5
func TestAVSContextMigration_FullChain(t *testing.T) {
	// User starts with a 0.0.1 configuration
	userYAML := `# Devnet context to be used for local deployments against Anvil chain
version: 0.0.1
context:
  # Name of the context
  name: "devnet"
  # Chains available to this context
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
  # All key material (BLS and ECDSA) within this file should be used for local testing ONLY
  # ECDSA keys used are from Anvil's private key set
  # BLS keystores are deterministically pre-generated and embedded. These are NOT derived from a secure seed
  # Available private keys for deploying
  deployer_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  app_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
  # List of Operators and their private keys / stake details
  operators:
    - address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
      ecdsa_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
      bls_keystore_path: "keystores/operator1.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
      ecdsa_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" # Anvil Private Key 1
      bls_keystore_path: "keystores/operator2.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
      ecdsa_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" # Anvil Private Key 2
      bls_keystore_path: "keystores/operator3.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" # Anvil Private Key 3
      bls_keystore_path: "keystores/operator4.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a" # Anvil Private Key 4
      bls_keystore_path: "keystores/operator5.keystore.json"
      bls_keystore_password: "testpass"
      stake: "1000ETH"
  # AVS configuration
  avs:
    address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
    avs_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" # Anvil Private Key 0
    metadata_url: "https://my-org.com/avs/metadata.json"
    registrar_address: "0x0123456789abcdef0123456789ABCDEF01234567"`

	userNode := testNode(t, userYAML)

	// Execute migration through the entire chain
	migratedNode, err := migration.MigrateNode(userNode, "0.0.1", "0.0.5", contexts.MigrationChain)
	if err != nil {
		t.Fatalf("Full chain migration failed: %v", err)
	}

	// Verify final state
	t.Run("final version is 0.0.5", func(t *testing.T) {
		version := migration.ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.5" {
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
	})

	t.Run("user customizations preserved", func(t *testing.T) {
		// User's custom stake should be preserved
		stake := migration.ResolveNode(migratedNode, []string{"context", "operators", "0", "stake"})
		if stake == nil || stake.Value != "1000ETH" {
			t.Errorf("Expected user's stake to be preserved, got %v", stake.Value)
		}
	})
}
