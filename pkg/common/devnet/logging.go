package devnet

import (
	"log"

	"devkit-cli/pkg/common"
)

func LogDevnetEnv(config *common.EigenConfig, port int) {
	log.Printf("Port: %d", port)

	chainImage := config.Env[DEVNET_ENV_KEY].ChainImage
	if chainImage == "" {
		log.Printf("⚠️  Chain image not provided in eigen.toml under [env.devnet]")
	} else {
		log.Printf("Chain Image: %s", chainImage)
	}
}

const (
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Reset  = "\033[0m"
)
