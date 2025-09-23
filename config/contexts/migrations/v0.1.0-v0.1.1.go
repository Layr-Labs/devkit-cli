package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	"gopkg.in/yaml.v3"
)

func Migration_0_1_0_to_0_1_1(user, old, new *yaml.Node) (*yaml.Node, error) {
	// Update fork block heights to match ponos project
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
					return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "9259079"}
				},
			},
			// Update L2 fork block
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: "31408197"}
				},
			},
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.1.1"
	}

	return user, nil
}
