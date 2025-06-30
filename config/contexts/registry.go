package contexts

import (
	_ "embed"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	contextMigrations "github.com/Layr-Labs/devkit-cli/config/contexts/migrations"
)

// Set the latest version
const LatestVersion = "0.0.7"

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

//go:embed v0.0.5.yaml
var v0_0_5_default []byte

//go:embed v0.0.6.yaml
var v0_0_6_default []byte

//go:embed v0.0.7.yaml
var v0_0_7_default []byte

// Map of context name -> content
var ContextYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
	"0.0.2": v0_0_2_default,
	"0.0.3": v0_0_3_default,
	"0.0.4": v0_0_4_default,
	"0.0.5": v0_0_5_default,
	"0.0.6": v0_0_6_default,
	"0.0.7": v0_0_7_default,
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
	{
		From:    "0.0.4",
		To:      "0.0.5",
		Apply:   contextMigrations.Migration_0_0_4_to_0_0_5,
		OldYAML: v0_0_4_default,
		NewYAML: v0_0_5_default,
	},
	{
		From:    "0.0.5",
		To:      "0.0.6",
		Apply:   contextMigrations.Migration_0_0_5_to_0_0_6,
		OldYAML: v0_0_5_default,
		NewYAML: v0_0_6_default,
	},
	{
		From:    "0.0.6",
		To:      "0.0.7",
		Apply:   contextMigrations.Migration_0_0_6_to_0_0_7,
		OldYAML: v0_0_6_default,
		NewYAML: v0_0_7_default,
	},
}
