package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_2_to_0_0_3(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			{
				Path:      []string{"context", "chains", "l1", "fork"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					// build the key node and scalar node
					key := &yaml.Node{Kind: yaml.ScalarNode, Value: "block_time"}
					val := &yaml.Node{Kind: yaml.ScalarNode, Value: migration.ResolveNode(new, []string{"context", "chains", "l1", "fork", "block_time"}).Value}
					// clone the existing fork mapping, then append our new pair
					forkMap := migration.CloneNode(migration.ResolveNode(user, []string{"context", "chains", "l1", "fork"}))
					forkMap.Content = append(forkMap.Content, key, val)
					return forkMap
				},
			},
			{
				Path:      []string{"context", "chains", "l2", "fork"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					// build the key node and scalar node
					key := &yaml.Node{Kind: yaml.ScalarNode, Value: "block_time"}
					val := &yaml.Node{Kind: yaml.ScalarNode, Value: migration.ResolveNode(new, []string{"context", "chains", "l2", "fork", "block_time"}).Value}
					// clone the existing fork mapping, then append our new pair
					forkMap := migration.CloneNode(migration.ResolveNode(user, []string{"context", "chains", "l2", "fork"}))
					forkMap.Content = append(forkMap.Content, key, val)
					return forkMap
				},
			},
		},
	}
	err := engine.Apply()
	if err != nil {
		return nil, err
	}

	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.3"
	}
	return user, nil
}
