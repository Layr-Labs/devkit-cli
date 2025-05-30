package common

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// HandlerFunc defines operations on a YAML node
type HandlerFunc func(node *yaml.Node, last bool, val string) (*yaml.Node, error)

// Set up regex patterns to match bracketed index/filters in path
var (
	idxRe  = regexp.MustCompile(`^(\w+)\[(\d+)\]$`)
	filtRe = regexp.MustCompile(`^(\w+)\[([^=]+)=([^\]]+)\]$`)
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

// WriteToPath sets or overwrites a value in the YAML tree given a dot-delimited path.
func WriteToPath(root *yaml.Node, path []string, val string) (*yaml.Node, error) {
	// Ensure the provided value is clean
	val = sanitizeValue(val)

	// WorkingNode is our cursor as we descend the tree
	workingNode := root

	// Move through the path 1 segment at a time
	for i, seg := range path {
		last := i == len(path)-1

		// Attempt to match indexed bracket path (operators[0]...)
		if handler, node := tryBracketIndex(workingNode, seg); handler != nil {
			node, err := handler(node, last, val)
			if err != nil {
				return nil, err
			}
			if last {
				return root, nil
			}
			workingNode = node
			continue
		}

		// Attempt to match on an indexed path (operators.0...)
		if handler, node := tryNumericIndex(workingNode, seg); handler != nil {
			node, err := handler(node, last, val)
			if err != nil {
				return nil, err
			}
			if last {
				return root, nil
			}
			workingNode = node
			continue
		}

		// Attempt to match filter path (operators[address=123]...)
		if handler, node := tryFilterSeq(workingNode, seg); handler != nil {
			node, err := handler(node, last, val)
			if err != nil {
				return nil, err
			}
			workingNode = node
			continue
		}

		// Fallback to mapping (.key)
		node, err := handleMapping(workingNode, seg, last, val)
		if err != nil {
			return nil, err
		}
		if last {
			return root, nil
		}
		workingNode = node
	}

	return root, nil
}

// sanitizeValue trims quotes from user input
func sanitizeValue(val string) string {
	return strings.Trim(val, `"'`)
}

// tryBracketIndex detects foo[2]
func tryBracketIndex(root *yaml.Node, seg string) (HandlerFunc, *yaml.Node) {
	if m := idxRe.FindStringSubmatch(seg); m != nil {
		key, iStr := m[1], m[2]
		idx, _ := strconv.Atoi(iStr)
		seq := GetChildByKey(root, key)
		return bracketHandler(idx), seq
	}
	return nil, nil
}

// bracketHandler returns a handler for bracketed index
func bracketHandler(idx int) HandlerFunc {
	return func(seqRoot *yaml.Node, last bool, val string) (*yaml.Node, error) {
		if seqRoot == nil || seqRoot.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("not a sequence")
		}
		// Append if targeting next index
		if idx == len(seqRoot.Content) {
			seqRoot.Content = append(seqRoot.Content, &yaml.Node{Kind: yaml.MappingNode})
		}
		if idx < 0 || idx > len(seqRoot.Content)-1 {
			return nil, fmt.Errorf("index out of range: %d", idx)
		}
		target := seqRoot.Content[idx]
		if last {
			return writeScalar(target, val)
		}
		return target, nil
	}
}

// tryNumericIndex detects .0 on a sequence
func tryNumericIndex(root *yaml.Node, seg string) (HandlerFunc, *yaml.Node) {
	if root.Kind == yaml.SequenceNode && regexp.MustCompile(`^\d+$`).MatchString(seg) {
		idx, _ := strconv.Atoi(seg)
		return numericHandler(idx), root
	}
	return nil, nil
}

// numericHandler handles numeric segment on a sequence
func numericHandler(idx int) HandlerFunc {
	return func(seqRoot *yaml.Node, last bool, val string) (*yaml.Node, error) {
		// Append if next index
		if idx == len(seqRoot.Content) {
			seqRoot.Content = append(seqRoot.Content, &yaml.Node{Kind: yaml.MappingNode})
		}
		if idx < 0 || idx > len(seqRoot.Content)-1 {
			return nil, fmt.Errorf("index out of range: %d", idx)
		}
		target := seqRoot.Content[idx]
		if last {
			return writeScalar(target, val)
		}
		return target, nil
	}
}

// tryFilterSeq detects foo[key=val]
func tryFilterSeq(root *yaml.Node, seg string) (HandlerFunc, *yaml.Node) {
	if m := filtRe.FindStringSubmatch(seg); m != nil {
		key, fk, fv := m[1], m[2], m[3]
		seq := GetChildByKey(root, key)
		return filterHandler(fk, fv), seq
	}
	return nil, nil
}

// filterHandler returns a handler for filter-by-key
func filterHandler(fk, fv string) HandlerFunc {
	return func(seqRoot *yaml.Node, last bool, val string) (*yaml.Node, error) {
		if seqRoot == nil || seqRoot.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("not a sequence")
		}
		for _, item := range seqRoot.Content {
			if child := GetChildByKey(item, fk); child != nil && child.Value == fv {
				if last {
					return writeScalar(item, val)
				}
				return item, nil
			}
		}
		return nil, fmt.Errorf("no match for %s=%s", fk, fv)
	}
}

// handleMapping handles map key creation or overwrite
func handleMapping(root *yaml.Node, key string, last bool, val string) (*yaml.Node, error) {
	child := GetChildByKey(root, key)
	if child == nil {
		if last {
			return appendKeyValue(root, key, val), nil
		}
		newMap := &yaml.Node{Kind: yaml.MappingNode}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			newMap,
		)
		return newMap, nil
	}
	if last {
		return writeScalar(child, val)
	}
	return child, nil
}

// appendKeyValue appends a new key/value pair
func appendKeyValue(root *yaml.Node, key, val string) *yaml.Node {
	valNode := &yaml.Node{Kind: yaml.ScalarNode, Value: val}
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		valNode,
	)
	return root
}

// writeScalar overwrites a node with a scalar value, preserving int vs string
func writeScalar(node *yaml.Node, val string) (*yaml.Node, error) {
	node.Kind = yaml.ScalarNode
	if _, err := strconv.Atoi(val); err == nil {
		// integer literal
		node.Tag = ""
		node.Style = 0
	} else {
		// explicit string
		node.Tag = "!!str"
		node.Style = yaml.DoubleQuotedStyle
	}
	node.Value = val
	return node, nil
}
