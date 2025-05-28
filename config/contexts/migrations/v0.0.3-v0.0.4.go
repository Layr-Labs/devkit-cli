package contextMigrations

import (
	"os"

	"github.com/Layr-Labs/devkit-cli/config"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_3_to_0_0_4(user, old, new *yaml.Node) (*yaml.Node, error) {
	log, _ := common.GetLogger(true) // We don't have context for logger here. So using verbose logs as default for migrations.
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
		// Add the key with comment first
		migration.EnsureKeyWithComment(user, []string{"context", "eigenlayer"}, "Core EigenLayer contract addresses")

		// Pull users eigenlayer key node
		keyNode := migration.ResolveNode(user, []string{"context", "eigenlayer"})

		// Replace the key-value pairs in the context eigenlayer mapping
		*keyNode = *migration.CloneNode(eigenlayerNode)
	}

	// Write Zeus config to project root if it doesn't exist already
	zeusConfigDst := common.ZeusConfig
	if _, err := os.Stat(zeusConfigDst); os.IsNotExist(err) {
		_ = os.WriteFile(zeusConfigDst, []byte(config.ZeusConfig), 0644)
	}

	log.Info("Copied .zeus config to project root")
	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.4"
	}
	return user, nil
}
