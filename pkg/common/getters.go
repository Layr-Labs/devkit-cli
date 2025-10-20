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
	// Default addresses according to context to return incase of bad context
	ALLOCATION_MANAGER_ADDRESS := SEPOLIA_ALLOCATION_MANAGER_ADDRESS
	DELEGATION_MANAGER_ADDRESS := SEPOLIA_DELEGATION_MANAGER_ADDRESS
	STRATEGY_MANAGER_ADDRESS := SEPOLIA_STRATEGY_MANAGER_ADDRESS
	KEY_REGISTRAR_ADDRESS := SEPOLIA_KEY_REGISTRAR_ADDRESS
	CROSS_CHAIN_REGISTRY_ADDRESS := SEPOLIA_CROSS_CHAIN_REGISTRY_ADDRESS
	RELEASE_MANAGER_ADDRESS := SEPOLIA_RELEASE_MANAGER_ADDRESS
	BN254_TABLE_CALCULATOR_ADDRESS := SEPOLIA_BN254_TABLE_CALCULATOR_ADDRESS
	ECDSA_TABLE_CALCULATOR_ADDRESS := SEPOLIA_ECDSA_TABLE_CALCULATOR_ADDRESS
	if contextName == "mainnet" {
		ALLOCATION_MANAGER_ADDRESS = MAINNET_ALLOCATION_MANAGER_ADDRESS
		DELEGATION_MANAGER_ADDRESS = MAINNET_DELEGATION_MANAGER_ADDRESS
		STRATEGY_MANAGER_ADDRESS = MAINNET_STRATEGY_MANAGER_ADDRESS
		KEY_REGISTRAR_ADDRESS = MAINNET_KEY_REGISTRAR_ADDRESS
		CROSS_CHAIN_REGISTRY_ADDRESS = MAINNET_CROSS_CHAIN_REGISTRY_ADDRESS
		RELEASE_MANAGER_ADDRESS = MAINNET_RELEASE_MANAGER_ADDRESS
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

	// Switch based on contexts chainId
	chainId := cfg.Context[contextName].Chains["l1"].ChainID

	// Default each address to constant if missing from discovered context
	allocationManager = ctx.EigenLayer.L1.AllocationManager
	if allocationManager == "" {
		allocationManager = GetAddressByChainId(chainId, MAINNET_ALLOCATION_MANAGER_ADDRESS, SEPOLIA_ALLOCATION_MANAGER_ADDRESS)
	}

	delegationManager = ctx.EigenLayer.L1.DelegationManager
	if delegationManager == "" {
		delegationManager = GetAddressByChainId(chainId, MAINNET_DELEGATION_MANAGER_ADDRESS, SEPOLIA_DELEGATION_MANAGER_ADDRESS)
	}

	strategyManager = ctx.EigenLayer.L1.StrategyManager
	if strategyManager == "" {
		strategyManager = GetAddressByChainId(chainId, MAINNET_STRATEGY_MANAGER_ADDRESS, SEPOLIA_STRATEGY_MANAGER_ADDRESS)
	}

	keyRegistrar = ctx.EigenLayer.L1.KeyRegistrar
	if keyRegistrar == "" {
		keyRegistrar = GetAddressByChainId(chainId, MAINNET_KEY_REGISTRAR_ADDRESS, SEPOLIA_KEY_REGISTRAR_ADDRESS)
	}

	crossChainRegistry = ctx.EigenLayer.L1.CrossChainRegistry
	if crossChainRegistry == "" {
		crossChainRegistry = GetAddressByChainId(chainId, MAINNET_CROSS_CHAIN_REGISTRY_ADDRESS, SEPOLIA_CROSS_CHAIN_REGISTRY_ADDRESS)
	}

	bn254TableCalculator = ctx.EigenLayer.L1.BN254TableCalculator
	if bn254TableCalculator == "" {
		bn254TableCalculator = GetAddressByChainId(chainId, MAINNET_BN254_TABLE_CALCULATOR_ADDRESS, SEPOLIA_BN254_TABLE_CALCULATOR_ADDRESS)
	}

	ecdsaTableCalculator = ctx.EigenLayer.L1.ECDSATableCalculator
	if ecdsaTableCalculator == "" {
		ecdsaTableCalculator = GetAddressByChainId(chainId, MAINNET_ECDSA_TABLE_CALCULATOR_ADDRESS, SEPOLIA_ECDSA_TABLE_CALCULATOR_ADDRESS)
	}

	releaseManager = ctx.EigenLayer.L1.ReleaseManager
	if releaseManager == "" {
		releaseManager = GetAddressByChainId(chainId, MAINNET_RELEASE_MANAGER_ADDRESS, SEPOLIA_RELEASE_MANAGER_ADDRESS)
	}

	return allocationManager, delegationManager, strategyManager, keyRegistrar, crossChainRegistry, bn254TableCalculator, ecdsaTableCalculator, releaseManager
}

func GetAddressByChainId(chainId int, mainnetAddress, sepoliaAddress string) string {
	if chainId == 1 {
		return mainnetAddress
	}
	return sepoliaAddress
}
