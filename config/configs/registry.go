package configs

import (
	_ "embed"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"
)

// Set the latest version
const LatestVersion = "0.0.1"

// --
// Versioned configs
// --

//go:embed v0.0.1.yaml
var v0_0_1_default []byte


// Map of context name -> content
var ConfigYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
}

// Map of sequential migrations
var MigrationChain = []migration.MigrationStep{}
