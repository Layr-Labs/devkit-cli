package common

import (
	"os"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestInterfaceNodeRoundTrip(t *testing.T) {
	orig := map[string]interface{}{
		"foo": "bar",
		"num": 42,
		"arr": []interface{}{"a", "b"},
	}

	node, err := InterfaceToNode(orig)
	if err != nil {
		t.Fatalf("InterfaceToNode failed: %v", err)
	}

	out, err := NodeToInterface(node)
	if err != nil {
		t.Fatalf("NodeToInterface failed: %v", err)
	}

	if !reflect.DeepEqual(orig, out) {
		t.Errorf("round-trip failed:\norig=%#v\nout=%#v", orig, out)
	}
}
func TestCloneNode(t *testing.T) {
	node, err := InterfaceToNode(map[string]interface{}{
		"foo": "bar",
	})
	if err != nil {
		t.Fatalf("InterfaceToNode failed: %v", err)
	}
	node.HeadComment = "header"
	clone := CloneNode(node)

	origOut, err := NodeToInterface(node)
	if err != nil {
		t.Fatalf("NodeToInterface failed: %v", err)
	}
	cloneOut, err := NodeToInterface(clone)
	if err != nil {
		t.Fatalf("NodeToInterface failed (clone): %v", err)
	}

	if !reflect.DeepEqual(origOut, cloneOut) {
		t.Error("CloneNode failed to preserve content")
	}
	if clone == node {
		t.Error("CloneNode returned same pointer")
	}
	if clone.HeadComment != "header" {
		t.Error("CloneNode did not preserve comments")
	}
}

func TestDeepMerge(t *testing.T) {
	dst, err := InterfaceToNode(map[string]interface{}{
		"key": "old",
		"obj": map[string]interface{}{"a": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	src, err := InterfaceToNode(map[string]interface{}{
		"key": "new",
		"obj": map[string]interface{}{"b": 2},
	})
	if err != nil {
		t.Fatal(err)
	}

	merged := DeepMerge(dst, src)
	out, err := NodeToInterface(merged)
	if err != nil {
		t.Fatal(err)
	}
	result := out.(map[string]interface{})

	if result["key"] != "new" {
		t.Errorf("expected 'new', got %v", result["key"])
	}
	obj := result["obj"].(map[string]interface{})
	if obj["a"] != 1 || obj["b"] != 2 {
		t.Errorf("merge failed: obj = %v", obj)
	}
}

func TestGetChildByKey(t *testing.T) {
	node, err := InterfaceToNode(map[string]interface{}{
		"foo": "bar",
	})
	if err != nil {
		t.Fatal(err)
	}
	val := GetChildByKey(node, "foo")
	if val == nil || val.Value != "bar" {
		t.Errorf("expected 'bar', got %v", val)
	}
	if GetChildByKey(node, "missing") != nil {
		t.Error("expected nil for missing key")
	}
}

func TestLoadWriteYAML(t *testing.T) {
	tmp := "test.yaml"
	defer os.Remove(tmp)

	node, err := InterfaceToNode(map[string]interface{}{
		"hello": "world",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := WriteYAML(tmp, node); err != nil {
		t.Fatal(err)
	}

	readNode, err := LoadYAML(tmp)
	if err != nil {
		t.Fatal(err)
	}

	out, err := NodeToInterface(readNode)
	if err != nil {
		t.Fatal(err)
	}
	if out.(map[string]interface{})["hello"] != "world" {
		t.Error("LoadYAML failed to preserve data")
	}

	// Empty file test
	emptyFile := "empty.yaml"
	if err := os.WriteFile(emptyFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(emptyFile)

	node, err = LoadYAML(emptyFile)
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if node == nil || len(node.Content) == 0 {
		// acceptable
	} else {
		t.Errorf("expected nil or empty content, got: %#v", node)
	}

	// Non-mapping root node (e.g. array root)
	nonMappingFile := "list_root.yaml"
	if err := os.WriteFile(nonMappingFile, []byte("- a\n- b"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(nonMappingFile)

	node, err = LoadYAML(nonMappingFile)
	if err != nil {
		t.Fatal(err)
	}
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 || node.Content[0].Kind != yaml.SequenceNode {
		t.Errorf("expected DocumentNode with SequenceNode child, got: %#v", node)
	}
}

func TestInterfaceToNode_Nil(t *testing.T) {
	node, err := InterfaceToNode(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag != "!!null" {
		t.Errorf("expected !!null node, got: %#v", node)
	}
}

func TestNodeToInterface_Nil(t *testing.T) {
	out, err := NodeToInterface(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if out != nil {
		t.Errorf("expected nil output, got: %#v", out)
	}
}

func TestCloneNode_Nil(t *testing.T) {
	if CloneNode(nil) != nil {
		t.Error("CloneNode(nil) should return nil")
	}
}

func TestDeepMerge_BothNil(t *testing.T) {
	if DeepMerge(nil, nil) != nil {
		t.Error("DeepMerge(nil, nil) should return nil")
	}
}

func TestDeepMerge_EitherNil(t *testing.T) {
	src, _ := InterfaceToNode(map[string]interface{}{"x": 1})
	dst, _ := InterfaceToNode(map[string]interface{}{"x": 0})

	if DeepMerge(nil, src) == nil {
		t.Error("DeepMerge(nil, src) should not return nil")
	}
	if DeepMerge(dst, nil) == nil {
		t.Error("DeepMerge(dst, nil) should not return nil")
	}
}

func TestGetChildByKey_NilNode(t *testing.T) {
	if GetChildByKey(nil, "x") != nil {
		t.Error("GetChildByKey(nil, _) should return nil")
	}
}

func TestLoadYAML_InvalidYAML(t *testing.T) {
	file := "invalid.yaml"
	defer os.Remove(file)

	if err := os.WriteFile(file, []byte(":\n - bad"), 0644); err != nil {
		t.Fatalf("failed to write invalid YAML: %v", err)
	}

	_, err := LoadYAML(file)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestWriteToPath_SequenceIndex(t *testing.T) {
	// start with two-item list under "ops"
	root, _ := InterfaceToNode(map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"a": 1},
			map[string]interface{}{"a": 2},
		},
	})
	// overwrite index 1
	withOpsNode, err := WriteToPath(root, []string{"ops", "1", "a"}, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, err := NodeToInterface(withOpsNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr := out.(map[string]interface{})["ops"].([]interface{})
	if arr[1].(map[string]interface{})["a"] != 42 {
		t.Errorf("expected 42 at ops[1].a, got %v", arr[1])
	}

	// append a third entry
	withOpsNode, err = WriteToPath(withOpsNode, []string{"ops", "2", "a"}, "99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err = NodeToInterface(withOpsNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr = out.(map[string]interface{})["ops"].([]interface{})
	if len(arr) != 3 || arr[2].(map[string]interface{})["a"] != 99 {
		t.Errorf("expected appended ops[2].a=99, got %#v", arr)
	}
}

func TestWriteToPath_BracketIndex(t *testing.T) {
	root, _ := InterfaceToNode(map[string]interface{}{
		"ops": []interface{}{
			map[string]interface{}{"a": "foo"},
		},
	})
	withOpsNode, err := WriteToPath(root, []string{"ops[0]", "a"}, "bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := NodeToInterface(withOpsNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	a := out.(map[string]interface{})["ops"].([]interface{})[0].(map[string]interface{})["a"]
	if a != "bar" {
		t.Errorf("expected ops[0].a=bar, got %v", a)
	}
}

func TestWriteToPath_FilterByKey(t *testing.T) {
	root, _ := InterfaceToNode(map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{"id": "x", "name": "Alice"},
			map[string]interface{}{"id": "y", "name": "Bob"},
		},
	})
	withOpsNode, err := WriteToPath(root, []string{"users[id=y]", "name"}, "Bobbert")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := NodeToInterface(withOpsNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	name := out.(map[string]interface{})["users"].([]interface{})[1].(map[string]interface{})["name"]
	if name != "Bobbert" {
		t.Errorf("expected users[id=y].name=Bobbert, got %v", name)
	}
}

func TestWriteToPath_MappingCreation(t *testing.T) {
	root, _ := InterfaceToNode(map[string]interface{}{})
	withOpsNode, err := WriteToPath(root, []string{"foo", "bar"}, "baz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out, err := NodeToInterface(withOpsNode)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := out.(map[string]interface{})["foo"].(map[string]interface{})["bar"]
	if m != "baz" {
		t.Errorf("expected foo.bar=baz, got %v", m)
	}
}
