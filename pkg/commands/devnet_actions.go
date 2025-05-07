package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/devnet"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	// "github.com/ethereum/go-ethereum/accounts/abi"
	// gethcommon "github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/ethclient"
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
	fork_url := devnet.GetDevnetForkUrlDefault(config)

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
		"MAINNET_RPC_URL="+fork_url,
		"STATE_PATH="+statePath,
		"AVS_CONTAINER_NAME="+containerName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(":x: Failed to start devnet: %w", err)
	}
	rpc_url := fmt.Sprintf("http://localhost:%d", port)

	// Sleep for 1 second to ensure the devnet is fully started
	time.Sleep(1 * time.Second)

	devnet.FundWalletsDevnet(config, rpc_url)
	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)

	make := exec.Command("make", "-f", common.DevkitMakefile, "deploy")
	make.Stdout = os.Stdout
	make.Stderr = os.Stderr
	if err := make.Run(); err != nil {
		return err
	}

	log.Printf("Devnet started successfully in %s", elapsed)

	// contractAddr := gethcommon.HexToAddress(devnet.CONTRACTS_REGISTRY)
	// client, err := ethclient.Dial(rpc_url)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to RPC: %v", err)
	// }

	// parsedABI, err := abi.JSON(strings.NewReader(devnet.CONTRACTS_REGISTRY_ABI))
	// if err != nil {
	// 	log.Fatalf("Failed to parse ABI: %v", err)
	// }

	// input, err := parsedABI.Pack("nameToAddress", "TaskAVSRegistrar")
	// if err != nil {
	// 	log.Fatalf("Failed to pack input: %v", err)
	// }

	// msg := ethereum.CallMsg{
	// 	To:   &contractAddr,
	// 	Data: input,
	// }

	// output, err := client.CallContract(context.Background(), msg, nil)
	// if err != nil {
	// 	log.Fatalf("Failed to call contract: %v", err)
	// }

	// var result gethcommon.Address
	// if err := parsedABI.UnpackIntoInterface(&result, "nameToAddress", output); err != nil {
	// 	log.Fatalf("Failed to unpack output: %v", err)
	// }

	// fmt.Printf("TaskAVSRegistrar address: %s\n", result.Hex())

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
