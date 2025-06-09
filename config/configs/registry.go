package configs

import (
	_ "embed"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	"gopkg.in/yaml.v3"
)

// Set the latest version
const LatestVersion = "0.0.2"

// --
// Versioned configs
// --

//go:embed v0.0.1.yaml
var v0_0_1_default []byte

//go:embed v0.0.2.yaml
var v0_0_2_default []byte

// Map of context name -> content
var ConfigYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
	"0.0.2": v0_0_2_default,
}

// Map of sequential migrations
var MigrationChain = []migration.MigrationStep{
	{
		From:    "0.0.1",
		To:      "0.0.2",
		Apply:   migrateConfigV0_0_1ToV0_0_2,
		OldYAML: v0_0_1_default,
		NewYAML: v0_0_2_default,
	},
}

// migrateConfigV0_0_1ToV0_0_2 adds project_uuid and telemetry_enabled fields
func migrateConfigV0_0_1ToV0_0_2(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Add project_uuid field (empty string by default)
			{
				Path:      []string{"config", "project", "project_uuid"},
				Condition: migration.Always{},
			},
			// Add telemetry_enabled field (false by default)
			{
				Path:      []string{"config", "project", "telemetry_enabled"},
				Condition: migration.Always{},
			},
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.2"
	}

	return user, nil
}
