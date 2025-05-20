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
			{Path: []string{"context", "chains", "l1", "fork", "block_time"}, Condition: migration.Always{}},
			{Path: []string{"context", "chains", "l2", "fork", "block_time"}, Condition: migration.Always{}},
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
