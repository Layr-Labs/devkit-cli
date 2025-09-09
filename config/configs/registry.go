package configs

import (
	_ "embed"
	"errors"
	"fmt"
	"path/filepath"

	configMigrations "github.com/Layr-Labs/devkit-cli/config/configs/migrations"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
)

// Set the latest version
const LatestVersion = "0.0.3"

// --
// Versioned configs
// --

//go:embed v0.0.1.yaml
var v0_0_1_default []byte

//go:embed v0.0.2.yaml
var v0_0_2_default []byte

//go:embed v0.0.3.yaml
var v0_0_3_default []byte

// Map of context name -> content
var ConfigYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
	"0.0.2": v0_0_2_default,
	"0.0.3": v0_0_3_default,
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
	{
		From:    "0.0.2",
		To:      "0.0.3",
		Apply:   configMigrations.Migration_0_0_2_to_0_0_3,
		OldYAML: v0_0_2_default,
		NewYAML: v0_0_3_default,
	},
}

func MigrateConfig(logger iface.Logger) (int, error) {
	// Set path for context yamls
	configDir := filepath.Join("config")
	configPath := filepath.Join(configDir, "config.yaml")

	// Migrate the config
	err := migration.MigrateYaml(logger, configPath, LatestVersion, MigrationChain)
	// Check for already upto date and ignore
	alreadyUptoDate := errors.Is(err, migration.ErrAlreadyUpToDate)

	// For any other error, migration has failed
	if err != nil && !alreadyUptoDate {
		return 0, fmt.Errorf("failed to migrate: %v", err)
	}

	// If config was migrated
	if !alreadyUptoDate {
		logger.Info("Migrated %s\n", configPath)

		return 1, nil
	}

	return 0, nil
}
