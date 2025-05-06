package devnet

import (
	"strings"

	"devkit-cli/pkg/common"
)

// GetDevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to CHAIN_ARGS constant if value is empty.
func GetDevnetChainArgsOrDefault(cfg *common.EigenConfig) string {
	args := cfg.Env[DEVNET_ENV_KEY].ChainArgs
	if len(args) == 0 {
		return CHAIN_ARGS
	}
	return strings.Join(args, " ")
}

// GetDevnetForkUrlDefault extracts the FORK_URL from env
func GetDevnetForkUrlDefault(cfg *common.EigenConfig) string {
	forkUrl := cfg.Env[DEVNET_ENV_KEY].ForkUrl
	return forkUrl
}

// GetDevnetChainImageOrDefault returns the devnet chain image,
// falling back to FOUNDRY_IMAGE if not provided.
func GetDevnetChainImageOrDefault(cfg *common.EigenConfig) string {
	image := cfg.Env[DEVNET_ENV_KEY].ChainImage
	if image == "" {
		return FOUNDRY_IMAGE
	}
	return image
}
