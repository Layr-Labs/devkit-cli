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
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "4056218"}
				},
			},
			// Update fork block for L2 chain
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "27764494"}
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
			// Update bn254_certificate_verifier for l2 chain
			{
				Path:      []string{"context", "eigenlayer", "l2", "bn254_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x824604a31b580Aec16D8Dd7ae9A27661Dc65cBA3"}
				},
			},
			// Update operator_table_updater for l2 chain
			{
				Path:      []string{"context", "eigenlayer", "l2", "operator_table_updater"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x798EB817B7C109c6780264D5161183809C817216"}
				},
			},
			// Add ecdsa_certificate_verifier for l2 chain 
			{
				Path:      []string{"context", "eigenlayer", "l2", "ecdsa_certificate_verifier"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x95A49cB0aED0e8f299223Da3A8A335440f5F00E7"}
				},
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
