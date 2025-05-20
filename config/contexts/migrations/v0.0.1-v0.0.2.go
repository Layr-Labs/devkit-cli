package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_1_to_0_0_2(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			{Path: []string{"context", "chains", "l1", "fork", "url"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "chains", "l2", "fork", "url"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "app_private_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "0", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "0", "ecdsa_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "1", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "1", "ecdsa_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "2", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "2", "ecdsa_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "3", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "3", "ecdsa_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "4", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "operators", "4", "ecdsa_key"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "avs", "address"}, Condition: migration.IfUnchanged{}},
			{Path: []string{"context", "avs", "avs_private_key"}, Condition: migration.IfUnchanged{}},
		},
	}
	err := engine.Apply()
	if err != nil {
		return nil, err
	}

	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.2"
	}
	return user, nil
}
