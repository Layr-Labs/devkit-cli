package commands

import (
	"devkit-cli/pkg/common"
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
	config, err := common.LoadEigenConfig()
	if err != nil {
		return err
	}

	port := cCtx.Int("port")
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

	// Sleep for 1 second to ensure the devnet is fully started
	time.Sleep(1 * time.Second)

	devnet.FundWalletsDevnet(config, rpc_url)
	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)
	log.Printf("Devnet started successfully in %s", elapsed)

	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {
	// Load config
	config, err := common.LoadEigenConfig()
	if err != nil {
		return err
	}

	port := cCtx.Int("port")

	if common.IsVerboseEnabled(cCtx, config) {
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
