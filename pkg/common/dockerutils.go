package common

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/docker/docker/client"
)

// EnsureDockerIsRunning checks if Docker is running and attempts to launch Docker Desktop if not.
func EnsureDockerIsRunning() error {

	if !isDockerInstalled() {
		return fmt.Errorf("docker is not installed. Please install Docker Desktop from https://www.docker.com/products/docker-desktop")
	}

	if isDockerRunning() {
		return nil
	}

	fmt.Println(" Docker is installed but not running. Attempting to start Docker Desktop...")

	switch runtime.GOOS {
	case "darwin":
		// Mac
		err := exec.Command("open", "-a", "Docker").Start()
		if err != nil {
			return fmt.Errorf("failed to launch Docker Desktop: %w", err)
		}
	case "windows":
		// Windows
		err := exec.Command("powershell", "Start-Process", "Docker Desktop").Start()
		if err != nil {
			return fmt.Errorf("failed to launch Docker Desktop: %w", err)
		}
	default:
		return fmt.Errorf("unsupported OS for automatic Docker launch! please start Docker Desktop manually")
	}

	// Wait for Docker to come online
	fmt.Print("⏳ Waiting for Docker to start...")
	for i := 0; i < 20; i++ { // up to 10 seconds
		if isDockerRunning() {
			fmt.Println("\n✅ Docker is now running.")
			return nil
		}
		fmt.Print(".")
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for Docker to start. Please start Docker manually")
}

// isDockerRunning attempts to ping the Docker daemon.
func isDockerRunning() bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	return err == nil
}

// Check if docker is installed
func isDockerInstalled() bool {
	cmd := exec.Command("docker", "version", "--format", "'{{.Client.Version}}'")
	err := cmd.Run()
	return err == nil
}
