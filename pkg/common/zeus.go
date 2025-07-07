package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"gopkg.in/yaml.v3"
)

// L1ZeusAddressData represents the addresses returned by zeus list command
type L1ZeusAddressData struct {
	AllocationManager    string `json:"allocationManager"`
	DelegationManager    string `json:"delegationManager"`
	StrategyManager      string `json:"strategyManager"`
	CrossChainRegistry   string `json:"crossChainRegistry"`
	KeyRegistrar         string `json:"keyRegistrar"`
	ReleaseManager       string `json:"releaseManager"`
	OperatorTableUpdater string `json:"operatorTableUpdater"`
}

// GetZeusAddresses runs the zeus env show mainnet command and extracts core EigenLayer addresses
func GetZeusAddresses(ctx context.Context, logger iface.Logger) (*L1ZeusAddressData, error) {

	// Run the zeus command with JSON output
	cmd := exec.CommandContext(context.Background(), "zeus", "env", "show", "testnet-sepolia", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute zeus env show testnet-sepolia --json: %w - output: %s", err, string(output))
	}

	logger.Info("Parsing Zeus JSON output")

	// Parse the JSON output
	var zeusData map[string]interface{}
	if err := json.Unmarshal(output, &zeusData); err != nil {
		return nil, fmt.Errorf("failed to parse Zeus JSON output: %w", err)
	}

	// Extract the addresses
	addresses := &L1ZeusAddressData{}

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

	// Get CrossChainRegistry address
	if val, ok := zeusData["ZEUS_DEPLOYED_CrossChainRegistry_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.CrossChainRegistry = strVal
		}
	}

	// Get KeyRegistrar address
	if val, ok := zeusData["ZEUS_DEPLOYED_KeyRegistrar_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.KeyRegistrar = strVal
		}
	}

	// Get ReleaseManager address
	if val, ok := zeusData["ZEUS_DEPLOYED_ReleaseManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.ReleaseManager = strVal
		}
	}

	// Get OperatorTableUpdater address
	if val, ok := zeusData["ZEUS_DEPLOYED_OperatorTableUpdater_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			addresses.OperatorTableUpdater = strVal
		}
	}

	// Verify we have both addresses
	if addresses.AllocationManager == "" || addresses.DelegationManager == "" || addresses.StrategyManager == "" || addresses.CrossChainRegistry == "" || addresses.KeyRegistrar == "" || addresses.ReleaseManager == "" || addresses.OperatorTableUpdater == "" {
		logger.Warn("failed to extract required addresses from zeus output")
		return nil, fmt.Errorf("failed to extract required addresses from zeus output")
	}

	return addresses, nil
}

// UpdateContextWithZeusAddresses updates the context configuration with addresses from Zeus
// TODO: Currently commented out as Zeus doesn't support the new L1/L2 contract structure
func UpdateContextWithZeusAddresses(context context.Context, logger iface.Logger, ctx *yaml.Node, contextName string) error {
	// Zeus integration temporarily disabled for new L1/L2 structure

	addresses, err := GetZeusAddresses(context, logger)
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
	payload := L1ZeusAddressData{
		AllocationManager:    addresses.AllocationManager,
		DelegationManager:    addresses.DelegationManager,
		StrategyManager:      addresses.StrategyManager,
		CrossChainRegistry:   addresses.CrossChainRegistry,
		KeyRegistrar:         addresses.KeyRegistrar,
		ReleaseManager:       addresses.ReleaseManager,
		OperatorTableUpdater: addresses.OperatorTableUpdater,
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
	ccrKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cross_chain_registry"}
	ccrVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.CrossChainRegistry}
	krKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key_registrar"}
	krVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.KeyRegistrar}
	rmKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "release_manager"}
	rmVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.ReleaseManager}
	otuKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operator_table_updater"}
	otuVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: addresses.OperatorTableUpdater}

	// Replace existing or append new entries in l1 section
	SetMappingValue(l1Map, amKey, amVal)
	SetMappingValue(l1Map, dmKey, dmVal)
	SetMappingValue(l1Map, smKey, smVal)
	SetMappingValue(l1Map, ccrKey, ccrVal)
	SetMappingValue(l1Map, krKey, krVal)
	SetMappingValue(l1Map, rmKey, rmVal)
	SetMappingValue(l1Map, otuKey, otuVal)

	return nil
}
