package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/devnet"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const (
	blue   = "\033[34m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	reset  = "\033[0m"
)

func StartDevnetAction(cCtx *cli.Context) error {
	// Load config
	config, err := common.LoadEigenConfig()
	if err != nil {
		return err
	}

	port := cCtx.Int("port")
	if !devnet.IsPortAvailable(port) {
		log.Printf("is_port_available %d, %t", port, false)
		return fmt.Errorf("❌ Port %d is already in use. Please choose a different port using --port", port)
	}
	chain_image := devnet.GetDevnetChainImageOrDefault(config)
	chain_args := devnet.GetDevnetChainArgsOrDefault(config)

	startTime := time.Now() // <-- start timing
	// if user gives , say, log = "DEBUG" Or "Debug", we normalize it to lowercase
	if common.IsVerboseEnabled(cCtx, config) {
		log.Printf("Starting devnet... ")

		if cCtx.Bool("reset") {
			log.Printf("Resetting devnet...")
		}
		if fork := cCtx.String("fork"); fork != "" {
			log.Printf("Forking from chain: %s", fork)
		}
		if cCtx.Bool("headless") {
			log.Printf("Running in headless mode")
		}
		devnet.LogDevnetEnv(config, cCtx.Int("port"))
	}
	// docker-compose for anvil devnet and anvil state.json
	composePath, statePath := devnet.WriteEmbeddedArtifacts()

	// Run docker compose up for anvil devnet
	cmd := exec.Command("docker", "compose", "-p", config.Project.Name, "-f", composePath, "up", "-d")

	containerName := fmt.Sprintf("devkit-devnet-%s", config.Project.Name)
	cmd.Env = append(os.Environ(),
		"FOUNDRY_IMAGE="+chain_image,
		"ANVIL_ARGS="+chain_args,
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"STATE_PATH="+statePath,
		"AVS_CONTAINER_NAME="+containerName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}
	rpc_url := fmt.Sprintf("http://localhost:%d", port)

	devnet.FundWalletsDevnet(config, rpc_url)
	elapsed := time.Since(startTime).Round(time.Second)
	log.Printf("Devnet started successfully in %s", elapsed)

	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {

	stopAllContainers := cCtx.Bool("all")
	if stopAllContainers {

		cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}: {{.Ports}}")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list devnet containers: %w", err)
		}
		containerNames := strings.Split(strings.TrimSpace(string(output)), "\n")

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			fmt.Printf("%s🚫 No devnet containers running.%s\n", yellow, reset)
			return nil
		}

		log.Printf("Stopping all devnet containers...")

		for _, name := range containerNames {
			log.Printf("containerName : %s", name)
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			// docker-compose for anvil devnet and anvil state.json
			parts := strings.Split(name, ": ")

			log.Printf("project name to stop : %s", parts[0])

			exec.Command("docker", "stop", parts[0]).Run()
			exec.Command("docker", "rm", parts[0]).Run()

			log.Printf("Devnet containers stopped and removed successfully.")

		}

		return nil
	}

	projectName := cCtx.String("project.name")
	projectPort := cCtx.Int("port")

	// Check if any of the args are provided
	if !(projectName == "") || !(projectPort == 0) {

		if projectName != "" {
			container := fmt.Sprintf("devkit-devnet-%s", projectName)
			if err := exec.Command("docker", "stop", container).Run(); err != nil {
				log.Printf("⚠️ Failed to stop container %s: %v", container, err)
			} else {
				log.Printf("✅ Stopped container %s", container)
			}
			if err := exec.Command("docker", "rm", container).Run(); err != nil {
				log.Printf("⚠️ Failed to remove container %s: %v", container, err)
			} else {
				log.Printf("✅ Removed container %s", container)
			}

		} else {
			// project.name is empty, but port is provided
			// Find which container is running on that port
			cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}: {{.Ports}}")
			output, err := cmd.Output()
			if err != nil {
				log.Fatalf("Failed to list running devnet containers: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			for _, line := range lines {
				parts := strings.Split(line, ": ")
				if len(parts) != 2 {
					continue
				}
				containerName := parts[0]
				hostPort := extractHostPort(parts[1])

				if hostPort == fmt.Sprintf("%d", projectPort) {
					// Derive project name from container name
					projectNameFromContainer := strings.TrimPrefix(containerName, "devkit-devnet-")

					exec.Command("docker", "stop", projectNameFromContainer).Run()
					exec.Command("docker", "rm", projectNameFromContainer).Run()

					log.Printf("Stopped devnet container running on port %d (%s)", projectPort, containerName)
					break
				}
			}
		}
		return nil
	}

	if devnet.FileExistsInRoot("eigen.toml") {
		// Load config
		config, err := common.LoadEigenConfig()
		if err != nil {
			return err
		}

		container := fmt.Sprintf("devkit-devnet-%s", config.Project.Name)

		if err := exec.Command("docker", "stop", container).Run(); err != nil {
			log.Printf("⚠️ Failed to stop container %s: %v", container, err)
		} else {
			log.Printf("✅ Stopped container %s", container)
		}
		if err := exec.Command("docker", "rm", container).Run(); err != nil {
			log.Printf("⚠️ Failed to remove container %s: %v", container, err)
		} else {
			log.Printf("✅ Removed container %s", container)
		}

	} else {
		log.Printf("Run this command from the avs directory  or run devkit avs devnet stop --help for available commands")
	}

	return nil
}

func ListDevnetContainersAction(cCtx *cli.Context) error {
	cmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}: {{.Ports}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list devnet containers: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("%s🚫 No devnet containers running.%s\n", yellow, reset)
		return nil
	}

	fmt.Printf("%s📦 Running Devnet Containers:%s\n\n", blue, reset)
	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		port := extractHostPort(parts[1])
		fmt.Printf("%s  -  %s%-25s%s %s→%s  %shttp://localhost:%s%s\n",
			cyan, reset,
			name,
			reset,
			green, reset,
			yellow, port, reset,
		)
	}

	return nil
}

func extractHostPort(portStr string) string {
	if strings.Contains(portStr, "->") {
		beforeArrow := strings.Split(portStr, "->")[0]
		hostPort := strings.Split(beforeArrow, ":")
		return hostPort[len(hostPort)-1]
	}
	return portStr
}
