package commands

import (
	"devkit-cli/pkg/common/config"
	"devkit-cli/pkg/common/devnet"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/urfave/cli/v2"
)

func StartDevnetAction(cCtx *cli.Context) error {
	// Load config
	config, err := config.LoadEigenConfig()
	if err != nil {
		return err
	}

	port := cCtx.Int("port")
	chain_image := devnet.GetDevnetChainImageOrDefault(config)
	chain_args := devnet.GetDevnetChainArgsOrDefault(config)

	startTime := time.Now() // <-- start timing

	if !cCtx.Bool("headless") {
		log.Printf("Starting devnet with eigenlayer contracts deployed ")
	}
	if cCtx.Bool("headless") {
		log.Printf("Running in headless mode")
	}

	if cCtx.Bool("verbose") {

		if cCtx.Bool("reset") {
			log.Printf("Resetting devnet...")
		}
		if fork := cCtx.String("fork"); fork != "" {
			log.Printf("Forking from chain: %s", fork)
		}

		devnet.LogDevnetEnv(config, cCtx.Int("port"))
	}
	// docker-compose for anvil devnet and anvil state.json
	composePath, statePath := devnet.WriteEmbeddedArtifacts()

	// Run docker compose up for anvil devnet
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")

	cmd.Env = append(os.Environ(),
		"FOUNDRY_IMAGE="+chain_image,
		"ANVIL_ARGS="+chain_args,
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"STATE_PATH="+statePath,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}
	rpc_url := fmt.Sprintf("http://localhost:%d", port)

	devnet.FundWalletsDevnet(config, rpc_url)
	elapsed := time.Since(startTime).Round(time.Second)
	log.Printf("Devnet started successfully in %s", elapsed)

	log.Printf("Starting stream for anvil logs")
	if !cCtx.Bool("headless") {
		log.Printf("📺 Streaming container logs to console")
		return devnet.StreamLogsWithLabel(DevkitRoleAnvil)
	} else {
		log.Printf("Headless mode : Streaming logs to file (unimplemented)")
	}

	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {
	// Load config
	config, err := config.LoadEigenConfig()
	if err != nil {
		return err
	}

	port := cCtx.Int("port")

	if cCtx.Bool("verbose") {
		log.Printf("Attempting to stop devnet containers...")
	}

	// Check if any devnet containers are running
	checkCmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
	output, err := checkCmd.Output()
	if err != nil {
		log.Fatalf("Failed to check running containers: %v", err)
	}

	if len(output) == 0 {
		log.Printf("No running devkit devnet containers found. Nothing to stop.")
		return nil
	}

	// docker-compose for anvil devnet and anvil state.json
	composePath, statePath := devnet.WriteEmbeddedArtifacts()

	// Run docker compose down for anvil devnet
	stopCmd := exec.Command("docker", "compose", "-f", composePath, "down")
	stopCmd.Env = append(os.Environ(), // required for ${} to resolve in compose
		"FOUNDRY_IMAGE="+devnet.GetDevnetChainImageOrDefault(config),
		"ANVIL_ARGS="+devnet.GetDevnetChainArgsOrDefault(config),
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"STATE_PATH="+statePath,
	)

	if err := stopCmd.Run(); err != nil {
		log.Fatalf("Failed to stop devnet containers: %v", err)
	}

	log.Printf("Devnet containers stopped and removed successfully.")
	return nil
}
