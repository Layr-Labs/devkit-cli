package configs

import (
	_ "embed"

	configMigrations "github.com/Layr-Labs/devkit-cli/config/configs/migrations"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
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
		Apply:   configMigrations.Migration_0_0_1_to_0_0_2,
		OldYAML: v0_0_1_default,
		NewYAML: v0_0_2_default,
	},
}
