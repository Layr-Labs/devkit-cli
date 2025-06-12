package common

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// ZeusAddressData represents the addresses returned by zeus list command
type ZeusAddressData struct {
	AllocationManager string
	DelegationManager string
}

// GetZeusAddresses runs the zeus env show mainnet command and extracts core EigenLayer addresses
func GetZeusAddresses(logger iface.Logger) (*ZeusAddressData, error) {
	// Run the zeus command with JSON output
	cmd := exec.Command("zeus", "env", "show", "mainnet", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute zeus env show mainnet --json: %w - output: %s", err, string(output))
	}

	logger.InfoWithActor(iface.ActorSystem, "Parsing Zeus JSON output")

	// Parse the JSON output
	var zeusData map[string]interface{}
	if err := json.Unmarshal(output, &zeusData); err != nil {
		return nil, fmt.Errorf("failed to parse Zeus JSON output: %w", err)
	}

	// Extract the addresses
	addresses := &ZeusAddressData{}

	// Get AllocationManager address
	if val, ok := zeusData["ZEUS_DEPLOYED_AllocationManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.AllocationManager = strVal
		}
	}

	// Get DelegationManager address
	if val, ok := zeusData["ZEUS_DEPLOYED_DelegationManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.DelegationManager = strVal
		}
	}

	// Verify we have both addresses
	if addresses.AllocationManager == "" || addresses.DelegationManager == "" {
		return nil, fmt.Errorf("failed to extract required addresses from zeus output")
	}

	return addresses, nil
}

// SetMappingValue sets or updates a key-value pair in a YAML mapping node
func SetMappingValue(mappingNode *yaml.Node, keyNode, valueNode *yaml.Node) {
	if mappingNode == nil || mappingNode.Kind != yaml.MappingNode {
		return
	}

	// Look for existing key
	for i := 0; i < len(mappingNode.Content); i += 2 {
		if i+1 < len(mappingNode.Content) && mappingNode.Content[i].Value == keyNode.Value {
			// Replace existing value
			mappingNode.Content[i+1] = valueNode
			return
		}
	}

	// Key not found, append new key-value pair
	mappingNode.Content = append(mappingNode.Content, keyNode, valueNode)
}

// UpdateContextWithZeusAddresses updates the context node with addresses from Zeus
func UpdateContextWithZeusAddresses(logger iface.Logger, contextNode *yaml.Node, contextName string) error {
	addresses, err := GetZeusAddresses(logger)
	if err != nil {
		return err
	}

	// Find or create "eigenlayer" mapping entry
	parentMap := GetChildByKey(contextNode, "eigenlayer")
	if parentMap == nil {
		// Create key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "eigenlayer",
		}
		// Create empty map node
		parentMap = &yaml.Node{
			Kind:    yaml.MappingNode,
			Tag:     "!!map",
			Content: []*yaml.Node{},
		}
		contextNode.Content = append(contextNode.Content, keyNode, parentMap)
	}

	// Print the fetched addresses
	payload := ZeusAddressData{
		AllocationManager: addresses.AllocationManager,
		DelegationManager: addresses.DelegationManager,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Found addresses (marshal failed): %w", err)
	}
	logger.InfoWithActor(iface.ActorSystem, "Found addresses: %s", b)

	// Prepare nodes
	amKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "allocation_manager"}
	amVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.AllocationManager}
	dmKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "delegation_manager"}
	dmVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.DelegationManager}

	// Replace existing or append new entries
	SetMappingValue(parentMap, amKey, amVal)
	SetMappingValue(parentMap, dmKey, dmVal)

	return nil
}

// UpdateConfigWithZeusAddresses updates a ConfigWithContextConfig struct with addresses from Zeus
func UpdateConfigWithZeusAddresses(logger iface.Logger, config *ConfigWithContextConfig, contextName string) error {
	addresses, err := GetZeusAddresses(logger)
	if err != nil {
		return err
	}

	// Get the context configuration
	contextConfig, exists := config.Context[contextName]
	if !exists {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	// Initialize EigenLayer config if it doesn't exist
	if contextConfig.EigenLayer == nil {
		contextConfig.EigenLayer = &EigenLayerConfig{}
	}

	// Update the addresses
	contextConfig.EigenLayer.AllocationManager = addresses.AllocationManager
	contextConfig.EigenLayer.DelegationManager = addresses.DelegationManager

	// Update the config map
	config.Context[contextName] = contextConfig

	// Print the fetched addresses
	payload := ZeusAddressData{
		AllocationManager: addresses.AllocationManager,
		DelegationManager: addresses.DelegationManager,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Found addresses (marshal failed): %w", err)
	}
	logger.InfoWithActor(iface.ActorSystem, "Found addresses: %s", b)

	return nil
}
