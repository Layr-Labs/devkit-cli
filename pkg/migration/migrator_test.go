package migration

import (
	"errors"
	"testing"

	"gopkg.in/yaml.v3"
)

// helper to parse YAML into *yaml.Node
func testNode(t *testing.T, input string) *yaml.Node {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(input), &node); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	// unwrap DocumentNode
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return &node
}

func TestResolveNode(t *testing.T) {
	src := `
version: v1
nested:
  key: value
list:
  - a
  - b
`
	node := testNode(t, src)

	// scalar
	vn := ResolveNode(node, []string{"version"})
	if vn == nil || vn.Value != "v1" {
		t.Error("ResolveNode version failed")
	}

	// nested
	kn := ResolveNode(node, []string{"nested", "key"})
	if kn == nil || kn.Value != "value" {
		t.Error("ResolveNode nested.key failed")
	}

	// list
	ln := ResolveNode(node, []string{"list", "1"})
	if ln == nil || ln.Value != "b" {
		t.Error("ResolveNode list[1] failed")
	}
}

func TestCloneNode(t *testing.T) {
	src := `key: orig`
	n := testNode(t, src)
	clone := CloneNode(n)

	// modify clone
	clone.Content[1].Value = "new"

	orig := testNode(t, src)
	ov := ResolveNode(orig, []string{"key"})
	if ov == nil || ov.Value != "orig" {
		t.Error("CloneNode did not deep copy")
	}
}

func TestPatchEngine_Apply(t *testing.T) {
	yamlOld := `
version: v1
param: old
`
	yamlNew := `
version: v1
param: new
`
	yamlUser := `
version: v1
param: old
`

	oldDef := testNode(t, yamlOld)
	newDef := testNode(t, yamlNew)
	user := testNode(t, yamlUser)

	engine := PatchEngine{
		Old:  oldDef,
		New:  newDef,
		User: user,
		Rules: []PatchRule{{
			Path:      []string{"param"},
			Condition: IfUnchanged{},
		}},
	}
	if err := engine.Apply(); err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	on := ResolveNode(user, []string{"param"})
	if on == nil || on.Value != "new" {
		t.Errorf("Expected param=new, got %v", on.Value)
	}
}

func TestMigrateNode_AlreadyUpToDate(t *testing.T) {
	yamlUser := `version: v1`
	node := testNode(t, yamlUser)
	// empty chain
	_, err := MigrateNode(node, "v1", "v1", nil)
	if !errors.Is(err, ErrAlreadyUpToDate) {
		t.Error("Expected ErrAlreadyUpToDate")
	}
}
