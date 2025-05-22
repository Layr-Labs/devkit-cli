package contextMigrations

import (
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_3_to_0_0_4(user, old, new *yaml.Node) (*yaml.Node, error) {
	// Extract eigenlayer section from new default
	eigenlayerNode := migration.ResolveNode(new, []string{"context", "eigenlayer"})

	// Check if context exists in user config, create if not
	contextNode := migration.ResolveNode(user, []string{"context"})
	if contextNode == nil || contextNode.Kind != yaml.MappingNode {
		// Something is wrong with user config, just return it unmodified
		return user, nil
	}

	// Add eigenlayer section to user config
	if eigenlayerNode != nil {
		// Create eigenlayer key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: "eigenlayer",
		}

		// Create a copy of the eigenlayer value node from the new config
		valueNode := migration.CloneNode(eigenlayerNode)

		// Append the key-value pair to the context mapping
		contextNode.Content = append(contextNode.Content, keyNode, valueNode)
	}

	// Copy Zeus config for projects created before Zeus integration
	zeusConfigSrc := filepath.Join("config", ".zeus")
	zeusConfigDst := ".zeus"

	// Check if source Zeus config exists
	if _, err := os.Stat(zeusConfigSrc); err == nil {
		// Read Zeus config content
		content, err := os.ReadFile(zeusConfigSrc)
		if err == nil {
			// Write Zeus config to project root if it doesn't exist already
			if _, err := os.Stat(zeusConfigDst); os.IsNotExist(err) {
				_ = os.WriteFile(zeusConfigDst, content, 0644)
			}
		}
	}

	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.4"
	}
	return user, nil
}
