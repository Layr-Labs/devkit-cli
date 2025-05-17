package devnet

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"

	"github.com/urfave/cli/v2"
)

// IsPortAvailable checks if a TCP port is not already bound by another service.
func IsPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		// If dialing fails, port is likely available
		return true
	}
	_ = conn.Close()
	return false
}

// / Stops the container and removes it
func StopAndRemoveContainer(ctx *cli.Context, containerName string) {
	if err := exec.CommandContext(ctx.Context, "docker", "stop", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to stop container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Stopped container %s", containerName)
	}
	if err := exec.CommandContext(ctx.Context, "docker", "rm", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to remove container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Removed container %s", containerName)
	}
}

// GetDockerPsDevnetArgs returns the arguments needed to list all running
// devkit devnet Docker containers along with their exposed ports.
// It filters containers by name prefix ("devkit-devnet") and formats
// the output to show container name and port mappings in a readable form.
func GetDockerPsDevnetArgs() []string {
	return []string{
		"ps",
		"--filter", "name=devkit-devnet",
		"--format", "{{.Names}}: {{.Ports}}",
	}
}
