package devnet

import (
	"fmt"
	"log"
	"net"
	"os/exec"
	"time"
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

func StopAndRemoveContainer(containerName string) {
	if err := exec.Command("docker", "stop", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to stop container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Stopped container %s", containerName)
	}
	if err := exec.Command("docker", "rm", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to remove container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Removed container %s", containerName)
	}
}

func GetDockerPsDevnetArgs() []string {
	return []string{
		"ps",
		"--filter", "name=devkit-devnet",
		"--format", "{{.Names}}: {{.Ports}}",
	}
}
