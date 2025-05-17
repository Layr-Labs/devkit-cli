package common

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadYAML reads a YAML file from the given path and unmarshals it into a *yaml.Node
func LoadYAML(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, fmt.Errorf("unmarshal to node: %w", err)
	}
	return &node, nil
}

// WriteYAML encodes a *yaml.Node to YAML and writes it to the specified file path
func WriteYAML(path string, node *yaml.Node) error {
	buf := &bytes.Buffer{}
	enc := yaml.NewEncoder(buf)
	enc.SetIndent(2)
	if err := enc.Encode(node); err != nil {
		return fmt.Errorf("encode yaml: %w", err)
	}
	enc.Close()
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// InterfaceToNode converts a Go value (typically map[string]interface{}) into a *yaml.Node
func InterfaceToNode(v interface{}) (*yaml.Node, error) {
	if v == nil {
		return &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: "",
		}, nil
	}

	var node yaml.Node
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal failed: %w", err)
	}
	if err := yaml.Unmarshal(b, &node); err != nil {
		return nil, fmt.Errorf("unmarshal to node failed: %w", err)
	}
	if len(node.Content) == 0 {
		return nil, fmt.Errorf("empty YAML node content")
	}
	return node.Content[0], nil
}

// NodeToInterface converts a *yaml.Node back into a Go interface{}
func NodeToInterface(node *yaml.Node) (interface{}, error) {
	if node == nil {
		return nil, nil
	}
	b, err := yaml.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("marshal node failed: %w", err)
	}
	var out interface{}
	if err := yaml.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal to interface failed: %w", err)
	}
	return CleanYAML(out), nil
}

// Normalizes YAML-parsed structures
func CleanYAML(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[fmt.Sprint(k)] = CleanYAML(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = CleanYAML(v)
		}
	}
	return v
}

// GetChildByKey returns the value node associated with the given key from a MappingNode
func GetChildByKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		k := node.Content[i]
		if k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// CloneNode performs a deep copy of a *yaml.Node, including its content and comments
func CloneNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	c := *n
	c.LineComment = n.LineComment
	c.HeadComment = n.HeadComment
	c.FootComment = n.FootComment
	if n.Content != nil {
		c.Content = make([]*yaml.Node, len(n.Content))
		for i, child := range n.Content {
			c.Content[i] = CloneNode(child)
		}
	}
	return &c
}

// DeepMerge merges two *yaml.Node trees recursively
//
// - If both nodes are MappingNodes, their keys are merged:
//   - Matching keys: recurse if both values are maps, else src replaces dst
//   - New keys in src are appended to dst
//
// - For non-mapping nodes, src replaces dst
// - All merged values are deep-cloned to avoid shared references
func DeepMerge(dst, src *yaml.Node) *yaml.Node {
	if src == nil {
		return CloneNode(dst)
	}
	if dst == nil {
		return CloneNode(src)
	}
	if dst.Kind != yaml.MappingNode || src.Kind != yaml.MappingNode {
		return CloneNode(src)
	}

	for i := 0; i < len(src.Content); i += 2 {
		srcKey := src.Content[i]
		srcVal := src.Content[i+1]

		found := false
		for j := 0; j < len(dst.Content); j += 2 {
			dstKey := dst.Content[j]
			dstVal := dst.Content[j+1]

			if dstKey.Value == srcKey.Value {
				found = true
				if dstVal != nil && srcVal != nil && dstVal.Kind == yaml.MappingNode && srcVal.Kind == yaml.MappingNode {
					dst.Content[j+1] = DeepMerge(dstVal, srcVal)
				} else {
					dst.Content[j+1] = CloneNode(srcVal)
				}
				break
			}
		}

		if !found {
			dst.Content = append(dst.Content, CloneNode(srcKey), CloneNode(srcVal))
		}
	}
	return dst
}
