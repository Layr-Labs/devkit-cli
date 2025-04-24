package devnet

import (
	"log"
	"os"
	"os/exec"

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

func StreamLogs(containerName string) error {
	cmd := exec.Command("docker", "logs", "-f", containerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() // blocks until logs stop
}
