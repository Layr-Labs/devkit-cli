package devnet

import (
	"os"

	"devkit-cli/pkg/common"
)

// GetDevnetChainArgsOrDefault extracts and formats the chain arguments for devnet.
// Falls back to CHAIN_ARGS constant if value is empty.
func GetDevnetChainArgsOrDefault(cfg *common.ConfigWithContextConfig) string {
	args := []string{}
	// args := cfg.Env[DEVNET_ENV_KEY].ChainArgs  // TODO(nova) : Get chain args from config.yaml ?  For now using default
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

func GetDevnetForkUrlDefault(cfg *common.ConfigWithContextConfig) string {
	forkUrl := cfg.Context[CONTEXT].Fork.Url
	return forkUrl
}
