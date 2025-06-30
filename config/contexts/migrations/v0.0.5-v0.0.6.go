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
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Update fork block for L1 chain
			{
				Path:      []string{"context", "chains", "l1", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "4017700"}
				},
			},
			// Update fork block for L2 chain
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "4017700"}
				},
			},
			{
				Path:      []string{"context", "chains", "l1", "chain_id"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "31337"}
				},
			},
			// Update fork block for L2 chain
			{
				Path:      []string{"context", "chains", "l2", "chain_id"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "31337"}
				},
			},
			// Replace eigenlayer config with new L1/L2 structure(We are not preserving the addresses since we are migrating to holesky)
			{
				Path:      []string{"context", "eigenlayer"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					// Get the new eigenlayer structure from v0.0.6 template
					newEigenLayer := migration.ResolveNode(new, []string{"context", "eigenlayer"})
					return migration.CloneNode(newEigenLayer)
				},
			},
			{
				Path:      []string{"context", "operators", "0"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					newOperator := migration.ResolveNode(new, []string{"context", "operators", "0"})
					return migration.CloneNode(newOperator)
				},
			},
			// Remove stake field and add allocations for operator 2 (0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65)
			{
				Path:      []string{"context", "operators", "1"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					newOperator := migration.ResolveNode(new, []string{"context", "operators", "1"})
					return migration.CloneNode(newOperator)
				},
			},
			// Remove stake field for operator 3 (0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc)
			{
				Path:      []string{"context", "operators", "2"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					newOperator := migration.ResolveNode(new, []string{"context", "operators", "2"})
					return migration.CloneNode(newOperator)
				},
			},
			// Remove stake field for operator 4 (0x976EA74026E726554dB657fA54763abd0C3a0aa9)
			{
				Path:      []string{"context", "operators", "3"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					newOperator := migration.ResolveNode(new, []string{"context", "operators", "3"})
					return migration.CloneNode(newOperator)
				},
			},
			// Remove stake field for operator 5 (0x14dC79964da2C08b23698B3D3cc7Ca32193d9955)
			{
				Path:      []string{"context", "operators", "4"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					newOperator := migration.ResolveNode(new, []string{"context", "operators", "4"})
					return migration.CloneNode(newOperator)
				},
			},
		},
	}
	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Update keystore files with new versions
	err := updateKeystoreFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to update keystore files: %w", err)
	}

	// Insert stakers section after app_private_key and before operators
	contextNode := migration.ResolveNode(user, []string{"context"})
	newStakers := migration.ResolveNode(new, []string{"context", "stakers"})
	if contextNode != nil && contextNode.Kind == yaml.MappingNode && newStakers != nil {
		// Find the position of app_private_key
		var insertIndex = -1
		for i := 0; i < len(contextNode.Content)-1; i += 2 {
			if contextNode.Content[i].Value == "app_private_key" {
				insertIndex = i + 2 // Insert after app_private_key (key + value)
				break
			}
		}

		if insertIndex != -1 {
			// Create stakers key and value nodes
			stakersKey := &yaml.Node{
				Kind:        yaml.ScalarNode,
				Value:       "stakers",
				HeadComment: "List of stakers and their delegations",
			}
			stakersValue := migration.CloneNode(newStakers)

			newContent := make([]*yaml.Node, 0, len(contextNode.Content)+2)
			newContent = append(newContent, contextNode.Content[:insertIndex]...)
			newContent = append(newContent, stakersKey, stakersValue)
			newContent = append(newContent, contextNode.Content[insertIndex:]...)
			contextNode.Content = newContent
		}
	}

	// Add artifacts at the end
	if contextNode != nil && contextNode.Kind == yaml.MappingNode {
		// --- Artifacts (at the end) ---
		artifactsKey := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "artifacts",
			HeadComment: "# Release artifacts",
		}
		artifactsValue := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "component", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "artifactId", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "digest", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "registry_url", Tag: "!!str"},
				{Kind: yaml.ScalarNode, Value: "", Tag: "!!str"},
			},
		}
		// Only add artifacts if not present
		foundArtifacts := false
		for i := 0; i < len(contextNode.Content)-1; i += 2 {
			if contextNode.Content[i].Value == "artifacts" {
				foundArtifacts = true
				break
			}
		}
		if !foundArtifacts {
			contextNode.Content = append(contextNode.Content, artifactsKey, artifactsValue)
		}
	}

	// Check if transporter already exists
	hasTransporter := false
	for i := 0; i < len(contextNode.Content)-1; i += 2 {
		if contextNode.Content[i].Value == "transporter" {
			hasTransporter = true
			break
		}
	}

	if !hasTransporter {
		// Create key + value nodes
		transporterKey := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Tag:         "!!str",
			Value:       "transporter",
			HeadComment: "Stake Root Transporter configuration",
		}
		transporterValue := &yaml.Node{
			Kind: yaml.MappingNode,
			Tag:  "!!map",
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "schedule"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0 */2 * * *"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "private_key"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0x2ba58f64c57faa1073d63add89799f2a0101855a8b289b1330cb500758d5d1ee"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bls_private_key"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "0x2ba58f64c57faa1073d63add89799f2a0101855a8b289b1330cb500758d5d1ee"},
				{Kind: yaml.ScalarNode, Tag: "!!str", Value: "active_stake_roots"},
				{Kind: yaml.SequenceNode, Tag: "!!seq"},
			},
		}

		// Insert after "chains"
		inserted := false
		for i := 0; i < len(contextNode.Content)-1; i += 2 {
			if contextNode.Content[i].Value == "chains" {
				before := contextNode.Content[:i+2]
				after := contextNode.Content[i+2:]

				// Insert the transporter between before and after
				newContent := make([]*yaml.Node, 0, len(contextNode.Content)+2)
				newContent = append(newContent, before...)
				newContent = append(newContent, transporterKey, transporterValue)
				newContent = append(newContent, after...)

				// Set back the content
				contextNode.Content = newContent
				inserted = true
				break
			}
		}
		if !inserted {
			contextNode.Content = append(contextNode.Content, transporterKey, transporterValue)
		}
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