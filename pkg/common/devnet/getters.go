package devnet

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

// GetL1DevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to L1_CHAIN_ARGS constant if value is empty.
func GetL1DevnetChainArgsOrDefault(cfg *common.ConfigWithContextConfig) string {
	args := []string{} // TODO(nova) : Get chain args from config.yaml ?  For now using default
	if len(args) == 0 {
		return L1_CHAIN_ARGS
	}
	return " "
}

// GetL2DevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to L2_CHAIN_ARGS constant if value is empty.
func GetL2DevnetChainArgsOrDefault(cfg *common.ConfigWithContextConfig) string {
	args := []string{} // TODO(nova) : Get chain args from config.yaml ?  For now using default
	if len(args) == 0 {
		return L2_CHAIN_ARGS
	}
	return ""
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
		if chainName == "l1" {
			logger.Error("failed to get chainConfig for l1: %s", chainName)
			return DEFAULT_L1_ANVIL_CHAINID, fmt.Errorf("failed to get chainConfig for l1 : %s", chainName)
		} else if chainName == "l2" {
			logger.Error("failed to get chainConfig for l2: %s", chainName)
			return DEFAULT_L1_ANVIL_CHAINID, fmt.Errorf("failed to get chainConfig for l2 : %s", chainName)
		}
	}
	if chainConfig.ChainID == 0 {
		if chainName == "l1" {
			logger.Error("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
			return DEFAULT_L1_ANVIL_CHAINID, fmt.Errorf("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
		} else if chainName == "l2" {
			logger.Error("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
			return DEFAULT_L1_ANVIL_CHAINID, fmt.Errorf("chain_id not set for %s; set chain_id in ./config/context/devnet.yaml or .env", chainName)
		}
	}
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

// GetL1Port returns the L1 devnet port (default port + 0)
func GetL1Port(basePort int) int {
	return basePort
}

// GetL2Port returns the L2 devnet port (default port + 1)
func GetL2Port(basePort int) int {
	return basePort + 1
}

// GetL1RPCURL returns the L1 RPC URL for the given port
func GetL1RPCURL(basePort int) string {
	return fmt.Sprintf("http://localhost:%d", GetL1Port(basePort))
}

// GetL2RPCURL returns the L2 RPC URL for the given port
func GetL2RPCURL(basePort int) string {
	return fmt.Sprintf("http://localhost:%d", GetL2Port(basePort))
}
