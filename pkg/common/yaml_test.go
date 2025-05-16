package common

import (
	"os"
	"reflect"
	"testing"
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
}
