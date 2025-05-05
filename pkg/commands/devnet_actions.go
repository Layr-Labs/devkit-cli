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

	// Execute make run with Makefile.Devkit
	// hourglass_cmd := exec.Command("make", "-f", common.DevkitMakefile, "setup-submodules")
	// hourglass_cmd.Stdout = os.Stdout
	// hourglass_cmd.Stderr = os.Stderr
	// if err := hourglass_cmd.Run(); err != nil {
	// 	return err
	// }
	// log.Printf("Hourglass contracts submodules setup successful")

	hourglass_cmd := exec.Command("make", "-f", common.DevkitMakefile, "deploy-taskmailbox")
	hourglass_cmd.Stdout = os.Stdout
	hourglass_cmd.Stderr = os.Stderr
	if err := hourglass_cmd.Run(); err != nil {
		return err
	}
	log.Printf("TaskMailbox contract deployed!")

	hourglass_cmd = exec.Command("make", "-f", common.DevkitMakefile, "deploy-taskavsregistrar")
	hourglass_cmd.Stdout = os.Stdout
	hourglass_cmd.Stderr = os.Stderr
	if err := hourglass_cmd.Run(); err != nil {
		return err
	}

	log.Printf("TaskAvsRegistrar contract deployed!")

	hourglass_cmd = exec.Command("make", "-f", common.DevkitMakefile, "deploy-avsl2contracts")
	hourglass_cmd.Stdout = os.Stdout
	hourglass_cmd.Stderr = os.Stderr
	if err := hourglass_cmd.Run(); err != nil {
		return err
	}

	log.Printf("Avs L2 contracts deployed!")
	
	hourglass_cmd = exec.Command("make", "-f", common.DevkitMakefile, "setup-avsl1contracts")
	hourglass_cmd.Stdout = os.Stdout
	hourglass_cmd.Stderr = os.Stderr
	if err := hourglass_cmd.Run(); err != nil {
		return err
	}

	log.Printf("Avs L1 contracts setup success!")

	hourglass_cmd = exec.Command("make", "-f", common.DevkitMakefile, "create-operator-set")
	hourglass_cmd.Stdout = os.Stdout
	hourglass_cmd.Stderr = os.Stderr
	if err := hourglass_cmd.Run(); err != nil {
		return err
	}

	log.Printf("Created operator sets")

	
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
	containerName := fmt.Sprintf("devkit-devnet-%s", config.Project.Name)

	// Run docker compose down for anvil devnet
	stopCmd := exec.Command("docker", "compose", "-p", config.Project.Name, "-f", composePath, "down")

	stopCmd.Env = append(os.Environ(), // required for ${} to resolve in compose
		"FOUNDRY_IMAGE="+devnet.GetDevnetChainImageOrDefault(config),
		"ANVIL_ARGS="+devnet.GetDevnetChainArgsOrDefault(config),
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"STATE_PATH="+statePath,
		"AVS_CONTAINER_NAME="+containerName,
	)

	if err := stopCmd.Run(); err != nil {
		log.Fatalf("Failed to stop devnet containers: %v", err)
	}

	log.Printf("Devnet containers stopped and removed successfully.")
	return nil
}
