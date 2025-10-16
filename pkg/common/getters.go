package common

import (
	"fmt"
	"os"
)

func GetForkUrlDefault(contextName string, cfg *ConfigWithContextConfig, chainName string) (string, error) {
	// Check in env first for L1 fork url
	l1ForkUrl := os.Getenv("L1_FORK_URL")
	if chainName == "l1" && l1ForkUrl != "" {
		return l1ForkUrl, nil
	}

	// Check in env first for L2 fork url
	l2ForkUrl := os.Getenv("L2_FORK_URL")
	if chainName == "l2" && l2ForkUrl != "" {
		return l2ForkUrl, nil
	}

	// Fallback to context defined value
	chainConfig, found := cfg.Context[contextName].Chains[chainName]
	if !found {
		return "", fmt.Errorf("failed to get chainConfig for chainName : %s", chainName)
	}
	if chainConfig.Fork.Url == "" {
		return "", fmt.Errorf("fork-url not set for %s; set fork-url in ./config/context/%s.yaml or .env and consult README for guidance", chainName, contextName)
	}
	return chainConfig.Fork.Url, nil
}

// GetEigenLayerAddresses returns EigenLayer L1 addresses from the context config
// Falls back to constants if not found in context
func GetEigenLayerAddresses(contextName string, cfg *ConfigWithContextConfig) (allocationManager, delegationManager, strategyManager, keyRegistrar, crossChainRegistry, bn254TableCalculator, ecdsaTableCalculator, releaseManager string) {
	// Default middleware addresses according to context
	var BN254_TABLE_CALCULATOR_ADDRESS = SEPOLIA_BN254_TABLE_CALCULATOR_ADDRESS
	var ECDSA_TABLE_CALCULATOR_ADDRESS = SEPOLIA_ECDSA_TABLE_CALCULATOR_ADDRESS
	if contextName == "mainnet" {
		BN254_TABLE_CALCULATOR_ADDRESS = MAINNET_BN254_TABLE_CALCULATOR_ADDRESS
		ECDSA_TABLE_CALCULATOR_ADDRESS = MAINNET_ECDSA_TABLE_CALCULATOR_ADDRESS
	}

	// Return constants for undefined context
	if cfg == nil || cfg.Context == nil {
		return ALLOCATION_MANAGER_ADDRESS, DELEGATION_MANAGER_ADDRESS, STRATEGY_MANAGER_ADDRESS, KEY_REGISTRAR_ADDRESS, CROSS_CHAIN_REGISTRY_ADDRESS, BN254_TABLE_CALCULATOR_ADDRESS, ECDSA_TABLE_CALCULATOR_ADDRESS, RELEASE_MANAGER_ADDRESS
	}

	// Return constants for missing context
	ctx, found := cfg.Context[contextName]
	if !found || ctx.EigenLayer == nil {
		return ALLOCATION_MANAGER_ADDRESS, DELEGATION_MANAGER_ADDRESS, STRATEGY_MANAGER_ADDRESS, KEY_REGISTRAR_ADDRESS, CROSS_CHAIN_REGISTRY_ADDRESS, BN254_TABLE_CALCULATOR_ADDRESS, ECDSA_TABLE_CALCULATOR_ADDRESS, RELEASE_MANAGER_ADDRESS
	}

	// Default each address to constant if missing from discovered context
	allocationManager = ctx.EigenLayer.L1.AllocationManager
	if allocationManager == "" {
		allocationManager = ALLOCATION_MANAGER_ADDRESS
	}

	delegationManager = ctx.EigenLayer.L1.DelegationManager
	if delegationManager == "" {
		delegationManager = DELEGATION_MANAGER_ADDRESS
	}
	strategyManager = ctx.EigenLayer.L1.StrategyManager
	if strategyManager == "" {
		strategyManager = STRATEGY_MANAGER_ADDRESS
	}
	keyRegistrar = ctx.EigenLayer.L1.KeyRegistrar
	if keyRegistrar == "" {
		keyRegistrar = KEY_REGISTRAR_ADDRESS
	}

	crossChainRegistry = ctx.EigenLayer.L1.CrossChainRegistry
	if crossChainRegistry == "" {
		crossChainRegistry = CROSS_CHAIN_REGISTRY_ADDRESS
	}

	bn254TableCalculator = ctx.EigenLayer.L1.BN254TableCalculator
	if bn254TableCalculator == "" {
		bn254TableCalculator = BN254_TABLE_CALCULATOR_ADDRESS
	}

	ecdsaTableCalculator = ctx.EigenLayer.L1.ECDSATableCalculator
	if ecdsaTableCalculator == "" {
		ecdsaTableCalculator = ECDSA_TABLE_CALCULATOR_ADDRESS
	}

	releaseManager = ctx.EigenLayer.L1.ReleaseManager
	if releaseManager == "" {
		releaseManager = RELEASE_MANAGER_ADDRESS
	}

	return allocationManager, delegationManager, strategyManager, keyRegistrar, crossChainRegistry, bn254TableCalculator, ecdsaTableCalculator, releaseManager
}
