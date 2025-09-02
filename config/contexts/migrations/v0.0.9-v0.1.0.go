package contextMigrations

import (
	"fmt"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_9_to_0_1_0(user, old, new *yaml.Node) (*yaml.Node, error) {
	// Add missing strategy upgrade to move stETH from holesky to sepolia
	const (
		oldStrat = "0x7D704507b76571a51d9caE8AdDAbBFd0ba0e63d3"
		newStrat = "0x8b29d91e67b013e855EaFe0ad704aC4Ab086a574"
	)

	// Patch all changes in context
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Move Operators keystore to map
			{
				Path:      []string{"context", "operators"},
				Condition: migration.Always{},
				Base:      migration.BaseUser,
				Transform: func(_ *yaml.Node) *yaml.Node {
					avsNode := migration.ResolveNode(user, []string{"context", "avs", "address"})
					avsAddr := ""
					if avsNode != nil {
						avsAddr = avsNode.Value
					}

					ops := migration.ResolveNode(user, []string{"context", "operators"})
					if ops == nil || ops.Kind != yaml.SequenceNode {
						return ops
					}
					out := migration.CloneNode(ops)

					// Fixed operatorSet IDs
					opsetIDs := []int{0, 1}

					for _, op := range out.Content {
						if op.Kind != yaml.MappingNode {
							continue
						}

						ecdsaPath := ""
						if n := migration.ResolveNode(op, []string{"ecdsa_keystore_path"}); n != nil {
							ecdsaPath = n.Value
						}
						ecdsaPass := ""
						if n := migration.ResolveNode(op, []string{"ecdsa_keystore_password"}); n != nil {
							ecdsaPass = n.Value
						}
						blsPath := ""
						if n := migration.ResolveNode(op, []string{"bls_keystore_path"}); n != nil {
							blsPath = n.Value
						}
						blsPass := ""
						if n := migration.ResolveNode(op, []string{"bls_keystore_password"}); n != nil {
							blsPass = n.Value
						}

						// build keystores
						keystoresSeq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}

						for _, setID := range opsetIDs {
							ksMap := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

							ksMap.Content = append(ksMap.Content,
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "avs"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: avsAddr},

								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operatorSet"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", setID)},

								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ecdsa_keystore_path"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ecdsaPath},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ecdsa_keystore_password"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: ecdsaPass},

								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bls_keystore_path"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: blsPath},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bls_keystore_password"},
								&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: blsPass},
							)

							keystoresSeq.Content = append(keystoresSeq.Content, ksMap)
						}

						// drop old flat fields and any preexisting keystore/keystores
						c := op.Content[:0]
						for j := 0; j < len(op.Content); j += 2 {
							k := op.Content[j].Value
							if k == "ecdsa_keystore_path" ||
								k == "ecdsa_keystore_password" ||
								k == "bls_keystore_path" ||
								k == "bls_keystore_password" ||
								k == "keystores" {
								continue
							}
							c = append(c, op.Content[j], op.Content[j+1])
						}
						op.Content = c

						// write new keystores
						op.Content = append(op.Content,
							&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "keystores"},
							keystoresSeq,
						)
					}
					return out
				},
			},
			// Rewrite strategy_address in stakers[].deposits[]
			{
				Path:      []string{"context", "stakers"},
				Condition: migration.Exists{},
				Base:      migration.BaseUser,
				Transform: func(node *yaml.Node) *yaml.Node {
					// node is a seq of stakers
					for _, staker := range node.Content {
						if staker.Kind != yaml.MappingNode {
							continue
						}
						deposits := migration.ResolveNode(staker, []string{"deposits"})
						if deposits == nil || deposits.Kind != yaml.SequenceNode {
							continue
						}
						for _, dep := range deposits.Content {
							if dep.Kind != yaml.MappingNode {
								continue
							}
							if sa := migration.ResolveNode(dep, []string{"strategy_address"}); sa != nil && strings.EqualFold(sa.Value, oldStrat) {
								sa.Value = newStrat
							}
						}
					}
					return node
				},
			},
			// Rewrite strategy_address in operators[].allocations[]
			{
				Path:      []string{"context", "operators"},
				Condition: migration.Exists{},
				Base:      migration.BaseUser,
				Transform: func(node *yaml.Node) *yaml.Node {
					// node is a seq of operators
					for _, op := range node.Content {
						if op.Kind != yaml.MappingNode {
							continue
						}
						allocs := migration.ResolveNode(op, []string{"allocations"})
						if allocs == nil || allocs.Kind != yaml.SequenceNode {
							continue
						}
						for _, alloc := range allocs.Content {
							if alloc.Kind != yaml.MappingNode {
								continue
							}
							if sa := migration.ResolveNode(alloc, []string{"strategy_address"}); sa != nil && strings.EqualFold(sa.Value, oldStrat) {
								sa.Value = newStrat
							}
						}
					}
					return node
				},
			},
			// Update L1 PermissionController address
			{
				Path:      []string{"context", "eigenlayer", "l1", "permission_controller"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "0x44632dfBdCb6D3E21EF613B0ca8A6A0c618F5a37"}
				},
			},
			// Update L1 fork block
			{
				Path:      []string{"context", "chains", "l1", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "9085290"}
				},
			},
			// Update L2 fork block
			{
				Path:      []string{"context", "chains", "l2", "fork", "block"},
				Condition: migration.Always{},
				Transform: func(_ *yaml.Node) *yaml.Node {
					return &yaml.Node{Kind: yaml.ScalarNode, Value: "30327360"}
				},
			},
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Update BLS keystore files
	if err := updateKeystoreFiles(); err != nil {
		return nil, fmt.Errorf("failed to migrate keystore files: %w", err)
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.1.0"
	}

	return user, nil
}
