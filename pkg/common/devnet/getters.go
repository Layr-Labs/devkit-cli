package devnet

import (
	"devkit-cli/pkg/common"
	"strings"
)

// GetDevnetChainArgs extracts and formats the chain arguments for devnet.
func GetDevnetChainArgs(cfg *common.EigenConfig) string {
	args := cfg.Env[DEVNET_ENV_KEY].ChainArgs
	return strings.Join(args, " ")
}

// GetDevnetChainImage returns the devnet chain image.
func GetDevnetChainImage(cfg *common.EigenConfig) string {
	return cfg.Env[DEVNET_ENV_KEY].ChainImage
}
