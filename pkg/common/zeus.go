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
	TaskMailbox          string `json:"taskMailbox"`
	PermissionController string `json:"permissionController"`
}

type L2ZeusAddressData struct {
	OperatorTableUpdater     string `json:"operatorTableUpdater"`
	ECDSACertificateVerifier string `json:"ecdsaCertificateVerifier"`
	BN254CertificateVerifier string `json:"bn254CertificateVerifier"`
	TaskMailbox              string `json:"taskMailbox"`
}

// GetZeusAddresses runs the zeus env show mainnet command and extracts core EigenLayer addresses
func GetZeusAddresses(ctx context.Context, logger iface.Logger) (*L1ZeusAddressData, *L2ZeusAddressData, error) {

	// Run the zeus command with JSON output
	cmd := exec.CommandContext(ctx, "zeus", "env", "show", "testnet-sepolia", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute zeus env show testnet-sepolia --json: %w - output: %s", err, string(output))
	}

	// Parse the JSON output
	var l1ZeusData map[string]interface{}
	if err := json.Unmarshal(output, &l1ZeusData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Zeus JSON output: %w", err)
	}

	l2cmd := exec.CommandContext(context.Background(), "zeus", "env", "show", "testnet-base-sepolia", "--json")
	l2output, err := l2cmd.CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute zeus env show testnet-base-sepolia --json: %w - output: %s", err, string(l2output))
	}

	// Parse the L2 JSON output
	var l2ZeusData map[string]interface{}
	if err := json.Unmarshal(l2output, &l2ZeusData); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Zeus JSON output: %w", err)
	}

	logger.Info("Parsing Zeus JSON output")

	// Extract the addresses
	l1Addresses := &L1ZeusAddressData{}
	l2Addresses := &L2ZeusAddressData{}

	// Get AllocationManager address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_AllocationManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.AllocationManager = strVal
		}
	}

	// Get DelegationManager address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_DelegationManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.DelegationManager = strVal
		}
	}

	// Get StrategyManager address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_StrategyManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.StrategyManager = strVal
		}
	}

	// Get CrossChainRegistry address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_CrossChainRegistry_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.CrossChainRegistry = strVal
		}
	}

	// Get KeyRegistrar address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_KeyRegistrar_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.KeyRegistrar = strVal
		}
	}

	// Get ReleaseManager address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_ReleaseManager_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.ReleaseManager = strVal
		}
	}

	// Get OperatorTableUpdater address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_OperatorTableUpdater_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.OperatorTableUpdater = strVal
		}
	}

	// Get TaskMailbox address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_TaskMailbox_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.TaskMailbox = strVal
		}
	}

	// Get PermissionController address
	if val, ok := l1ZeusData["ZEUS_DEPLOYED_PermissionController_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l1Addresses.PermissionController = strVal
		}
	}

	// Verify we have both addresses
	if l1Addresses.AllocationManager == "" || l1Addresses.DelegationManager == "" || l1Addresses.StrategyManager == "" || l1Addresses.CrossChainRegistry == "" || l1Addresses.KeyRegistrar == "" || l1Addresses.ReleaseManager == "" || l1Addresses.OperatorTableUpdater == "" {
		logger.Warn("failed to extract required addresses from zeus output")
		return nil, nil, fmt.Errorf("failed to extract required addresses from zeus output")
	}

	// Get OperatorTableUpdater address
	if val, ok := l2ZeusData["ZEUS_DEPLOYED_OperatorTableUpdater_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l2Addresses.OperatorTableUpdater = strVal
		}
	}

	// Get ECDSACertificateVerifier address
	if val, ok := l2ZeusData["ZEUS_DEPLOYED_ECDSACertificateVerifier_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l2Addresses.ECDSACertificateVerifier = strVal
		}
	}

	// Get BN254CertificateVerifier address
	if val, ok := l2ZeusData["ZEUS_DEPLOYED_BN254CertificateVerifier_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l2Addresses.BN254CertificateVerifier = strVal
		}
	}

	// Get TaskMailbox address
	if val, ok := l2ZeusData["ZEUS_DEPLOYED_TaskMailbox_Proxy"]; ok {
		if strVal, ok := val.(string); ok {
			l2Addresses.TaskMailbox = strVal
		}
	}

	return l1Addresses, l2Addresses, nil
}

// UpdateContextWithZeusAddresses updates the context configuration with addresses from Zeus
func UpdateContextWithZeusAddresses(context context.Context, logger iface.Logger, ctx *yaml.Node, contextName string) error {

	logger.Info("Fetching EigenLayer core addresses for L1 and L2 from Zeus...")
	l1Addresses, l2Addresses, err := GetZeusAddresses(context, logger)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"l1": l1Addresses,
		"l2": l2Addresses,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("found addresses (marshal failed): %w", err)
	}
	logger.Info("Found addresses: %s", b)

	logger.Info("Updating context with Zeus addresses...")

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
	allocationManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "allocation_manager"}
	allocationManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.AllocationManager}
	delegationManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "delegation_manager"}
	delegationManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.DelegationManager}
	strategyManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "strategy_manager"}
	strategyManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.StrategyManager}
	crossChainRegistryKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "cross_chain_registry"}
	crossChainRegistryVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.CrossChainRegistry}
	keyRegistrarKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "key_registrar"}
	keyRegistrarVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.KeyRegistrar}
	releaseManagerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "release_manager"}
	releaseManagerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.ReleaseManager}
	operatorTableUpdaterKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operator_table_updater"}
	operatorTableUpdaterVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.OperatorTableUpdater}
	taskMailboxKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "task_mailbox"}
	taskMailboxVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.TaskMailbox}
	permissionControllerKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "permission_controller"}
	permissionControllerVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l1Addresses.PermissionController}

	// Replace existing or append new entries in l1 section
	SetMappingValue(l1Map, allocationManagerKey, allocationManagerVal)
	SetMappingValue(l1Map, delegationManagerKey, delegationManagerVal)
	SetMappingValue(l1Map, strategyManagerKey, strategyManagerVal)
	SetMappingValue(l1Map, crossChainRegistryKey, crossChainRegistryVal)
	SetMappingValue(l1Map, keyRegistrarKey, keyRegistrarVal)
	SetMappingValue(l1Map, releaseManagerKey, releaseManagerVal)
	SetMappingValue(l1Map, operatorTableUpdaterKey, operatorTableUpdaterVal)
	SetMappingValue(l1Map, taskMailboxKey, taskMailboxVal)
	SetMappingValue(l1Map, permissionControllerKey, permissionControllerVal)

	// Find or create "l2" mapping entry under eigenlayer
	l2Map := GetChildByKey(parentMap, "l2")
	if l2Map == nil {
		// Create l2 key node
		l2KeyNode := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "l2"}
		// Create empty l2 map node
		l2Map = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{}}
		parentMap.Content = append(parentMap.Content, l2KeyNode, l2Map)
	}

	// Prepare nodes for L2 contracts
	l2OperatorTableUpdaterKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "operator_table_updater"}
	l2OperatorTableUpdaterVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.OperatorTableUpdater}
	l2ECDSACertificateVerifierKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "ecdsa_certificate_verifier"}
	l2ECDSACertificateVerifierVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.ECDSACertificateVerifier}
	bn254Key := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "bn254_certificate_verifier"}
	bn254Val := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.BN254CertificateVerifier}
	l2TaskMailboxKey := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "task_mailbox"}
	l2TaskMailboxVal := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: l2Addresses.TaskMailbox}

	// Replace existing or append new entries in l2 section
	SetMappingValue(l2Map, l2OperatorTableUpdaterKey, l2OperatorTableUpdaterVal)
	SetMappingValue(l2Map, l2ECDSACertificateVerifierKey, l2ECDSACertificateVerifierVal)
	SetMappingValue(l2Map, bn254Key, bn254Val)
	SetMappingValue(l2Map, l2TaskMailboxKey, l2TaskMailboxVal)

	return nil
}
