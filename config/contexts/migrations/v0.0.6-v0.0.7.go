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
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "4056218"}
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
