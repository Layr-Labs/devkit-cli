package contexts

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	contextMigrations "github.com/Layr-Labs/devkit-cli/config/contexts/migrations"
)

// Set the latest version
const LatestVersion = "0.1.1"

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

//go:embed v0.0.8.yaml
var v0_0_8_default []byte

//go:embed v0.0.9.yaml
var v0_0_9_default []byte

//go:embed v0.1.0.yaml
var v0_1_0_default []byte

//go:embed v0.1.1.yaml
var v0_1_1_default []byte

// Map of context name -> content
var ContextYamls = map[string][]byte{
	"0.0.1": v0_0_1_default,
	"0.0.2": v0_0_2_default,
	"0.0.3": v0_0_3_default,
	"0.0.4": v0_0_4_default,
	"0.0.5": v0_0_5_default,
	"0.0.6": v0_0_6_default,
	"0.0.7": v0_0_7_default,
	"0.0.8": v0_0_8_default,
	"0.0.9": v0_0_9_default,
	"0.1.0": v0_1_0_default,
	"0.1.1": v0_1_1_default,
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
	{
		From:    "0.0.7",
		To:      "0.0.8",
		Apply:   contextMigrations.Migration_0_0_7_to_0_0_8,
		OldYAML: v0_0_7_default,
		NewYAML: v0_0_8_default,
	},
	{
		From:    "0.0.8",
		To:      "0.0.9",
		Apply:   contextMigrations.Migration_0_0_8_to_0_0_9,
		OldYAML: v0_0_8_default,
		NewYAML: v0_0_9_default,
	},
	{
		From:    "0.0.9",
		To:      "0.1.0",
		Apply:   contextMigrations.Migration_0_0_9_to_0_1_0,
		OldYAML: v0_0_9_default,
		NewYAML: v0_1_0_default,
	},
	{
		From:    "0.1.0",
		To:      "0.1.1",
		Apply:   contextMigrations.Migration_0_1_0_to_0_1_1,
		OldYAML: v0_1_0_default,
		NewYAML: v0_1_1_default,
	},
}

func MigrateContexts(logger iface.Logger) (int, error) {
	// Count the number of contexts we migrate
	contextsMigrated := 0

	// Set path for context yamls
	contextDir := filepath.Join("config", "contexts")

	// Read all contexts/*.yamls
	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return 0, fmt.Errorf("unable to read context directory: %v", err)
	}

	// Attempt to upgrade every entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		contextPath := filepath.Join(contextDir, e.Name())

		// Migrate the context
		err := migration.MigrateYaml(logger, contextPath, LatestVersion, MigrationChain)
		// Check for already upto date and ignore
		alreadyUptoDate := errors.Is(err, migration.ErrAlreadyUpToDate)

		// For every other error, migration failed
		if err != nil && !alreadyUptoDate {
			logger.Error("failed to migrate: %v", err)
			continue
		}

		// If context was migrated
		if !alreadyUptoDate {
			// Incr number of contextsMigrated
			contextsMigrated += 1

			// If migration succeeds
			logger.Info("Migrated %s\n", contextPath)
		}
	}

	return contextsMigrated, nil
}
