package contextMigrations

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_6_to_0_0_7(user, old, new *yaml.Node) (*yaml.Node, error) {
	ctxNode := migration.ResolveNode(user, []string{"context"})
	if ctxNode == nil || ctxNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("context node not found or not a mapping")
	}

	// Check if transporter already exists
	hasTransporter := false
	for i := 0; i < len(ctxNode.Content)-1; i += 2 {
		if ctxNode.Content[i].Value == "transporter" {
			hasTransporter = true
			break
		}
	}

	if !hasTransporter {
		// Create key + value nodes
		transporterKey := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Tag:         "!!str",
			Value:       "transporter",
			HeadComment: "Stake Root Transporter configuration",
		}
		transporterValue := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "schedule"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0 */2 * * *"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "private_key"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0x2ba58f64c57faa1073d63add89799f2a0101855a8b289b1330cb500758d5d1ee"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bls_private_key"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0x2ba58f64c57faa1073d63add89799f2a0101855a8b289b1330cb500758d5d1ee"},
			},
		}

		// Insert after "chains"
		inserted := false
		for i := 0; i < len(ctxNode.Content)-1; i += 2 {
			if ctxNode.Content[i].Value == "chains" {
				before := ctxNode.Content[:i+2]
				after := ctxNode.Content[i+2:]

				// Insert the transporter between before and after
				newContent := make([]*yaml.Node, 0, len(ctxNode.Content)+2)
				newContent = append(newContent, before...)
				newContent = append(newContent, transporterKey, transporterValue)
				newContent = append(newContent, after...)

				// Set back the content
				ctxNode.Content = newContent
				inserted = true
				break
			}
		}
		if !inserted {
			ctxNode.Content = append(ctxNode.Content, transporterKey, transporterValue)
		}
	}

	// Update version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.7"
	}

	return user, nil
}
