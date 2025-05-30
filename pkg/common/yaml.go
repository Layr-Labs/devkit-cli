package common

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
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

// LoadMap loads the YAML file at path into a map[string]interface{}
func LoadMap(path string) (map[string]interface{}, error) {
	node, err := LoadYAML(path)
	if err != nil {
		return nil, err
	}
	iface, err := NodeToInterface(node)
	if err != nil {
		return nil, err
	}
	m, ok := iface.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map at top level, got %T", iface)
	}
	return m, nil
}

// YamlToMap unmarshals your YAML into interface{}, normalizes all maps, and returns the top‐level map[string]interface{}
func YamlToMap(b []byte) (map[string]interface{}, error) {
	var raw interface{}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	norm := Normalize(raw)
	if m, ok := norm.(map[string]interface{}); ok {
		return m, nil
	}
	return nil, fmt.Errorf("expected top‐level map, got %T", norm)
}

// Normalize will walk any nested map[interface{}]interface{} -> map[string]interface{}, and also recurse into []interface{}
func Normalize(i interface{}) interface{} {
	switch v := i.(type) {
	case map[interface{}]interface{}:
		m2 := make(map[string]interface{}, len(v))
		for key, val := range v {
			m2[fmt.Sprint(key)] = Normalize(val)
		}
		return m2
	case []interface{}:
		for idx, elem := range v {
			v[idx] = Normalize(elem)
		}
		return v
	default:
		return v
	}
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

// WriteMap takes a map[string]interface{} and writes it back to a file
func WriteMap(path string, m map[string]interface{}) error {
	node, err := InterfaceToNode(m)
	if err != nil {
		return err
	}
	return WriteYAML(path, node)
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

// WriteToPath navigates the given YAML node tree according to the dot-delimited
// path segments in `path`, creating intermediate mapping nodes as needed, and
// sets or overwrites the final key to the provided string value. Returns the
// original root node for chaining or further use.
func WriteToPath(root *yaml.Node, path []string, val string) (*yaml.Node, error) {
	// workingNode is our cursor as we descend the tree
	workingNode := root

	// move through the path segment at a time
	for i, seg := range path {
		last := i == len(path)-1

		// look for an existing child mapping key under workingNode
		child := GetChildByKey(workingNode, seg)

		// if no child exists, create or append
		if child == nil {
			if last {
				// final segment missing—append key + scalar value,
				// tagging as int if possible, else as quoted string
				valNode := &yaml.Node{Kind: yaml.ScalarNode}
				if _, err := strconv.Atoi(val); err == nil {
					// integer literal
					valNode.Value = val
				} else {
					// explicit string
					valNode.Tag = "!!str"
					valNode.Style = yaml.DoubleQuotedStyle
					valNode.Value = val
				}
				workingNode.Content = append(workingNode.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: seg},
					valNode,
				)
				break
			}
			// intermediate segment missing—create nested map
			newMap := &yaml.Node{Kind: yaml.MappingNode}
			workingNode.Content = append(workingNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: seg},
				newMap,
			)
			workingNode = newMap
			continue
		}

		if last {
			// final segment exists—overwrite its scalar value,
			// preserving int if possible, else using quoted string
			child.Kind = yaml.ScalarNode
			if _, err := strconv.Atoi(val); err == nil {
				child.Tag = "" // let YAML infer !!int
				child.Style = 0
				child.Value = val
			} else {
				child.Tag = "!!str"
				child.Style = yaml.DoubleQuotedStyle
				child.Value = val
			}
		} else {
			// descend into existing mapping node
			workingNode = child
		}
	}

	return root, nil
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

// ListYaml prints the contents of a YAML file to stdout, preserving order and comments.
// It rejects non-.yaml/.yml extensions and surfaces precise errors.
func ListYaml(filePath string, logger iface.Logger) error {
	// verify file exists and is regular
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("cannot stat %s: %w", filePath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", filePath)
	}

	// ensure extension is .yaml or .yml
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" {
		return fmt.Errorf("unsupported extension %q: only .yaml/.yml allowed", ext)
	}

	// load the raw YAML node tree so we preserve ordering
	rootNode, err := LoadYAML(filePath)
	if err != nil {
		return fmt.Errorf("❌ Failed to read or parse %s: %v\n\n", filePath, err)
	}

	// header
	logger.Info("--- %s ---", filePath)

	// encode the node back to YAML on stdout, preserving order & comments
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(rootNode); err != nil {
		enc.Close()
		return fmt.Errorf("Failed to emit %s: %v\n\n", filePath, err)
	}
	enc.Close()

	return nil
}

// SetMappingValue sets mapNode[keyNode.Value] = valNode, replacing existing or appending if missing.
func SetMappingValue(mapNode, keyNode, valNode *yaml.Node) {
	// Ensure mapNode is a MappingNode
	if mapNode.Kind != yaml.MappingNode {
		return
	}

	// Scan existing entries (Content holds [key, value, key, value, …])
	for i := 0; i < len(mapNode.Content); i += 2 {
		existingKey := mapNode.Content[i]
		if existingKey.Value == keyNode.Value {
			// replace the paired value node
			mapNode.Content[i+1] = valNode
			return
		}
	}

	// Not found: append key and value
	mapNode.Content = append(mapNode.Content, keyNode, valNode)
}
