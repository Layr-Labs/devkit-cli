package migration

import (
	"errors"
	"testing"

	contextMigrations "github.com/Layr-Labs/devkit-cli/config/contexts/migrations"
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

func TestResolveNode(t *testing.T) {
	src := `
version: v1
nested:
  key: value
list:
  - a
  - b
`
	node := testNode(t, src)

	// scalar
	vn := ResolveNode(node, []string{"version"})
	if vn == nil || vn.Value != "v1" {
		t.Error("ResolveNode version failed")
	}

	// nested
	kn := ResolveNode(node, []string{"nested", "key"})
	if kn == nil || kn.Value != "value" {
		t.Error("ResolveNode nested.key failed")
	}

	// list
	ln := ResolveNode(node, []string{"list", "1"})
	if ln == nil || ln.Value != "b" {
		t.Error("ResolveNode list[1] failed")
	}
}

func TestCloneNode(t *testing.T) {
	src := `key: orig`
	n := testNode(t, src)
	clone := CloneNode(n)

	// modify clone
	clone.Content[1].Value = "new"

	orig := testNode(t, src)
	ov := ResolveNode(orig, []string{"key"})
	if ov == nil || ov.Value != "orig" {
		t.Error("CloneNode did not deep copy")
	}
}

func TestPatchEngine_Apply(t *testing.T) {
	yamlOld := `
version: v1
param: old
`
	yamlNew := `
version: v1
param: new
`
	yamlUser := `
version: v1
param: old
`

	oldDef := testNode(t, yamlOld)
	newDef := testNode(t, yamlNew)
	user := testNode(t, yamlUser)

	engine := PatchEngine{
		Old:  oldDef,
		New:  newDef,
		User: user,
		Rules: []PatchRule{{
			Path:      []string{"param"},
			Condition: IfUnchanged{},
		}},
	}
	if err := engine.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	on := ResolveNode(user, []string{"param"})
	if on == nil || on.Value != "new" {
		t.Errorf("Expected param=new, got %v", on.Value)
	}
}

func TestMigrateNode_AlreadyUpToDate(t *testing.T) {
	yamlUser := `version: v1`
	node := testNode(t, yamlUser)
	// empty chain
	_, err := MigrateNode(node, "v1", "v1", nil)
	if !errors.Is(err, ErrAlreadyUpToDate) {
		t.Error("Expected ErrAlreadyUpToDate")
	}
}

// TestAVSContextMigration_0_0_1_to_0_0_2 tests the specific migration from version 0.0.1 to 0.0.2
// using the actual migration function from config/contexts/migrations
func TestAVSContextMigration_0_0_1_to_0_0_2(t *testing.T) {
	// This represents a user's devnet.yaml file at version 0.0.1 (simplified for testing)
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

	// This represents the old default template (v0.0.1) - simplified for migration testing
	oldDefaultYAML := `version: 0.0.1
context:
  chains:
    l1:
      fork:
        url: "https://eth.llamarpc.com"
    l2:
      fork:
        url: "https://eth.llamarpc.com"
  app_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
  operators:
    - address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
      ecdsa_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    - address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
      ecdsa_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
    - address: "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
      ecdsa_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
  avs:
    address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
    avs_private_key: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"`

	// This represents the new default template (v0.0.2) - simplified for migration testing
	newDefaultYAML := `version: 0.0.2
context:
  chains:
    l1:
      fork:
        url: ""
    l2:
      fork:
        url: ""
  app_private_key: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
  operators:
    - address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906"
      ecdsa_key: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
    - address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
      ecdsa_key: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
    - address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"
      ecdsa_key: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
    - address: "0x976EA74026E726554dB657fA54763abd0C3a0aa9"
      ecdsa_key: "0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e"
    - address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"
      ecdsa_key: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356"
  avs:
    address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
    avs_private_key: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"`

	// Parse YAML nodes
	userNode := testNode(t, userYAML)
	// oldNode := testNode(t, oldDefaultYAML)
	// newNode := testNode(t, newDefaultYAML)

	// Create migration step using the ACTUAL migration function from contexts/migrations
	migrationStep := MigrationStep{
		From:    "0.0.1",
		To:      "0.0.2",
		Apply:   contextMigrations.Migration_0_0_1_to_0_0_2, // This is the actual migration function!
		OldYAML: []byte(oldDefaultYAML),
		NewYAML: []byte(newDefaultYAML),
	}

	// Execute migration using the actual migration function
	migrationChain := []MigrationStep{migrationStep}
	migratedNode, err := MigrateNode(userNode, "0.0.1", "0.0.2", migrationChain)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Verify the migration results
	t.Run("version updated", func(t *testing.T) {
		version := ResolveNode(migratedNode, []string{"version"})
		if version == nil || version.Value != "0.0.2" {
			t.Errorf("Expected version to be updated to 0.0.2, got %v", version.Value)
		}
	})

	t.Run("L1 fork URL updated", func(t *testing.T) {
		l1ForkUrl := ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "url"})
		if l1ForkUrl == nil || l1ForkUrl.Value != "" {
			t.Errorf("Expected L1 fork URL to be empty, got %v", l1ForkUrl.Value)
		}
	})

	t.Run("L2 fork URL updated", func(t *testing.T) {
		l2ForkUrl := ResolveNode(migratedNode, []string{"context", "chains", "l2", "fork", "url"})
		if l2ForkUrl == nil || l2ForkUrl.Value != "" {
			t.Errorf("Expected L2 fork URL to be empty, got %v", l2ForkUrl.Value)
		}
	})

	t.Run("app_private_key updated", func(t *testing.T) {
		appKey := ResolveNode(migratedNode, []string{"context", "app_private_key"})
		if appKey == nil || appKey.Value != "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" {
			t.Errorf("Expected app_private_key to be updated to new value, got %v", appKey.Value)
		}
	})

	t.Run("operator details preserved", func(t *testing.T) {
		// Operator address should remain unchanged
		opAddress := ResolveNode(migratedNode, []string{"context", "operators", "0", "address"})
		if opAddress == nil || opAddress.Value != "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" {
			t.Errorf("Expected operator address to be preserved, got %v", opAddress.Value)
		}

		// Operator ECDSA key should remain unchanged
		opKey := ResolveNode(migratedNode, []string{"context", "operators", "0", "ecdsa_key"})
		if opKey == nil || opKey.Value != "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" {
			t.Errorf("Expected operator ECDSA key to be preserved, got %v", opKey.Value)
		}

		// Additional operator fields should be preserved (these aren't in defaults but exist in user config)
		opStake := ResolveNode(migratedNode, []string{"context", "operators", "0", "stake"})
		if opStake == nil || opStake.Value != "1000ETH" {
			t.Errorf("Expected operator stake to be preserved, got %v", opStake.Value)
		}
	})

	t.Run("AVS details preserved", func(t *testing.T) {
		// AVS address should remain unchanged
		avsAddress := ResolveNode(migratedNode, []string{"context", "avs", "address"})
		if avsAddress == nil || avsAddress.Value != "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC" {
			t.Errorf("Expected AVS address to be preserved, got %v", avsAddress.Value)
		}

		// AVS private key should remain unchanged
		avsKey := ResolveNode(migratedNode, []string{"context", "avs", "avs_private_key"})
		if avsKey == nil || avsKey.Value != "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" {
			t.Errorf("Expected AVS private key to be preserved, got %v", avsKey.Value)
		}

		// Additional AVS fields should be preserved (these aren't in defaults but exist in user config)
		avsMetadata := ResolveNode(migratedNode, []string{"context", "avs", "metadata_url"})
		if avsMetadata == nil || avsMetadata.Value != "https://my-org.com/avs/metadata.json" {
			t.Errorf("Expected AVS metadata URL to be preserved, got %v", avsMetadata.Value)
		}
	})

	t.Run("chain configuration preserved", func(t *testing.T) {
		// Chain IDs should be preserved
		l1ChainId := ResolveNode(migratedNode, []string{"context", "chains", "l1", "chain_id"})
		if l1ChainId == nil || l1ChainId.Value != "31337" {
			t.Errorf("Expected L1 chain ID to be preserved, got %v", l1ChainId.Value)
		}

		// RPC URLs should be preserved
		l1RpcUrl := ResolveNode(migratedNode, []string{"context", "chains", "l1", "rpc_url"})
		if l1RpcUrl == nil || l1RpcUrl.Value != "http://localhost:8545" {
			t.Errorf("Expected L1 RPC URL to be preserved, got %v", l1RpcUrl.Value)
		}

		// Fork block should be preserved
		l1ForkBlock := ResolveNode(migratedNode, []string{"context", "chains", "l1", "fork", "block"})
		if l1ForkBlock == nil || l1ForkBlock.Value != "22475020" {
			t.Errorf("Expected L1 fork block to be preserved, got %v", l1ForkBlock.Value)
		}
	})
}
