package contextMigrations

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/config"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_5_to_0_0_6(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{ /* … */ }
	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Update keystore files with new versions
	err := updateKeystoreFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to update keystore files: %w", err)
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.6"
	}
	return user, nil
}

func updateKeystoreFiles() error {
	// Get the project directory (assuming we're in the project root)
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	keystoreDir := filepath.Join(projectDir, "keystores")

	// Ensure keystores directory exists
	if err := os.MkdirAll(keystoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create keystores directory: %w", err)
	}

	// List of keystore files to update
	keystoreFiles := []string{
		"operator1.keystore.json",
		"operator2.keystore.json",
		"operator3.keystore.json",
		"operator4.keystore.json",
		"operator5.keystore.json",
	}

	// Update each keystore file with the new version from the embedded files
	for _, filename := range keystoreFiles {
		// Get the new keystore content from embedded files
		content, exists := config.KeystoreEmbeds[filename]
		if !exists {
			return fmt.Errorf("keystore file %s not found in embedded files", filename)
		}

		// Write the updated content to the file
		filePath := filepath.Join(keystoreDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write keystore file %s: %w", filename, err)
		}
	}

	return nil
}
