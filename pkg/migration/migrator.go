package migration

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// PatchCondition defines a node-level condition
type PatchCondition interface {
	// ShouldApply returns true if the userNode should be patched based on oldNode
	ShouldApply(userNode, oldNode *yaml.Node) bool
}

// Available conditions
type Always struct{}
type IfUnchanged struct{}

// Always applies unconditionally
func (Always) ShouldApply(_, _ *yaml.Node) bool { return true }

// IfUnchanged applies only if userNode equals oldNode
func (IfUnchanged) ShouldApply(userNode, oldNode *yaml.Node) bool {
	ub, _ := yaml.Marshal(userNode)
	ob, _ := yaml.Marshal(oldNode)
	return bytes.Equal(ub, ob)
}

// PatchRule defines a YAML node patch rule
type PatchRule struct {
	// Path: sequence of map keys or sequence indices (as strings)
	Path []string
	// Condition: returns true if the patch should apply
	Condition PatchCondition
	// Transform: optional node-level transformation on the new node
	Transform func(newNode *yaml.Node) *yaml.Node
	// Remove: if true, delete the node instead of patching
	Remove bool
}

// MigrationStep represents one version-to-version migration
type MigrationStep struct {
	From    string
	To      string
	Apply   func(user, oldDef, newDef *yaml.Node) (*yaml.Node, error)
	OldYAML []byte
	NewYAML []byte
}

// PatchEngine applies a set of PatchRule against a user YAML AST preserving order, comments, and anchors
type PatchEngine struct {
	Old   *yaml.Node
	New   *yaml.Node
	User  *yaml.Node
	Rules []PatchRule
}

// VersionGreaterThan uses semantic dot-separated compare for strings
type VersionComparator func(string, string) bool

// Known errors which we can ignore
var ErrAlreadyUpToDate = errors.New("already up to date")

// Apply walks each rule, and when Condition is met, either removes the node or replaces it with a (transformed) copy
func (e *PatchEngine) Apply() error {
	for _, rule := range e.Rules {
		userNode := ResolveNode(e.User, rule.Path)
		oldNode := ResolveNode(e.Old, rule.Path)
		newNode := ResolveNode(e.New, rule.Path)

		// insert new node if missing
		if userNode == nil {
			parent, _ := findParent(e.User, rule.Path)
			if parent == nil {
				continue
			}
			repl := ResolveNode(e.New, rule.Path)
			if repl == nil {
				continue
			}
			if rule.Transform != nil {
				repl = rule.Transform(CloneNode(repl))
			}
			insertNode(parent, len(parent.Content), rule.Path[len(rule.Path)-1], CloneNode(repl))
			continue
		}

		// check if we should apply (compare users to old)
		if rule.Condition.ShouldApply(userNode, oldNode) {
			parent, idx := findParent(e.User, rule.Path)
			// if deleting
			if rule.Remove {
				if parent != nil {
					deleteNode(parent, idx)
				}
				continue
			}
			// choose replacement
			repl := newNode
			if rule.Transform != nil {
				repl = rule.Transform(CloneNode(newNode))
			}
			// overwrite in place
			*userNode = *CloneNode(repl)
		}
	}
	return nil
}

// Run all migrations after current version upto latestVersion according to migrationChain
func MigrateYaml(logger iface.Logger, path string, latestVersion string, migrationChain []MigrationStep) error {

	// Load as YAML AST
	userNode, err := common.LoadYAML(path)
	if err != nil {
		return fmt.Errorf("load error %s: %v", path, err)
	}

	// Extract version scalar
	verNode := ResolveNode(userNode, []string{"version"})
	if verNode == nil {
		return fmt.Errorf("no version field %s", path)
	}
	from := verNode.Value
	to := latestVersion

	// Continue and don't say anything if the user version is latest
	if from == to {
		return ErrAlreadyUpToDate
	}
	logger.Info("Migrating %s v%s -> v%s", path, from, to)

	// Perform node-based migration
	migrated, err := MigrateNode(userNode, from, to, migrationChain)
	if err != nil {
		return fmt.Errorf("migration failed %s: %v", path, err)
	}

	// Write AST back to disk
	if err := common.WriteYAML(path, migrated); err != nil {
		return fmt.Errorf("failed to write %s: %v", path, err)
	}

	return nil
}

// MigrateNode runs all MigrationStep from 'from' to 'to' on the provided user YAML AST, returning the migrated AST
func MigrateNode(
	user *yaml.Node,
	from, to string,
	chain []MigrationStep,
) (*yaml.Node, error) {
	// End early if from == to
	if from == to {
		return user, ErrAlreadyUpToDate
	}
	current := from
	for _, step := range chain {
		if step.From != current {
			continue
		}
		if versionGreaterThan(step.To, to) {
			break
		}

		oldDef := &yaml.Node{}
		if err := yaml.Unmarshal(step.OldYAML, oldDef); err != nil {
			return nil, fmt.Errorf("failed to unmarshal old default for %s: %w", step.From, err)
		}

		newDef := &yaml.Node{}
		if err := yaml.Unmarshal(step.NewYAML, newDef); err != nil {
			return nil, fmt.Errorf("failed to unmarshal new default for %s: %w", step.To, err)
		}

		var err error
		user, err = step.Apply(user, oldDef, newDef)
		if err != nil {
			return nil, fmt.Errorf("migration %s->%s failed: %w", step.From, step.To, err)
		}
		current = step.To
	}
	if current != to {
		return nil, fmt.Errorf("incomplete migration: ended at %s, target %s", current, to)
	}
	return user, nil
}

// ResolveNode walks the YAML AST following path segments and returns the node or nil
func ResolveNode(root *yaml.Node, path []string) *yaml.Node {
	// if DocumentNode, unwrap
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	curr := root
	for _, p := range path {
		found := false
		switch curr.Kind {
		case yaml.MappingNode:
			for j := 0; j < len(curr.Content)-1; j += 2 {
				if curr.Content[j].Value == p {
					curr = curr.Content[j+1]
					found = true
					break
				}
			}
		case yaml.SequenceNode:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(curr.Content) {
				return nil
			}
			curr = curr.Content[idx]
			found = true
		default:
			return nil
		}
		if !found {
			return nil
		}
	}
	return curr
}

// CloneNode deep-copies a *yaml.Node, preserving comments and anchors
func CloneNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	c := *n
	c.Content = make([]*yaml.Node, len(n.Content))
	for i, ch := range n.Content {
		c.Content[i] = CloneNode(ch)
	}
	return &c
}

// EnsureKeyWithComment adds “# comment\nkey:” (with an empty sequence value) if key missing
func EnsureKeyWithComment(root *yaml.Node, path []string, comment string) {
	// locate parent mapping
	parentPath := path[:len(path)-1]
	keyName := path[len(path)-1]

	parent := ResolveNode(root, parentPath)
	if parent == nil || parent.Kind != yaml.MappingNode {
		return
	}

	// skip if key already present
	for i := 0; i < len(parent.Content)-1; i += 2 {
		if parent.Content[i].Value == keyName {
			return
		}
	}

	// append key (with comment) + empty seq
	keyNode := &yaml.Node{
		Kind:        yaml.ScalarNode,
		Tag:         "!!str",
		Value:       keyName,
		HeadComment: comment,
	}
	valNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq",
	}
	parent.Content = append(parent.Content, keyNode, valNode)
}

// findParent locates the parent mapping or sequence node and the index/key position
func findParent(root *yaml.Node, path []string) (*yaml.Node, int) {
	if len(path) == 0 {
		return nil, -1
	}
	// if DocumentNode, unwrap
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	curr := root
	for _, p := range path[:len(path)-1] {
		switch curr.Kind {
		case yaml.MappingNode:
			for j := 0; j < len(curr.Content)-1; j += 2 {
				if curr.Content[j].Value == p {
					// next node is value
					curr = curr.Content[j+1]
					break
				}
			}
		case yaml.SequenceNode:
			idx, _ := strconv.Atoi(p)
			if idx < len(curr.Content) {
				curr = curr.Content[idx]
			}
		}
	}
	// now curr is parent of target
	target := path[len(path)-1]
	// mapping parent
	if curr.Kind == yaml.MappingNode {
		for j := 0; j < len(curr.Content)-1; j += 2 {
			if curr.Content[j].Value == target {
				return curr, j
			}
		}
	}
	// sequence parent
	if curr.Kind == yaml.SequenceNode {
		idx, _ := strconv.Atoi(target)
		return curr, idx
	}
	return curr, -1
}

// insertNode inserts a key/value pair from a mapping or an element from a sequence
func insertNode(parent *yaml.Node, idx int, key string, value *yaml.Node) {
	if parent.Kind == yaml.MappingNode {
		k := &yaml.Node{Kind: yaml.ScalarNode, Value: key, Tag: "!!str"}
		parent.Content = append(parent.Content, nil, nil)
		copy(parent.Content[idx+2:], parent.Content[idx:])
		parent.Content[idx] = k
		parent.Content[idx+1] = value
	}
}

// deleteNode removes a key/value pair from a mapping or an element from a sequence
func deleteNode(parent *yaml.Node, idx int) {
	if parent.Kind == yaml.MappingNode && idx%2 == 0 {
		// remove key and value
		parent.Content = append(parent.Content[:idx], parent.Content[idx+2:]...)
	} else if parent.Kind == yaml.SequenceNode {
		// remove element
		parent.Content = append(parent.Content[:idx], parent.Content[idx+1:]...)
	}
}

func versionLessThan(v1, v2 string) bool {
	s1 := strings.Split(v1, ".")
	s2 := strings.Split(v2, ".")
	for i := 0; i < len(s1) && i < len(s2); i++ {
		n1, _ := strconv.Atoi(s1[i])
		n2, _ := strconv.Atoi(s2[i])
		if n1 < n2 {
			return true
		} else if n1 > n2 {
			return false
		}
	}
	return len(s1) < len(s2)
}

func versionGreaterThan(v1, v2 string) bool {
	return versionLessThan(v2, v1)
}
