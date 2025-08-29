package contextMigrations

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/config"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_7_to_0_0_8(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Add ECDSA keystore fields to operators
			{
				Path:      []string{"context", "operators"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					// Get the user's existing operators node
					userOperators := migration.ResolveNode(user, []string{"context", "operators"})
					if userOperators == nil || userOperators.Kind != yaml.SequenceNode {
						return userOperators
					}

					// Clone to avoid modifying the original
					operatorsNode := migration.CloneNode(userOperators)

					// Update each operator
					for opIndex, opNode := range operatorsNode.Content {
						if opNode.Kind != yaml.MappingNode {
							continue
						}

						// Use operator index (1-based) for keystore naming
						operatorNum := fmt.Sprintf("%d", opIndex+1)

						// Check if this operator already has ECDSA keystore fields
						hasECDSAKeystore := false
						for i := 0; i < len(opNode.Content)-1; i += 2 {
							if opNode.Content[i].Value == "ecdsa_keystore_path" {
								hasECDSAKeystore = true
								break
							}
						}

						if !hasECDSAKeystore {
							// Find the position after ecdsa_key to insert the new fields
							insertIndex := -1
							for i := 0; i < len(opNode.Content)-1; i += 2 {
								if opNode.Content[i].Value == "ecdsa_key" {
									insertIndex = i + 2
									break
								}
							}

							if insertIndex != -1 {
								// Create new keystore fields
								ecdsaKeystorePath := &yaml.Node{
									Kind:  yaml.ScalarNode,
									Value: "ecdsa_keystore_path",
								}
								ecdsaKeystorePathValue := &yaml.Node{
									Kind:  yaml.ScalarNode,
									Value: fmt.Sprintf("keystores/operator%s.ecdsa.keystore.json", operatorNum),
								}
								ecdsaKeystorePassword := &yaml.Node{
									Kind:  yaml.ScalarNode,
									Value: "ecdsa_keystore_password",
								}
								ecdsaKeystorePasswordValue := &yaml.Node{
									Kind:  yaml.ScalarNode,
									Value: "testpass",
								}

								// Insert the new fields
								newContent := make([]*yaml.Node, 0, len(opNode.Content)+4)
								newContent = append(newContent, opNode.Content[:insertIndex]...)
								newContent = append(newContent, ecdsaKeystorePath, ecdsaKeystorePathValue)
								newContent = append(newContent, ecdsaKeystorePassword, ecdsaKeystorePasswordValue)
								newContent = append(newContent, opNode.Content[insertIndex:]...)
								opNode.Content = newContent
							}
						}
					}

					return operatorsNode
				},
			},
			// Update BLS keystore paths to new naming convention
			{
				Path:      []string{"context", "operators"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					// Get the user's existing operators node
					userOperators := migration.ResolveNode(user, []string{"context", "operators"})
					if userOperators == nil || userOperators.Kind != yaml.SequenceNode {
						return userOperators
					}

					// Clone to avoid modifying the original
					operatorsNode := migration.CloneNode(userOperators)

					// Update each operator's BLS keystore path
					for _, opNode := range operatorsNode.Content {
						if opNode.Kind != yaml.MappingNode {
							continue
						}

						// Find and update bls_keystore_path
						for i := 0; i < len(opNode.Content)-1; i += 2 {
							if opNode.Content[i].Value == "bls_keystore_path" {
								// Update the path from operatorN.keystore.json to operatorN.bls.keystore.json
								oldPath := opNode.Content[i+1].Value
								if strings.HasSuffix(oldPath, ".keystore.json") && !strings.Contains(oldPath, ".bls.") {
									// Extract operator number and update path
									parts := strings.Split(oldPath, "/")
									filename := parts[len(parts)-1]
									if strings.HasPrefix(filename, "operator") && strings.HasSuffix(filename, ".keystore.json") {
										operatorNum := strings.TrimSuffix(strings.TrimPrefix(filename, "operator"), ".keystore.json")
										newPath := fmt.Sprintf("keystores/operator%s.bls.keystore.json", operatorNum)
										opNode.Content[i+1].Value = newPath
									}
								}
								break
							}
						}
					}

					return operatorsNode
				},
			},
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Migrate keystore files (rename BLS and add ECDSA)
	if err := migrateKeystoreFiles(); err != nil {
		return nil, fmt.Errorf("failed to migrate keystore files: %w", err)
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.8"
	}

	return user, nil
}

// migrateKeystoreFiles renames existing BLS keystores to new naming convention
// and adds ECDSA keystores from embedded files
func migrateKeystoreFiles() error {
	// Get the project directory
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	keystoreDir := filepath.Join(projectDir, "keystores")

	// Ensure keystores directory exists
	if _, err := os.Stat(keystoreDir); os.IsNotExist(err) {
		// No keystores directory, nothing to migrate
		return nil
	}

	// Migrate each operator's keystores
	for i := 1; i <= 5; i++ {
		operatorNum := fmt.Sprintf("operator%d", i)

		// Old and new BLS keystore names
		oldBLSName := fmt.Sprintf("%s.keystore.json", operatorNum)
		newBLSName := fmt.Sprintf("%s.bls.keystore.json", operatorNum)
		oldBLSPath := filepath.Join(keystoreDir, oldBLSName)
		newBLSPath := filepath.Join(keystoreDir, newBLSName)

		// Rename BLS keystore if it exists with old name
		if _, err := os.Stat(oldBLSPath); err == nil {
			// File exists, rename it
			if err := os.Rename(oldBLSPath, newBLSPath); err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", oldBLSName, newBLSName, err)
			}
		}

		// Add ECDSA keystore from embedded files
		ecdsaName := fmt.Sprintf("%s.ecdsa.keystore.json", operatorNum)
		ecdsaPath := filepath.Join(keystoreDir, ecdsaName)

		// Only create ECDSA keystore if it doesn't already exist
		if _, err := os.Stat(ecdsaPath); os.IsNotExist(err) {
			// Get ECDSA keystore content from embedded files
			content, exists := config.KeystoreEmbeds[ecdsaName]
			if !exists {
				// ECDSA keystore not found in embedded files, skip
				// This allows for graceful handling if some operators don't have ECDSA keystores
				continue
			}

			// Write ECDSA keystore
			if err := os.WriteFile(ecdsaPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write ECDSA keystore %s: %w", ecdsaName, err)
			}
		}
	}

	return nil
}
