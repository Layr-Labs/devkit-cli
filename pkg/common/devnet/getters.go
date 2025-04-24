package devnet

import (
	"strings"

	"devkit-cli/pkg/common/config"
)

// GetDevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to CHAIN_ARGS constant if value is empty.
func GetDevnetChainArgsOrDefault(cfg *config.EigenConfig) string {
	args := cfg.Env[DEVNET_ENV_KEY].ChainArgs
	if len(args) == 0 {
		return CHAIN_ARGS
	}
	return strings.Join(args, " ")
}

// GetDevnetChainImageOrDefault returns the devnet chain image,
// falling back to FOUNDRY_IMAGE if not provided.
func GetDevnetChainImageOrDefault(cfg *config.EigenConfig) string {
	image := cfg.Env[DEVNET_ENV_KEY].ChainImage
	if image == "" {
		return FOUNDRY_IMAGE
	}
	return image
}
