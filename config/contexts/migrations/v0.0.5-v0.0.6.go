package contextMigrations

import (
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
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "22640530"}
				},
			},
			// Update fork block for L2 chain
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "22640530"}
				},
			},
			// Add strategy_manager to eigenlayer config
			{
				Path:      []string{"context", "eigenlayer"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					eigenLayerMap := migration.CloneNode(migration.ResolveNode(user, []string{"context", "eigenlayer"}))
					strategyManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "strategy_manager"}
					strategyManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Value: "0x858646372CC42E1A627fcE94aa7A7033e7CF075A"}
					eigenLayerMap.Content = append(eigenLayerMap.Content, strategyManagerKey, strategyManagerVal)
					return eigenLayerMap
				},
			},
			// Remove stake field and add allocations for operator 1 (0x90F79bf6EB2c4f870365E785982E1f101E93b906)
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

	// Add stakers section with comment first, then populate it
	migration.EnsureKeyWithComment(user, []string{"context", "stakers"}, "List of stakers and their delegations")

	// Now populate the stakers with content from new config
	stakersNode := migration.ResolveNode(user, []string{"context", "stakers"})
	newStakers := migration.ResolveNode(new, []string{"context", "stakers"})
	if stakersNode != nil && newStakers != nil {
		*stakersNode = *migration.CloneNode(newStakers)
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.6"
	}
	return user, nil
}
