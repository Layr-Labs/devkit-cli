package contextMigrations

import (
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	"gopkg.in/yaml.v3"
)

func Migration_0_1_1_to_0_1_2(user, old, new *yaml.Node) (*yaml.Node, error) {
	// Update fork block heights to match ponos project
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Remove transporter keys
			{
				Path:      []string{"context", "transporter", "private_key"},
				Condition: migration.Always{},
				Remove:    true,
			},
			{
				Path:      []string{"context", "transporter", "bls_private_key"},
				Condition: migration.Always{},
				Remove:    true,
			},
		},
	}
	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Get the contexts name
	contextName := migration.ResolveNode(user, []string{"context", "name"})

	// If contextName contains devnet, insert mnemonic
	if strings.Contains(contextName.Value, devnet.DEVNET_CONTEXT) {
		// Insert mnemonic into yaml after chains
		_ = migration.InsertAfterKeyWithComment(
			user,
			[]string{"context"},
			"chains",
			"mnemonic",
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Style: yaml.DoubleQuotedStyle,
				Value: devnet.DEFAULT_MNEMONIC,
			},
			"Devnet mnemonic for unlocked accounts",
			false,
		)
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.1.2"
	}

	return user, nil
}
