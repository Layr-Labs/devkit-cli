package devnet

import (
	"log"

	"devkit-cli/pkg/common/config"
)

func LogDevnetEnv(config *config.EigenConfig, port int) {
	log.Printf("Port: %d", port)

	chainImage := config.Env[DEVNET_ENV_KEY].ChainImage
	if chainImage == "" {
		log.Printf("⚠️  Chain image not provided in eigen.toml under [env.devnet]")
	} else {
		log.Printf("Chain Image: %s", chainImage)
	}
}
