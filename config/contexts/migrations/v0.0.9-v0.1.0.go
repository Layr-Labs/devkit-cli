package contextMigrations

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_9_to_0_1_0(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Move Operators keystore to map
			{
				Path:      []string{"context", "operators"},
				Condition: migration.Always{},
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
		},
	}

	if err := engine.Apply(); err != nil {
		return nil, err
	}

	// Upgrade the version
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.1.0"
	}

	return user, nil
}
