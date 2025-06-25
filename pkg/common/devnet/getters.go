package devnet

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

// GetDevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to CHAIN_ARGS constant if value is empty.
func GetDevnetChainArgsOrDefault(cfg *common.ConfigWithContextConfig) string {
	args := []string{} // TODO(nova) : Get chain args from config.yaml ?  For now using default
	if len(args) == 0 {
		return CHAIN_ARGS
	}
	return " "
}

// GetDevnetChainImageOrDefault returns the devnet chain image,
// falling back to FOUNDRY_IMAGE if not provided.
func GetDevnetChainImageOrDefault(cfg *common.ConfigWithContextConfig) string {
	image := "" // TODO(nova): Get Foundry image from config.yaml ? For now using default
	if image == "" {
		return FOUNDRY_IMAGE
	}

	return image
}

func FileExistsInRoot(filename string) bool {
	// Assumes current working directory is the root of the project
	_, err := os.Stat(filename)
	return err == nil || !os.IsNotExist(err)
}

func GetDevnetChainIdOrDefault(cfg *common.ConfigWithContextConfig, chainName string, logger iface.Logger) (int, error) {
	// Check in env first for L1 chain id
	l1ChainId := os.Getenv("L1_CHAIN_ID")
	l1ChainIdInt, err := strconv.Atoi(l1ChainId)
	if chainName == "l1" && err != nil && l1ChainIdInt != 0 {
		logger.Info("L1_CHAIN_ID is set to %d", l1ChainIdInt)
		return l1ChainIdInt, nil
	}

	// Check in env first for L2 chain id
	l2ChainId := os.Getenv("L2_CHAIN_ID")
	l2ChainIdInt, err := strconv.Atoi(l2ChainId)
	if chainName == "l2" && err != nil && l2ChainIdInt != 0 {
		logger.Info("L2_CHAIN_ID is set to %d", l2ChainIdInt)
		return l2ChainIdInt, nil
	}

	// Fallback to context defined value or DefaultAnvilChainId if undefined
	chainConfig, found := cfg.Context[DEVNET_CONTEXT].Chains[chainName]
	if !found {
		logger.Error("failed to get chainConfig for chainName : %s", chainName)
		return common.DefaultAnvilChainId, fmt.Errorf("failed to get chainConfig for chainName : %s", chainName)
	}
	if chainConfig.ChainID == 0 {
		logger.Error("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
		return common.DefaultAnvilChainId, fmt.Errorf("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
	}
	logger.Info("chain_id is set to %d", chainConfig.ChainID)
	return chainConfig.ChainID, nil
}

func GetDevnetBlockTimeOrDefault(cfg *common.ConfigWithContextConfig, chainName string) (int, error) {
	// Check in env first for L1 block time
	l1BlockTime := os.Getenv("L1_BLOCK_TIME")
	l1BlockTimeInt, err := strconv.Atoi(l1BlockTime)
	if chainName == "l1" && err != nil && l1BlockTimeInt != 0 {
		return l1BlockTimeInt, nil
	}

	// Check in env first for l2 block time
	l2BlockTime := os.Getenv("L2_BLOCK_TIME")
	l2BlockTimeInt, err := strconv.Atoi(l2BlockTime)
	if chainName == "l2" && err != nil && l2BlockTimeInt != 0 {
		return l2BlockTimeInt, nil
	}

	// Fallback to context defined value or 12s if undefined
	chainConfig, found := cfg.Context[DEVNET_CONTEXT].Chains[chainName]
	if !found {
		return 12, fmt.Errorf("failed to get chainConfig for chainName : %s", chainName)
	}
	if chainConfig.Fork.BlockTime == 0 {
		return 12, fmt.Errorf("block-time not set for %s; set block-time in ./config/context/devnet.yaml or .env", chainName)
	}

	return chainConfig.Fork.BlockTime, nil
}

func GetDevnetRPCUrlDefault(cfg *common.ConfigWithContextConfig, chainName string) (string, error) {
	// Check in env first for L1 RPC url
	l1RPCUrl := os.Getenv("L1_RPC_URL")
	if chainName == "l1" && l1RPCUrl != "" {
		return l1RPCUrl, nil
	}

	// Check in env first for L2 RPC url
	l2RPCUrl := os.Getenv("L2_RPC_URL")
	if chainName == "l2" && l2RPCUrl != "" {
		return l2RPCUrl, nil
	}

	// Fallback to context defined value
	chainConfig, found := cfg.Context[DEVNET_CONTEXT].Chains[chainName]
	if !found {
		return "", fmt.Errorf("failed to get chainConfig for chainName : %s", chainName)
	}
	if chainConfig.RPCURL == "" {
		return "", fmt.Errorf("rpc_url not set for %s; set rpc_url in ./config/context/devnet.yaml or .env and consult README for guidance", chainName)
	}
	return chainConfig.RPCURL, nil
}

func GetDevnetForkUrlDefault(cfg *common.ConfigWithContextConfig, chainName string) (string, error) {
	// Check in env first for L1 fork url
	l1ForkUrl := os.Getenv("L1_FORK_URL")
	if chainName == "l1" && l1ForkUrl != "" {
		return l1ForkUrl, nil
	}

	// Check in env first for l2 fork url
	l2ForkUrl := os.Getenv("L2_FORK_URL")
	if chainName == "l2" && l2ForkUrl != "" {
		return l2ForkUrl, nil
	}

	// Fallback to context defined value
	chainConfig, found := cfg.Context[DEVNET_CONTEXT].Chains[chainName]
	if !found {
		return "", fmt.Errorf("failed to get chainConfig for chainName : %s", chainName)
	}
	if chainConfig.Fork.Url == "" {
		return "", fmt.Errorf("fork-url not set for %s; set fork-url in ./config/context/devnet.yaml or .env and consult README for guidance", chainName)
	}
	return chainConfig.Fork.Url, nil
}

// GetEigenLayerAddresses returns EigenLayer L1 addresses from the context config
// Falls back to constants if not found in context
func GetEigenLayerAddresses(cfg *common.ConfigWithContextConfig) (allocationManager, delegationManager string, strategyManager string, keyRegistrar string, crossChainRegistry string, bn254TableCalculator string, releaseManager string) {
	if cfg == nil || cfg.Context == nil {
		return ALLOCATION_MANAGER_ADDRESS, DELEGATION_MANAGER_ADDRESS, STRATEGY_MANAGER_ADDRESS, KEY_REGISTRAR_ADDRESS, CROSS_CHAIN_REGISTRY_ADDRESS, BN254_TABLE_CALCULATOR_ADDRESS, RELEASE_MANAGER_ADDRESS
	}

	devnetCtx, found := cfg.Context[DEVNET_CONTEXT]
	if !found || devnetCtx.EigenLayer == nil {
		return ALLOCATION_MANAGER_ADDRESS, DELEGATION_MANAGER_ADDRESS, STRATEGY_MANAGER_ADDRESS, KEY_REGISTRAR_ADDRESS, CROSS_CHAIN_REGISTRY_ADDRESS, BN254_TABLE_CALCULATOR_ADDRESS, RELEASE_MANAGER_ADDRESS
	}

	allocationManager = devnetCtx.EigenLayer.L1.AllocationManager
	if allocationManager == "" {
		allocationManager = ALLOCATION_MANAGER_ADDRESS
	}

	delegationManager = devnetCtx.EigenLayer.L1.DelegationManager
	if delegationManager == "" {
		delegationManager = DELEGATION_MANAGER_ADDRESS
	}
	strategyManager = devnetCtx.EigenLayer.L1.StrategyManager
	if strategyManager == "" {
		strategyManager = STRATEGY_MANAGER_ADDRESS
	}
	keyRegistrar = devnetCtx.EigenLayer.L1.KeyRegistrar
	if keyRegistrar == "" {
		keyRegistrar = KEY_REGISTRAR_ADDRESS
	}

	crossChainRegistry = devnetCtx.EigenLayer.L1.CrossChainRegistry
	if crossChainRegistry == "" {
		crossChainRegistry = CROSS_CHAIN_REGISTRY_ADDRESS
	}
	bn254TableCalculator = devnetCtx.EigenLayer.L1.BN254TableCalculator
	if bn254TableCalculator == "" {
		bn254TableCalculator = BN254_TABLE_CALCULATOR_ADDRESS
	}
	releaseManager = devnetCtx.EigenLayer.L1.ReleaseManager
	if releaseManager == "" {
		releaseManager = RELEASE_MANAGER_ADDRESS
	}

	return allocationManager, delegationManager, strategyManager, keyRegistrar, crossChainRegistry, bn254TableCalculator, releaseManager
}
