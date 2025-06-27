package common

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// ZeusAddressData represents the addresses returned by zeus list command
type ZeusAddressData struct {
	AllocationManager string `json:"allocationManager"`
	DelegationManager string `json:"delegationManager"`
	StrategyManager   string `json:"strategyManager"`
}

// GetZeusAddresses runs the zeus env show mainnet command and extracts core EigenLayer addresses
// TODO: Currently commented out as Zeus doesn't support the new L1/L2 contract structure
func GetZeusAddresses(logger iface.Logger) (*ZeusAddressData, error) {
	// Zeus integration temporarily disabled for new L1/L2 structure
	return nil, fmt.Errorf("Zeus integration is currently disabled for the new L1/L2 contract structure")

	/* Temporarily commented out until Zeus supports new structure
	// Run the zeus command with JSON output
	cmd := exec.Command("zeus", "env", "show", "mainnet", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute zeus env show mainnet --json: %w - output: %s", err, string(output))
	}

	logger.Info("Parsing Zeus JSON output")

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

	// Get StrategyManager address
	if val, ok := zeusData["ZEUS_DEPLOYED_StrategyManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.StrategyManager = strVal
		}
	}

	// Verify we have both addresses
	if addresses.AllocationManager == "" || addresses.DelegationManager == "" {
		return nil, fmt.Errorf("failed to extract required addresses from zeus output")
	}

	return addresses, nil
	*/
}

// UpdateContextWithZeusAddresses updates the context configuration with addresses from Zeus
// TODO: Currently commented out as Zeus doesn't support the new L1/L2 contract structure
func UpdateContextWithZeusAddresses(logger iface.Logger, ctx *yaml.Node, contextName string) error {
	// Zeus integration temporarily disabled for new L1/L2 structure
	logger.Info("Zeus integration is currently disabled for the new L1/L2 contract structure")
	return fmt.Errorf("Zeus integration is currently disabled for the new L1/L2 contract structure")

	/* Temporarily commented out until Zeus supports new structure
	addresses, err := GetZeusAddresses(logger)
	if err != nil {
		return err
	}

	// Find or create "eigenlayer" mapping entry
	parentMap := GetChildByKey(ctx, "eigenlayer")
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
		ctx.Content = append(ctx.Content, keyNode, parentMap)
	}

	// Print the fetched addresses
	payload := ZeusAddressData{
		AllocationManager: addresses.AllocationManager,
		DelegationManager: addresses.DelegationManager,
		StrategyManager:   addresses.StrategyManager,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Found addresses (marshal failed): %w", err)
	}
	logger.Info("Found addresses: %s", b)

	// Find or create "l1" mapping entry under eigenlayer
	l1Map := GetChildByKey(parentMap, "l1")
	if l1Map == nil {
		// Create l1 key node
		l1KeyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "l1",
		}
		// Create empty l1 map node
		l1Map = &yaml.Node{
			Kind:    yaml.MappingNode,
			Tag:     "!!map",
			Content: []*yaml.Node{},
		}
		parentMap.Content = append(parentMap.Content, l1KeyNode, l1Map)
	}

	// Prepare nodes for L1 contracts
	amKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "allocation_manager"}
	amVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.AllocationManager}
	dmKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "delegation_manager"}
	dmVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.DelegationManager}
	smKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "strategy_manager"}
	smVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.StrategyManager}

	// Replace existing or append new entries in l1 section
	SetMappingValue(l1Map, amKey, amVal)
	SetMappingValue(l1Map, dmKey, dmVal)
	SetMappingValue(l1Map, smKey, smVal)

	return nil
	*/
}
