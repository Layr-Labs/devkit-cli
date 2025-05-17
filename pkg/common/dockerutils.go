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
	log, _ := GetLogger()
	if !isDockerInstalled() {
		return fmt.Errorf("docker is not installed. Please install Docker Desktop from https://www.docker.com/products/docker-desktop")
	}

	if isDockerRunning(ctx) {
		return nil
	}

	log.Info(" Docker is installed but not running. Attempting to start Docker Desktop...")

	switch runtime.GOOS {
	case "darwin":
		err := exec.Command("open", "-a", "Docker").Start()
		if err != nil {
			return fmt.Errorf("failed to launch Docker Desktop: %w", err)
		}
	case "windows":
		err := exec.Command("powershell", "Start-Process", "Docker Desktop").Start()
		if err != nil {
			return fmt.Errorf("failed to launch Docker Desktop: %w", err)
		}
	default:
		return fmt.Errorf("unsupported OS for automatic Docker launch! please start Docker manually")
	}

	log.Info("⏳ Waiting for Docker to start")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	timeout := time.After(10 * time.Second)

	var lastErr error

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timed out waiting for Docker to start after %s: error: %v",
				time.Since(start).Round(time.Millisecond), lastErr)
		case <-ticker.C:
			if err := tryDockerPing(ctx); err == nil {
				log.Info("\n✅ Docker is now running.")
				return nil
			} else {
				lastErr = err
			}
			fmt.Print(".")
		}
	}
}

// helper that returns error instead of bool
func tryDockerPing(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	defer cli.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err = cli.Ping(pingCtx)
	return err
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
