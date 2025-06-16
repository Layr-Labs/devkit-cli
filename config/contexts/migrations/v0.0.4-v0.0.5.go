package contextMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_4_to_0_0_5(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{}
	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Append comments+keys at bottom if missing
	migration.EnsureKeyWithComment(user, []string{"context", "deployed_contracts"}, "Contracts deployed on `devnet start`")
	migration.EnsureKeyWithComment(user, []string{"context", "operator_sets"}, "Operator Sets registered on `devnet start`")
	migration.EnsureKeyWithComment(user, []string{"context", "operator_registrations"}, "Operators registered on `devnet start`")

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.5"
	}
	return user, nil
}
