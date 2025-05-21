package contexts

import (
	_ "embed"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	contextMigrations "github.com/Layr-Labs/devkit-cli/config/contexts/migrations"
)

// Set the latest version
const LatestVersion = "0.0.4"

// Array of default contexts to create in project
var DefaultContexts = [...]string{
	"devnet",
}

// --
// Versioned contexts
// --

//go:embed v0.0.1.yaml
var v0_0_1_default []byte

//go:embed v0.0.2.yaml
var v0_0_2_default []byte

//go:embed v0.0.3.yaml
var v0_0_3_default []byte

//go:embed v0.0.4.yaml
var v0_0_4_default []byte

// Map of context name -> content
var ContextYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
	"0.0.2": v0_0_2_default,
	"0.0.3": v0_0_3_default,
	"0.0.4": v0_0_4_default,
}

// Map of sequential migrations
var MigrationChain = []migration.MigrationStep{
	{
		From:    "0.0.1",
		To:      "0.0.2",
		Apply:   contextMigrations.Migration_0_0_1_to_0_0_2,
		OldYAML: v0_0_1_default,
		NewYAML: v0_0_2_default,
	},
	{
		From:    "0.0.2",
		To:      "0.0.3",
		Apply:   contextMigrations.Migration_0_0_2_to_0_0_3,
		OldYAML: v0_0_2_default,
		NewYAML: v0_0_3_default,
	},
	{
		From:    "0.0.3",
		To:      "0.0.4",
		Apply:   contextMigrations.Migration_0_0_3_to_0_0_4,
		OldYAML: v0_0_3_default,
		NewYAML: v0_0_4_default,
	},
}
