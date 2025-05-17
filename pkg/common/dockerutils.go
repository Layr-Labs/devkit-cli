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
func EnsureDockerIsRunning(ctx context.Context) error {

	if !isDockerInstalled() {
		return fmt.Errorf("docker is not installed. Please install Docker Desktop from https://www.docker.com/products/docker-desktop")
	}

	if isDockerRunning(ctx) {
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

	fmt.Print("⏳ Waiting for Docker to start")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timed out waiting for Docker to start")
		case <-ticker.C:
			if isDockerRunning(ctx) {
				fmt.Println("\n✅ Docker is now running.")
				return nil
			}
			fmt.Print(".")
		}
	}
}

// isDockerRunning attempts to ping the Docker daemon.
func isDockerRunning(ctx context.Context) bool {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = cli.Ping(ctx)
	return err == nil
}

// Check if docker is installed
func isDockerInstalled() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}
