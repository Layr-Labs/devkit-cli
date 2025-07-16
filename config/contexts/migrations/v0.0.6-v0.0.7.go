package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_6_to_0_0_7(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Update fork block for L1 chain
			{
				Path:      []string{"context", "chains", "l1", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "8713384"}
				},
			},
			// Update fork block for L2 chain
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "28069764"}
				},
			},
			// Update rpc url for l2 chain
			{
				Path:      []string{"context", "chains", "l2", "rpc_url"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "http://localhost:9545"}
				},
			},
			// Update chain id for l2 chain
			{
				Path:      []string{"context", "chains", "l2", "chain_id"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "31338"}
				},
			},
			// Update the transporter private_key
			{
				Path:      []string{"context", "transporter", "private_key"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x5f8e6420b9cb0c940e3d3f8b99177980785906d16fb3571f70d7a05ecf5f2172"}
				},
			},
			// Update the transporter bls_private_key
			{
				Path:      []string{"context", "transporter", "bls_private_key"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x5f8e6420b9cb0c940e3d3f8b99177980785906d16fb3571f70d7a05ecf5f2172"}
				},
			},
			// Update allocation_manager for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "allocation_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x42583067658071247ec8CE0A516A58f682002d07"}
				},
			},
			// Update delegation_manager for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "delegation_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xD4A7E1Bd8015057293f0D0A557088c286942e84b"}
				},
			},
			// Update strategy_manager for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "strategy_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x2E3D6c0744b10eb0A4e6F679F71554a39Ec47a5D"}
				},
			},
			// Update bn254_table_calculator for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "bn254_table_calculator"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xc2c0bc13571aC5115709C332dc7AE666606b08E8"}
				},
			},
			// Update cross_chain_manager for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "cross_chain_registry"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xe850D8A178777b483D37fD492a476e3E6004C816"}
				},
			},
			// Update key_registrar for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "key_registrar"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x78De554Ac8DfF368e3CAa73B3Df8AccCfD92928A"}
				},
			},
			// Update release_manager for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "release_manager"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xd9Cb89F1993292dEC2F973934bC63B0f2A702776"}
				},
			},
			// Update operator_table_updater for l1 chain
			{
				Path:      []string{"context", "eigenlayer", "l1", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xE12C4cebd680a917271145eDbFB091B1BdEFD74D"}
				},
			},
			// Update bn254_certificate_verifier for l2 chain
			{
				Path:      []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x998535833f3feE44ce720440E735554699f728a5"}
				},
			},
			// Update operator_table_updater for l2 chain
			{
				Path:      []string{"context", "eigenlayer", "l2", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xE12C4cebd680a917271145eDbFB091B1BdEFD74D"}
				},
			},
			// Add ecdsa_certificate_verifier for l2 chain
			{
				Path:      []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0xAD2F58A551bD0e77fa20b5531dA96eF440C392BF"}
				},
			},
			// Add deployed_l1_contracts section
			{
				Path:      []string{"context", "deployed_l1_contracts"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
				},
			},
			// Add deployed_l2_contracts section
			{
				Path:      []string{"context", "deployed_l2_contracts"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.SequenceNode, Content: []*yaml.Node{}}
				},
			},
			// Remove deployed_contracts section
			{
				Path:      []string{"context", "deployed_contracts"},
				Condition: migration.Always{},
				Remove:    true,
			},
		},
	}
	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Insert stakers section after app_private_key and before operators
	contextNode := migration.ResolveNode(user, []string{"context"})

	// Update or create artifact section (renamed from artifacts to artifact)
	if contextNode != nil && contextNode.Kind == yaml.MappingNode {
		// Find existing artifacts section
		artifactsIndex := -1
		artifactsKeyIndex := -1

		for i := 0; i < len(contextNode.Content)-1; i += 2 {
			if contextNode.Content[i].Value == "artifacts" {
				artifactsIndex = i + 1
				artifactsKeyIndex = i
				break
			}
		}

		// Create the proper artifact structure with artifactId field
		newArtifactValue := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "artifactId", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "component", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "digest", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "registry", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "version", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
			},
		}

		if artifactsIndex != -1 {
			// Update the key name from "artifacts" to "artifact" and update the value
			contextNode.Content[artifactsKeyIndex].Value = "artifact"
			contextNode.Content[artifactsKeyIndex].HeadComment = "# Release artifact"
			contextNode.Content[artifactsIndex] = newArtifactValue
		} else {
			// Add new artifact section if it doesn't exist
			artifactKey := &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "artifact",
				HeadComment: "# Release artifact",
			}
			contextNode.Content = append(contextNode.Content, artifactKey, newArtifactValue)
		}
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.7"
	}
	return user, nil
}
