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

	node := InterfaceToNode(orig)
	out := NodeToInterface(node)

	if !reflect.DeepEqual(orig, out) {
		t.Errorf("round-trip failed:\norig=%#v\nout=%#v", orig, out)
	}
}

func TestCloneNode(t *testing.T) {
	node := InterfaceToNode(map[string]interface{}{
		"foo": "bar",
	})
	node.HeadComment = "header"
	clone := CloneNode(node)

	if !reflect.DeepEqual(NodeToInterface(node), NodeToInterface(clone)) {
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
	dst := InterfaceToNode(map[string]interface{}{
		"key": "old",
		"obj": map[string]interface{}{"a": 1},
	})
	src := InterfaceToNode(map[string]interface{}{
		"key": "new",
		"obj": map[string]interface{}{"b": 2},
	})

	merged := DeepMerge(dst, src)
	out := NodeToInterface(merged).(map[string]interface{})

	if out["key"] != "new" {
		t.Errorf("expected 'new', got %v", out["key"])
	}
	obj := out["obj"].(map[string]interface{})
	if obj["a"] != 1 || obj["b"] != 2 {
		t.Errorf("merge failed: obj = %v", obj)
	}
}

func TestGetChildByKey(t *testing.T) {
	node := InterfaceToNode(map[string]interface{}{
		"foo": "bar",
	})
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

	node := InterfaceToNode(map[string]interface{}{
		"hello": "world",
	})
	err := WriteYAML(tmp, node)
	if err != nil {
		t.Fatal(err)
	}

	readNode, err := LoadYAML(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if NodeToInterface(readNode).(map[string]interface{})["hello"] != "world" {
		t.Error("LoadYAML failed to preserve data")
	}
}
