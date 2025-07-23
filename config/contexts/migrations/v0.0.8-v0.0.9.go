package contextMigrations

import (
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
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "8825097"}
				},
			},
			// Update L2 fork block
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "28753690"}
				},
			},
			// Update L1 CrossChainRegistry address
			{
				Path:      []string{"context", "eigenlayer", "l1", "cross_chain_registry"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x287381B1570d9048c4B4C7EC94d21dDb8Aa1352a"}
				},
			},
			// Update L1 OperatorTableUpdater address
			{
				Path:      []string{"context", "eigenlayer", "l1", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xB02A15c6Bd0882b35e9936A9579f35FB26E11476"}
				},
			},
			// Update L1 KeyRegistrar address
			{
				Path:      []string{"context", "eigenlayer", "l1", "key_registrar"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xA4dB30D08d8bbcA00D40600bee9F029984dB162a"}
				},
			},
			// Update L2 OperatorTableUpdater address
			{
				Path:      []string{"context", "eigenlayer", "l2", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xB02A15c6Bd0882b35e9936A9579f35FB26E11476"}
				},
			},
			// Update L2 BN254CertificateVerifier address
			{
				Path:      []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xff58A373c18268F483C1F5cA03Cf885c0C43373a"}
				},
			},
			// Update L2 ECDSACertificateVerifier address
			{
				Path:      []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xb3Cd1A457dEa9A9A6F6406c6419B1c326670A96F"}
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
