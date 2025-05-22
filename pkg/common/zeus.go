package common

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// ZeusAddressData represents the addresses returned by zeus list command
type ZeusAddressData struct {
	AllocationManager string
	DelegationManager string
}

// GetZeusAddresses runs the zeus env show mainnet command and extracts core EigenLayer addresses
func GetZeusAddresses() (*ZeusAddressData, error) {
	log, _ := GetLogger()
	// Run the zeus command with JSON output
	cmd := exec.Command("zeus", "env", "show", "mainnet", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute zeus env show mainnet --json: %w - output: %s", err, string(output))
	}

	log.Info("Parsing Zeus JSON output")

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

// UpdateContextWithZeusAddresses updates the context configuration with addresses from Zeus
func UpdateContextWithZeusAddresses(ctx *ConfigWithContextConfig, contextName string) error {
	log, _ := GetLogger()
	addresses, err := GetZeusAddresses()
	if err != nil {
		return err
	}

	// Ensure the context and eigenlayer section exist
	envCtx, ok := ctx.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	// Create EigenLayer config if it doesn't exist
	if envCtx.EigenLayer == nil {
		envCtx.EigenLayer = &EigenLayerConfig{}
	}

	log.Info("Updating context with addresses:")
	log.Info("AllocationManager: %s", addresses.AllocationManager)
	log.Info("DelegationManager: %s", addresses.DelegationManager)

	// Update addresses
	envCtx.EigenLayer.AllocationManager = addresses.AllocationManager
	envCtx.EigenLayer.DelegationManager = addresses.DelegationManager

	// Update context in the config
	ctx.Context[contextName] = envCtx

	return nil
}
