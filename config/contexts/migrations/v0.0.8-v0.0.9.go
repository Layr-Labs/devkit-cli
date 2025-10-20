package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_8_to_0_0_9(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Update L1 fork block
			{
				Path:      []string{"context", "chains", "l1", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "8836193"}
				},
			},
			// Update L2 fork block
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "28820370"}
				},
			},
			// Update L1 AllocationManager address
			{
				Path:      []string{"context", "eigenlayer", "l1", "allocation_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_ALLOCATION_MANAGER_ADDRESS, common.SEPOLIA_ALLOCATION_MANAGER_ADDRESS)
				},
			},
			// Update L1 DelegationManager address
			{
				Path:      []string{"context", "eigenlayer", "l1", "delegation_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_DELEGATION_MANAGER_ADDRESS, common.SEPOLIA_DELEGATION_MANAGER_ADDRESS)
				},
			},
			// Update L1 StrategyManager address
			{
				Path:      []string{"context", "eigenlayer", "l1", "strategy_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_STRATEGY_MANAGER_ADDRESS, common.SEPOLIA_STRATEGY_MANAGER_ADDRESS)
				},
			},
			// Update L1 ReleaseManager address
			{
				Path:      []string{"context", "eigenlayer", "l1", "release_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_RELEASE_MANAGER_ADDRESS, common.SEPOLIA_RELEASE_MANAGER_ADDRESS)
				},
			},
			// Update L1 CrossChainRegistry address
			{
				Path:      []string{"context", "eigenlayer", "l1", "cross_chain_registry"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_CROSS_CHAIN_REGISTRY_ADDRESS, common.SEPOLIA_CROSS_CHAIN_REGISTRY_ADDRESS)
				},
			},
			// Update L1 OperatorTableUpdater address
			{
				Path:      []string{"context", "eigenlayer", "l1", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_L1_OPERATOR_TABLE_UPDATER_ADDRESS, common.SEPOLIA_L1_OPERATOR_TABLE_UPDATER_ADDRESS)
				},
			},
			// Update L1 KeyRegistrar address
			{
				Path:      []string{"context", "eigenlayer", "l1", "key_registrar"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_KEY_REGISTRAR_ADDRESS, common.SEPOLIA_KEY_REGISTRAR_ADDRESS)
				},
			},
			// Update L1 TaskMailbox address
			{
				Path:      []string{"context", "eigenlayer", "l1", "task_mailbox"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_L1_TASK_MAILBOX_ADDRESS, common.SEPOLIA_L1_TASK_MAILBOX_ADDRESS)
				},
			},
			// Update L2 TaskMailbox address
			{
				Path:      []string{"context", "eigenlayer", "l2", "task_mailbox"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_L2_TASK_MAILBOX_ADDRESS, common.SEPOLIA_L2_TASK_MAILBOX_ADDRESS)
				},
			},
			// Update L2 OperatorTableUpdater address
			{
				Path:      []string{"context", "eigenlayer", "l2", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_L2_OPERATOR_TABLE_UPDATER_ADDRESS, common.SEPOLIA_L2_OPERATOR_TABLE_UPDATER_ADDRESS)
				},
			},
			// Update L2 BN254CertificateVerifier address
			{
				Path:      []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_BN254_CERTIFICATE_VERIFIER_ADDRESS, common.SEPOLIA_BN254_CERTIFICATE_VERIFIER_ADDRESS)
				},
			},
			// Update L2 ECDSACertificateVerifier address
			{
				Path:      []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_ECDSA_CERTIFICATE_VERIFIER_ADDRESS, common.SEPOLIA_ECDSA_CERTIFICATE_VERIFIER_ADDRESS)
				},
			},
			// Update L1 BN254TableCalculator address (env aware - this will not be updated by zeus)
			{
				Path:      []string{"context", "eigenlayer", "l1", "bn254_table_calculator"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_BN254_TABLE_CALCULATOR_ADDRESS, common.SEPOLIA_BN254_TABLE_CALCULATOR_ADDRESS)
				},
			},
			// Update L1 ECDSATableCalculator middleware address (env aware - this will not be updated by zeus)
			{
				Path:      []string{"context", "eigenlayer", "l1", "ecdsa_table_calculator"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return GetAddressByChainIdFromCtx(user, common.MAINNET_ECDSA_TABLE_CALCULATOR_ADDRESS, common.SEPOLIA_ECDSA_TABLE_CALCULATOR_ADDRESS)
				},
			},
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.9"
	}

	return user, nil
}

func GetAddressByChainIdFromCtx(ctx *yaml.Node, mainnetAddress, sepoliaAddress string) *yaml.Node {
	// check l1 chainId - if == 1 then use MAINNET address
	chainId := migration.ResolveNode(ctx, []string{"context", "chains", "l1", "chain_id"})
	address := sepoliaAddress
	if chainId != nil && chainId.Value == "1" {
		address = mainnetAddress
	}

	return &yaml.Node{Kind: yaml.ScalarNode, Value: address}
}
