package devnet

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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

// StreamLogs attaches and streams logs from a container with the given role label.
func StreamLogsWithLabel(role string) error {
	// Find the container by label
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("label=devkit.role=%s", role), "--format", "{{.Names}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find container with label devkit.role=%s: %w", role, err)
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return fmt.Errorf("no running container found with label devkit.role=%s", role)
	}

	// Stream logs
	log.Printf("📺 Attaching to logs of container: %s", name)
	logCmd := exec.Command("docker", "logs", "-f", name)
	logCmd.Stdout = os.Stdout
	logCmd.Stderr = os.Stderr
	return logCmd.Run()
}
