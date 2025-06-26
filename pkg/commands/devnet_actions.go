package commands

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/crypto-libs/pkg/keystore"
	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

type DeployContractTransport struct {
	Name    string
	Address string
	ABI     string
}

type DeployContractJson struct {
	Name    string      `json:"name"`
	Address string      `json:"address"`
	ABI     interface{} `json:"abi"`
}

func StartDevnetAction(cCtx *cli.Context) error {
	// Check if docker is running, else try to start it
	if err := common.EnsureDockerIsRunning(cCtx); err != nil {

		if errors.Is(err, context.Canceled) {
			return err // propagate the cancellation directly
		}
		return cli.Exit(err.Error(), 1)
	}

	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)

	// Extract vars
	skipAvsRun := cCtx.Bool("skip-avs-run")
	skipDeployContracts := cCtx.Bool("skip-deploy-contracts")
	skipTransporter := cCtx.Bool("skip-transporter")
	useZeus := cCtx.Bool("use-zeus")

	// Migrate config
	configMigrated, err := migrateConfig(logger)
	if err != nil {
		logger.Error("config migration failed: %w", err)
	}
	if configMigrated > 0 {
		logger.Info("Config migration complete")
	}

	// Migrate contexts
	contextsMigrated, err := migrateContexts(logger)
	if err != nil {
		logger.Error("context migrations failed: %w", err)
	}
	if contextsMigrated > 0 {
		suffix := "s"
		if contextsMigrated == 1 {
			suffix = ""
		}
		logger.Info("%d context migration%s complete", contextsMigrated, suffix)
	}

	// Load config for devnet
	config, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return err
	}

	// Check for context
	yamlPath, rootNode, contextNode, err := common.LoadContext("devnet") // @TODO: use selected context name
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Fetch EigenLayer addresses using Zeus if requested
	if useZeus {
		logger.Info("Fetching EigenLayer core addresses from Zeus...")
		err = common.UpdateContextWithZeusAddresses(logger, contextNode, devnet.DEVNET_CONTEXT)
		if err != nil {
			logger.Warn("Failed to fetch addresses from Zeus: %v", err)
			logger.Info("Continuing with addresses from config...")
		} else {
			logger.Info("Successfully updated context with addresses from Zeus")

			// Write yaml back to project directory
			if err := common.WriteYAML(yamlPath, rootNode); err != nil {
				return fmt.Errorf("failed to save updated context: %v", err)
			}
		}
	}
	port := cCtx.Int("port")
	if !devnet.IsPortAvailable(port) {
		return fmt.Errorf("❌ Port %d is already in use. Please choose a different port using --port", port)
	}
	chainImage := devnet.GetDevnetChainImageOrDefault(config)
	chainArgs := devnet.GetDevnetChainArgsOrDefault(config)

	// Start timer
	startTime := time.Now()

	logger.Info("Starting devnet...\n")

	if cCtx.Bool("reset") {
		logger.Debug("Resetting devnet...")
	}
	if fork := cCtx.String("fork"); fork != "" {
		logger.Debug("Forking from chain: %s", fork)
	}
	if cCtx.Bool("headless") {
		logger.Debug("Running in headless mode")
	}

	// Docker-compose for anvil devnet
	composePath := devnet.WriteEmbeddedArtifacts()
	forkUrl, err := devnet.GetDevnetForkUrlDefault(config, devnet.L1)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	// Error if the forkUrl has not been modified
	if forkUrl == "" {
		return fmt.Errorf("fork-url not set; set fork-url in ./config/context/devnet.yaml or .env and consult README for guidance")
	}

	// Ensure fork URL uses appropriate Docker host for container environments
	dockerForkUrl := devnet.EnsureDockerHost(forkUrl)

	// Get the block_time from env/config
	blockTime, err := devnet.GetDevnetBlockTimeOrDefault(config, devnet.L1)
	if err != nil {
		blockTime = 12
	}

	// Get the chain_id from env/config
	chainId, err := devnet.GetDevnetChainIdOrDefault(config, devnet.L1, logger)
	if err != nil {
		chainId = common.DefaultAnvilChainId
	}

	// Append config defined details to chainArgs
	chainArgs = fmt.Sprintf("%s --chain-id %d", chainArgs, chainId)
	chainArgs = fmt.Sprintf("%s --block-time %d", chainArgs, blockTime)

	// Run docker compose up for anvil devnet
	cmd := exec.CommandContext(cCtx.Context, "docker", "compose", "-p", config.Config.Project.Name, "-f", composePath, "up", "-d")

	containerName := fmt.Sprintf("devkit-devnet-%s", config.Config.Project.Name)
	l1ChainConfig, found := config.Context[devnet.DEVNET_CONTEXT].Chains["l1"]
	if !found {
		return fmt.Errorf("failed to find a chain with name: l1 in devnet.yaml")
	}
	cmd.Env = append(os.Environ(),
		"FOUNDRY_IMAGE="+chainImage,
		"ANVIL_ARGS="+chainArgs,
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"FORK_RPC_URL="+dockerForkUrl,
		fmt.Sprintf("FORK_BLOCK_NUMBER=%d", l1ChainConfig.Fork.Block),
		"AVS_CONTAINER_NAME="+containerName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}

	// On cancel, always call down if skipAvsRun=false
	if !skipDeployContracts && !skipAvsRun {
		defer func() {
			logger.Info("Stopping containers")
			// clone cCtx but overwrite the context to Background
			cloned := *cCtx
			cloned.Context = context.Background()
			if err := StopDevnetAction(&cloned); err != nil {
				logger.Warn("automatic StopDevnetAction failed: %v", err)
			}
		}()
	}

	// Construct RPC url to pass to scripts
	rpcUrl := devnet.GetRPCURL(port)
	logger.Info("Waiting for devnet to be ready...")

	// Get chains node
	chainsNode := common.GetChildByKey(contextNode, "chains")
	if chainsNode == nil {
		return fmt.Errorf("missing 'chains' key in context")
	}

	// Update RPC URLs for both L1 and L2 chains
	for i := 0; i < len(chainsNode.Content); i += 2 {
		chainNode := chainsNode.Content[i+1]

		rpcUrlNode := common.GetChildByKey(chainNode, "rpc_url")
		if rpcUrlNode != nil {
			rpcUrlNode.Value = rpcUrl
		}
	}

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return err
	}

	// Sleep for 4 second to ensure the devnet is fully started
	time.Sleep(4 * time.Second)
	// Fund the wallets defined in config
	err = devnet.FundWalletsDevnet(config, rpcUrl)
	if err != nil {
		return err
	}

	// Fund stakers with strategy tokens
	if devnet.DEVNET_CONTEXT == "devnet" {
		logger.Info("Funding stakers with strategy tokens...")

		var tokenAddresses []string
		var tokenErr error
		tokenAddresses, tokenErr = devnet.GetUnderlyingTokenAddressesFromStrategies(config, rpcUrl, logger)
		if tokenErr != nil {
			logger.Warn("Failed to get underlying token addresses from strategies: %v", tokenErr)
			logger.Info("Continuing with devnet startup...")
		}

		if len(tokenAddresses) > 0 {
			err = devnet.FundStakersWithStrategyTokens(config, rpcUrl, tokenAddresses)
			if err != nil {
				logger.Warn("Failed to fund stakers with strategy tokens: %v", err)
				logger.Info("Continuing with devnet startup...")
			}
		} else {
			logger.Info("No tokens to fund stakers with, skipping token funding")
		}
	} else {
		logger.Info("Skipping token funding for non-devnet context")
	}

	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)
	logger.Info("\nDevnet started successfully in %s", elapsed)

	if err := WhitelistChainIdInCrossRegistryAction(cCtx, logger); err != nil {
		return fmt.Errorf("whitelisting chain id in cross registry failed: %w", err)
	}

	// Deploy the contracts after starting devnet unless skipped
	if !skipDeployContracts {
		if err := DeployContractsAction(cCtx); err != nil { // Assumes DeployContractsAction remains as is or is also refactored if needed
			return fmt.Errorf("deploy-contracts failed: %w", err)
		}

		// Sleep for 1 second to make sure new context values have been written
		time.Sleep(1 * time.Second)

		logger.Title("Registering AVS with EigenLayer...")

		if !cCtx.Bool("skip-setup") {
			if err := UpdateAVSMetadataAction(cCtx, logger); err != nil {
				return fmt.Errorf("updating AVS metadata failed: %w", err)
			}

			if err := SetAVSRegistrarAction(cCtx, logger); err != nil {
				return fmt.Errorf("setting AVS registrar failed: %w", err)
			}

			if err := CreateAVSOperatorSetsAction(cCtx, logger); err != nil {
				return fmt.Errorf("creating AVS operator sets failed: %w", err)
			}

			if err := ConfigureOpSetCurveTypeAction(cCtx, logger); err != nil {
				return fmt.Errorf("failed to configure OpSet in KeyRegistrar")
			}

			if err := CreateGenerationReservationAction(cCtx, logger); err != nil {
				return fmt.Errorf("failed to request op set generation reservation: %w", err)
			}

			if err := RegisterOperatorsToEigenLayerFromConfigAction(cCtx, logger); err != nil {
				return fmt.Errorf("registering operators failed: %w", err)
			}

			if err := RegisterKeyInKeyRegistrarAction(cCtx, logger); err != nil {
				return fmt.Errorf("registering key in key registrar failed: %w", err)
			}

			if err := DepositIntoStrategiesAction(cCtx, logger); err != nil {
				return fmt.Errorf("depositing into strategies failed: %w", err)
			}

			if err := DelegateToOperatorsAction(cCtx, logger); err != nil {
				return fmt.Errorf("delegating to operators failed: %w", err)
			}

			if err := SetAllocationDelayAction(cCtx, logger); err != nil {
				return fmt.Errorf("setting allocation delay failed: %w", err)
			}

			if err := ModifyAllocationsAction(cCtx, logger); err != nil {
				return fmt.Errorf("modifying allocations failed: %w", err)
			}

			if err := RegisterOperatorsToAvsFromConfigAction(cCtx, logger); err != nil {
				return fmt.Errorf("registering operators to AVS failed: %w", err)
			}
		} else {
			logger.Info("Skipping AVS setup steps...")
		}
	}

	// Run Transport against schedule - exit when AVSRun exits
	if !skipTransporter {
		// Post initial stake roots to L1
		if err := Transport(cCtx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("transport run failed: %w", err)
		}
		go func() {
			err := ScheduleTransport(cCtx, config.Context[devnet.DEVNET_CONTEXT].Transporter.Schedule)
			if err != nil {
				logger.Error("ScheduleTransport failed: %v", err)
			}
		}()
		// Keep scheduler alive
		if skipAvsRun {
			select {}
		}
	}

	// Start offchain AVS components after starting devnet and deploying contracts unless skipped
	if !skipDeployContracts && !skipAvsRun {
		if err := AVSRun(cCtx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("avs run failed: %w", err)
		}
	}

	return nil
}

func DeployContractsAction(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)
	// Check if docker is running, else try to start it
	err := common.EnsureDockerIsRunning(cCtx)
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}

	// Start timing execution runtime
	startTime := time.Now()

	// Run scriptPath from cwd
	const dir = ""
	const context = "devnet" // @TODO: use selected context name

	// Set path for .devkit scripts
	scriptsDir := filepath.Join(".devkit", "scripts")

	// List of scripts we want to call and curry context through
	scriptNames := []string{
		"deployContracts",
		"getOperatorSets",
		"getOperatorRegistrationMetadata",
	}

	// Check for context
	yamlPath, rootNode, contextNode, err := common.LoadContext("devnet") // @TODO: use selected context name
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Loop scripts with cloned context
	for _, name := range scriptNames {
		// Log the script name that's about to be executed
		logger.Info("Executing script: %s", name)
		// Clone context node and convert to map
		clonedCtxNode := common.CloneNode(contextNode)
		ctxInterface, err := common.NodeToInterface(clonedCtxNode)
		if err != nil {
			return fmt.Errorf("context decode failed: %w", err)
		}

		// Check context is a map
		ctxMap, ok := ctxInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cloned context is not a map")
		}

		// Parse the provided params
		inputJSON, err := json.Marshal(map[string]interface{}{"context": ctxMap})
		if err != nil {
			return fmt.Errorf("marshal context: %w", err)
		}

		// Set path in scriptsDir
		scriptPath := filepath.Join(scriptsDir, name)
		// Expect a JSON response which we will curry to the next call and later save to context
		outMap, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectJSONResponse, inputJSON)
		if err != nil {
			return fmt.Errorf("%s failed: %w", name, err)
		}

		// Convert to node for merge
		outNode, err := common.InterfaceToNode(outMap)
		if err != nil {
			return fmt.Errorf("%s output invalid: %w", name, err)
		}

		// Merge output into original context node
		common.DeepMerge(contextNode, outNode)
	}

	// Create output .json files for each of the deployed contracts
	contracts := common.GetChildByKey(contextNode, "deployed_contracts")
	if contracts == nil {
		return fmt.Errorf("deployed_contracts node not found")
	}
	var contractsList []DeployContractTransport
	if err := contracts.Decode(&contractsList); err != nil {
		return fmt.Errorf("decode deployed_contracts: %w", err)
	}
	// Empty log line to split these logs from the main body for easy identification
	logger.Title("Save contract artefacts")
	err = extractContractOutputs(cCtx, context, contractsList)
	if err != nil {
		return fmt.Errorf("failed to write contract artefacts: %w", err)
	}

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return err
	}

	// Measure how long we ran for
	elapsed := time.Since(startTime).Round(time.Second)
	logger.Info("\nDevnet contracts deployed successfully in %s", elapsed)
	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {
	// Get logger
	log := common.LoggerFromContext(cCtx.Context)

	// Read flags
	stopAllContainers := cCtx.Bool("all")

	// Should we stop all?
	if stopAllContainers {
		// Get all running containers
		cmd := exec.CommandContext(cCtx.Context, "docker", devnet.GetDockerPsDevnetArgs()...)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to list devnet containers: %w", err)
		}
		containerNames := strings.Split(strings.TrimSpace(string(output)), "\n")

		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			fmt.Printf("%s🚫 No devnet containers running.%s\n", devnet.Yellow, devnet.Reset)
			return nil
		}

		if cCtx.Bool("verbose") {
			log.Info("Attempting to stop devnet containers...")
		}

		for _, name := range containerNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			containerName := strings.Split(name, ": ")[0]

			devnet.StopAndRemoveContainer(cCtx, containerName)

		}

		return nil
	}

	projectName := cCtx.String("project.name")
	projectPort := cCtx.Int("port")

	// Check if any of the args are provided
	if !(projectName == "") || !(projectPort == 0) {
		if projectName != "" {
			container := fmt.Sprintf("devkit-devnet-%s", projectName)
			devnet.StopAndRemoveContainer(cCtx, container)
		} else {
			// project.name is empty, but port is provided
			// List all running Docker containers whose names include "devkit-devnet",
			// and format the output to show each container's name and its exposed ports.
			cmd := exec.CommandContext(cCtx.Context, "docker", devnet.GetDockerPsDevnetArgs()...)

			output, err := cmd.Output()
			if err != nil {
				log.Warn("Failed to list running devnet containers: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			containerFoundUsingthePort := false
			for _, line := range lines {
				parts := strings.Split(line, ": ")
				if len(parts) != 2 {
					continue
				}
				containerName := parts[0]
				port := parts[1]
				hostPort := extractHostPort(port)

				if hostPort == fmt.Sprintf("%d", projectPort) {
					// Derive project name from container name
					projectName := strings.TrimPrefix(containerName, "devkit-devnet-")
					devnet.StopAndRemoveContainer(cCtx, containerName)

					log.Info("Stopped devnet container running on port %d, project.name %s", projectPort, projectName)
					containerFoundUsingthePort = true
					break
				}
			}
			if !containerFoundUsingthePort {
				log.Info("No container found with port %d. Try %sdevkit avs devnet list%s to get a list of running devnet containers", projectPort, devnet.Cyan, devnet.Reset)
			}

		}
		return nil
	}

	if devnet.FileExistsInRoot(filepath.Join(common.DefaultConfigWithContextConfigPath, common.BaseConfig)) {
		// Load config
		config, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
		if err != nil {
			return err
		}

		container := fmt.Sprintf("devkit-devnet-%s", config.Config.Project.Name)

		devnet.StopAndRemoveContainer(cCtx, container)

	} else {
		log.Info("Run this command from the avs directory  or run %sdevkit avs devnet stop --help%s for available commands", devnet.Cyan, devnet.Reset)
	}

	return nil
}

func ListDevnetContainersAction(cCtx *cli.Context) error {
	cmd := exec.CommandContext(cCtx.Context, "docker", devnet.GetDockerPsDevnetArgs()...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list devnet containers: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("%s🚫 No devnet containers running.%s\n", devnet.Yellow, devnet.Reset)
		return nil
	}
	fmt.Printf("%s📦 Running Devnet Containers:%s\n\n", devnet.Blue, devnet.Reset)
	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		port := extractHostPort(parts[1])
		fmt.Printf("%s  -  %s%-25s%s %s→%s  %shttp://localhost:%s%s\n",
			devnet.Cyan, devnet.Reset,
			name,
			devnet.Reset,
			devnet.Green, devnet.Reset,
			devnet.Yellow, port, devnet.Reset,
		)
	}
	return nil
}

func UpdateAVSMetadataAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	uri := cCtx.String("uri")
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.DEVNET_CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	return contractCaller.UpdateAVSMetadata(cCtx.Context, avsAddr, uri)
}

func SetAVSRegistrarAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.DEVNET_CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	var registrarAddr ethcommon.Address
	logger.Info("Attempting to find AvsRegistrar in deployed contracts...")
	foundInDeployed := false
	for _, contract := range envCtx.DeployedContracts {
		if strings.Contains(strings.ToLower(contract.Name), "avsregistrar") {
			registrarAddr = ethcommon.HexToAddress(contract.Address)
			logger.Info("Found AvsRegistrar: '%s' at address %s", contract.Name, registrarAddr.Hex())
			foundInDeployed = true
			break
		}
	}
	if !foundInDeployed {
		return fmt.Errorf("AvsRegistrar contract not found in deployed contracts for context '%s'", devnet.DEVNET_CONTEXT)
	}

	return contractCaller.SetAVSRegistrar(cCtx.Context, avsAddr, registrarAddr)
}

func CreateAVSOperatorSetsAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.DEVNET_CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	if len(envCtx.OperatorSets) == 0 {
		logger.Info("No operator sets to create.")
		return nil
	}
	createSetParams := make([]allocationmanager.IAllocationManagerTypesCreateSetParams, len(envCtx.OperatorSets))
	for i, opSet := range envCtx.OperatorSets {
		strategies := make([]ethcommon.Address, len(opSet.Strategies))
		for j, strategy := range opSet.Strategies {
			strategies[j] = ethcommon.HexToAddress(strategy.StrategyAddress)
		}
		createSetParams[i] = allocationmanager.IAllocationManagerTypesCreateSetParams{
			OperatorSetId: uint32(opSet.OperatorSetID),
			Strategies:    strategies,
		}
	}

	return contractCaller.CreateOperatorSets(cCtx.Context, avsAddr, createSetParams)
}

func DelegateToOperatorsAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for delegate to operators: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	logger.Info("Delegating to operators...")

	for _, stakerSpec := range envCtx.Stakers {
		logger.Info("Delegating to operators for staker %s", stakerSpec.StakerAddress)
		if err := delegateToOperator(cCtx, stakerSpec, ethcommon.HexToAddress(stakerSpec.OperatorAddress), logger); err != nil {
			logger.Error("Failed to delegate to operators for staker %s: %v. Continuing...", stakerSpec.StakerAddress, err)
			continue
		}
	}
	logger.Info("Delegating to operators completed.")
	return nil
}

func DepositIntoStrategiesAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for deposit into strategies: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	logger.Info("Depositing into strategies...")
	for _, stakerSpec := range envCtx.Stakers {
		logger.Info("Depositing into strategies for staker %s", stakerSpec.StakerAddress)
		if err := depositIntoStrategy(cCtx, stakerSpec, logger); err != nil {
			logger.Error("Failed to deposit into strategies for staker %s: %v. Continuing...", stakerSpec.StakerAddress, err)
			continue
		}
	}
	logger.Info("Depositing into strategies completed.")
	return nil
}

func RegisterOperatorsToEigenLayerFromConfigAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for operator registration: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	logger.Info("Registering operators with EigenLayer...")
	if len(envCtx.OperatorRegistrations) == 0 {
		logger.Info("No operator registrations found in context, skipping operator registration.")
		return nil
	}

	for _, opReg := range envCtx.OperatorRegistrations {
		logger.Info("Processing registration for operator at address %s", opReg.Address)
		if err := registerOperatorEL(cCtx, opReg.Address, logger); err != nil {
			logger.Error("Failed to register operator %s with EigenLayer: %v. Continuing...", opReg.Address, err)
			continue
		}
	}
	logger.Info("Operators registration with EigenLayer completed.")
	return nil
}

func RegisterOperatorsToAvsFromConfigAction(cCtx *cli.Context, logger iface.Logger) error {

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for operator registration: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	logger.Info("Registering operators to AVS from config...")
	if len(envCtx.OperatorRegistrations) == 0 {
		logger.Info("No operator registrations found in context, skipping operator registration.")
		return nil
	}

	for _, opReg := range envCtx.OperatorRegistrations {
		logger.Info("Processing avs registration for operator at address %s", opReg.Address)
		if err := registerOperatorAVS(cCtx, logger, opReg.Address, uint32(opReg.OperatorSetID), opReg.Payload); err != nil {
			logger.Error("Failed to register operator %s for AVS: %v. Continuing...", opReg.Address, err)
			continue
		}
		logger.Info("Successfully registered operator %s for OperatorSetID %d", opReg.Address, opReg.OperatorSetID)
	}
	return nil
}

func FetchZeusAddressesAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)
	contextName := cCtx.String("context")

	// Check for context
	yamlPath, rootNode, contextNode, err := common.LoadContext("devnet") // @TODO: use selected context name
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Fetch addresses from Zeus
	logger.Info("Fetching EigenLayer core addresses from Zeus...")
	addresses, err := common.GetZeusAddresses(logger)
	if err != nil {
		return fmt.Errorf("failed to get addresses from Zeus for %s: %w", contextName, err)
	}

	// Print the fetched addresses
	payload := common.ZeusAddressData{
		AllocationManager: addresses.AllocationManager,
		DelegationManager: addresses.DelegationManager,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("found addresses (marshal failed): %w", err)
	}
	logger.Info("Found addresses: %s", b)

	// Update the context with the fetched addresses
	err = common.UpdateContextWithZeusAddresses(logger, contextNode, contextName)
	if err != nil {
		return fmt.Errorf("failed to update context (%s) with Zeus addresses: %w", contextName, err)
	}

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return fmt.Errorf("failed to save updated context: %v", err)
	}

	logger.Info("Successfully updated %s context with EigenLayer core addresses", contextName)
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

func registerOperatorEL(cCtx *cli.Context, operatorAddress string, logger iface.Logger) error {
	if operatorAddress == "" {
		return fmt.Errorf("operatorAddress parameter is required and cannot be empty")
	}

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	var operatorPrivateKey string
	for _, op := range envCtx.Operators {
		key, keyErr := crypto.HexToECDSA(strings.TrimPrefix(op.ECDSAKey, "0x"))
		if keyErr != nil {
			continue
		}
		if strings.EqualFold(crypto.PubkeyToAddress(key.PublicKey).Hex(), operatorAddress) {
			operatorPrivateKey = op.ECDSAKey
			break
		}
	}
	if operatorPrivateKey == "" {
		return fmt.Errorf("operator with address %s not found in config", operatorAddress)
	}
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		operatorPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	return contractCaller.RegisterAsOperator(cCtx.Context, ethcommon.HexToAddress(operatorAddress), 0, "test")
}

func registerOperatorAVS(cCtx *cli.Context, logger iface.Logger, operatorAddress string, operatorSetID uint32, payloadHex string) error {
	if operatorAddress == "" {
		return fmt.Errorf("operatorAddress parameter is required and cannot be empty")
	}
	if payloadHex == "" {
		return fmt.Errorf("payloadHex parameter is required and cannot be empty")
	}

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	var operatorPrivateKey string
	for _, op := range envCtx.Operators {
		key, keyErr := crypto.HexToECDSA(strings.TrimPrefix(op.ECDSAKey, "0x"))
		if keyErr != nil {
			continue
		}
		if strings.EqualFold(crypto.PubkeyToAddress(key.PublicKey).Hex(), operatorAddress) {
			operatorPrivateKey = op.ECDSAKey
			break
		}
	}
	if operatorPrivateKey == "" {
		return fmt.Errorf("operator with address %s not found in config", operatorAddress)
	}

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		operatorPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	payloadBytes, err := hex.DecodeString(payloadHex)
	if err != nil {
		return fmt.Errorf("failed to decode payload hex '%s': %w", payloadHex, err)
	}

	return contractCaller.RegisterForOperatorSets(
		cCtx.Context,
		ethcommon.HexToAddress(operatorAddress),
		ethcommon.HexToAddress(envCtx.Avs.Address),
		[]uint32{operatorSetID},
		payloadBytes,
	)
}

func depositIntoStrategy(cCtx *cli.Context, stakerSpec common.StakerSpec, logger iface.Logger) error {
	if stakerSpec.StakerAddress == "" {
		return fmt.Errorf("staker address parameter is required and cannot be empty")
	}

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)
	stakerPrivateKey := strings.TrimPrefix(stakerSpec.StakerECDSAKey, "0x")

	contractCaller, err := common.NewContractCaller(
		stakerPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	for _, deposit := range stakerSpec.Deposits {
		strategyAddress := deposit.StrategyAddress
		depositAmount := deposit.DepositAmount
		amount, err := common.ParseETHAmount(depositAmount)
		if err != nil {
			return fmt.Errorf("failed to parse deposit amount '%s': %w", depositAmount, err)
		}
		if err := contractCaller.DepositIntoStrategy(cCtx.Context, ethcommon.HexToAddress(strategyAddress), amount); err != nil {
			return fmt.Errorf("failed to deposit into strategy: %w", err)
		}
	}

	return nil
}

func delegateToOperator(cCtx *cli.Context, stakerSpec common.StakerSpec, operator ethcommon.Address, logger iface.Logger) error {

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)
	stakerPrivateKey := strings.TrimPrefix(stakerSpec.StakerECDSAKey, "0x")

	contractCaller, err := common.NewContractCaller(
		stakerPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
		ethcommon.HexToAddress(strategyManagerAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}
	// After depositing, delegate to the operator
	// Extract the private key of the operator we are delegating to in order to create an approval signature
	var operatorPrivateKey string
	for _, op := range envCtx.Operators {
		if strings.EqualFold(op.Address, operator.Hex()) {
			operatorPrivateKey = op.ECDSAKey
			break
		}
	}
	if operatorPrivateKey == "" {
		return fmt.Errorf("ECDSAkey not found for operator %s in operators in config.This means we cannot create an approval signature for this delegation", operator)
	}

	// expiry is 10 minutes from now
	expiry := big.NewInt(time.Now().Add(10 * time.Minute).Unix())

	// generate a random salt
	salt := [32]byte{}
	_, err = rand.Read(salt[:])
	if err != nil {
		return fmt.Errorf("failed to generate random salt: %w", err)
	}

	// Create the approval signature
	signature, err := contractCaller.CreateApprovalSignature(cCtx.Context, ethcommon.HexToAddress(stakerSpec.StakerAddress), operator, operator, operatorPrivateKey, salt, expiry)
	if err != nil {
		return fmt.Errorf("failed to create approval signature: %w", err)
	}

	if err := contractCaller.DelegateToOperator(cCtx.Context, operator, signature, salt); err != nil {
		return fmt.Errorf("failed to delegate to operator: %w", err)
	}
	return nil
}

func extractContractOutputs(cCtx *cli.Context, context string, contractsList []DeployContractTransport) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Push contract artefacts to ./contracts/outputs
	outDir := filepath.Join("contracts", "outputs", context)
	if err := os.MkdirAll(outDir, fs.ModePerm); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// For each contract extract details and produce json file in outputs/<context>/<contract.name>.json
	for _, contract := range contractsList {
		nameVal := contract.Name
		addressVal := contract.Address
		abiVal := contract.ABI

		// Read the ABI file
		raw, err := os.ReadFile(abiVal)
		if err != nil {
			return fmt.Errorf("read ABI for %s (%s) from %q: %w", nameVal, addressVal, abiVal, err)
		}

		// Temporary struct to pick only the "abi" field from the artifact
		var abi struct {
			ABI interface{} `json:"abi"`
		}
		if err := json.Unmarshal(raw, &abi); err != nil {
			return fmt.Errorf("unmarshal artifact JSON for %s (%s) failed: %w", nameVal, addressVal, err)
		}

		// Check if provided abi is valid
		if err := common.IsValidABI(abi.ABI); err != nil {
			return fmt.Errorf("ABI for %s (%s) is invalid: %v", nameVal, addressVal, err)
		}

		// Build the output struct
		out := DeployContractJson{
			Name:    nameVal,
			Address: addressVal,
			ABI:     abi.ABI,
		}

		// Marshal with indentation
		data, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal output for %s (%s): %w", nameVal, addressVal, err)
		}

		// Write to ./contracts/outputs/<context>/<name>.json
		outPath := filepath.Join(outDir, nameVal+".json")
		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			return fmt.Errorf("write output to %s (%s): %w", outPath, addressVal, err)
		}

		logger.Info("Written contract output: %s\n", outPath)
	}
	return nil
}

func migrateConfig(logger iface.Logger) (int, error) {
	// Set path for context yamls
	configDir := filepath.Join("config")
	configPath := filepath.Join(configDir, "config.yaml")

	// Migrate the config
	err := migration.MigrateYaml(logger, configPath, configs.LatestVersion, configs.MigrationChain)
	// Check for already upto date and ignore
	alreadyUptoDate := errors.Is(err, migration.ErrAlreadyUpToDate)

	// For any other error, migration has failed
	if err != nil && !alreadyUptoDate {
		return 0, fmt.Errorf("failed to migrate: %v", err)
	}

	// If config was migrated
	if !alreadyUptoDate {
		logger.Info("Migrated %s\n", configPath)

		return 1, nil
	}

	return 0, nil
}

func migrateContexts(logger iface.Logger) (int, error) {
	// Count the number of contexts we migrate
	contextsMigrated := 0

	// Set path for context yamls
	contextDir := filepath.Join("config", "contexts")

	// Read all contexts/*.yamls
	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return 0, fmt.Errorf("unable to read context directory: %v", err)
	}

	// Attempt to upgrade every entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		contextPath := filepath.Join(contextDir, e.Name())

		// Migrate the context
		err := migration.MigrateYaml(logger, contextPath, contexts.LatestVersion, contexts.MigrationChain)
		// Check for already upto date and ignore
		alreadyUptoDate := errors.Is(err, migration.ErrAlreadyUpToDate)

		// For every other error, migration failed
		if err != nil && !alreadyUptoDate {
			logger.Error("failed to migrate: %v", err)
			continue
		}

		// If context was migrated
		if !alreadyUptoDate {
			// Incr number of contextsMigrated
			contextsMigrated += 1

			// If migration succeeds
			logger.Info("Migrated %s\n", contextPath)
		}
	}

	return contextsMigrated, nil
}

func ModifyAllocationsAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for modify allocations: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	for _, op := range envCtx.Operators {
		logger.Info("Modifying allocations for operator %s", op.Address)
		if len(op.Allocations) == 0 {
			logger.Info("Operator %s has no allocations specified, skipping allocation modification", op.Address)
			continue
		}
		if err := modifyAllocations(cCtx, op.Address, op.ECDSAKey, logger); err != nil {
			logger.Debug("Failed to modify allocations for operator %s: %v. Continuing...", op.Address, err)
			continue
		}
	}
	logger.Info("Modifying allocations completed.")
	return nil
}

func modifyAllocations(cCtx *cli.Context, operatorAddress string, operatorPrivateKey string, logger iface.Logger) error {
	if operatorAddress == "" {
		return fmt.Errorf("modifyAllocations:operatorAddress parameter is required and cannot be empty")
	}

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	// Find the operator in config
	var targetOperator *common.OperatorSpec
	for i, op := range envCtx.Operators {
		if strings.EqualFold(op.Address, operatorAddress) {
			targetOperator = &envCtx.Operators[i]
			break
		}
	}
	if targetOperator == nil {
		return fmt.Errorf("operator with address %s not found in config", operatorAddress)
	}

	if len(targetOperator.Allocations) == 0 {
		logger.Info("Operator %s has no allocations specified, skipping allocation modification", operatorAddress)
		return nil
	}

	// Check deployed operator sets from context
	deployedOperatorSets := envCtx.OperatorSets
	if len(deployedOperatorSets) == 0 {
		logger.Warn("No deployed operator sets found in context, skipping allocation modification")
		return nil
	}

	// For each allocation in the operator config
	for _, allocation := range targetOperator.Allocations {
		strategyAddress := allocation.StrategyAddress

		// For each operator set allocation within this allocation
		for _, opSetAllocation := range allocation.OperatorSetAllocations {
			operatorSetID := opSetAllocation.OperatorSet
			allocationInWads := opSetAllocation.AllocationInWads

			// Check if this operator set ID exists in  deployed operator_sets and contains this strategy
			var strategyFound bool
			for _, deployedOpSet := range deployedOperatorSets {
				if fmt.Sprintf("%d", deployedOpSet.OperatorSetID) == operatorSetID {
					// Check if this operator set contains the strategy we're allocating to
					for _, strategy := range deployedOpSet.Strategies {
						if strings.EqualFold(strategy.StrategyAddress, strategyAddress) {
							strategyFound = true
							break
						}
					}
					break
				}
			}

			if !strategyFound {
				logger.Warn("Operator set %s with strategy %s not found in deployed operator sets, skipping allocation", operatorSetID, strategyAddress)
				continue
			}

			logger.Info("Modifying allocation for operator %s: operator_set=%s, strategy=%s, allocation=%s",
				operatorAddress, operatorSetID, strategyAddress, allocationInWads)

			allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

			contractCaller, err := common.NewContractCaller(
				operatorPrivateKey,
				big.NewInt(int64(l1Cfg.ChainID)),
				client,
				ethcommon.HexToAddress(allocationManagerAddr),
				ethcommon.HexToAddress(delegationManagerAddr),
				ethcommon.HexToAddress(strategyManagerAddr),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				logger,
			)
			if err != nil {
				return fmt.Errorf("failed to create contract caller: %w", err)
			}

			// Convert operatorSetID string to uint32
			operatorSetIDUint32, err := strconv.ParseUint(operatorSetID, 10, 32)
			if err != nil {
				return fmt.Errorf("failed to parse operator set ID '%s' to uint32: %w", operatorSetID, err)
			}

			// Build strategies array from matched operator set
			strategies := make([]ethcommon.Address, 1)
			strategies[0] = ethcommon.HexToAddress(strategyAddress)

			// Parse allocation amount to uint64
			allocationMagnitude, err := strconv.ParseUint(allocationInWads, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse allocation amount '%s' to uint64: %w", allocationInWads, err)
			}
			newMagnitudes := []uint64{allocationMagnitude}
			err = contractCaller.ModifyAllocations(
				cCtx.Context,
				ethcommon.HexToAddress(operatorAddress),
				operatorPrivateKey,
				strategies,
				newMagnitudes,
				ethcommon.HexToAddress(envCtx.Avs.Address),
				uint32(operatorSetIDUint32),
				logger,
			)
			if err != nil {
				return fmt.Errorf("failed to modify allocations: %w", err)
			}

			logger.Info("✅ Successfully modified allocation for operator %s (operator_set=%s, strategy=%s)",
				operatorAddress, operatorSetID, strategyAddress)
		}
	}

	return nil
}

func SetAllocationDelayAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for set allocation delay: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	// Instead of mining blocks(because it's infeasible for 126000 blocks(for mainnet) or 30 on holesky), use anvil_setStorageAt to bypass ALLOCATION_CONFIGURATION_DELAY
	// We need to manipulate the storage that tracks when allocation delays were set for each operator by modifying
	// the effectBlock field in the AllocationDelayInfo struct.
	logger.Info("Bypassing allocation configuration delay using anvil_setStorageAt...")

	allocationManagerAddr, _, _, _, _, _, _ := devnet.GetEigenLayerAddresses(cfg)
	currentBlock, err := client.BlockNumber(cCtx.Context)
	if err != nil {
		return fmt.Errorf("failed to get current block number: %w", err)
	}
	rpcClient := client.Client()
	// For each operator, modify their AllocationDelayInfo struct
	// Ref https://github.com/Layr-Labs/eigenlayer-contracts/blob/c08c9e849c27910f36f3ab746f3663a18838067f/src/contracts/core/AllocationManagerStorage.sol#L63
	for _, op := range envCtx.Operators {
		operatorAddr := ethcommon.HexToAddress(op.Address)

		// Calculate storage slot for _allocationDelayInfo mapping
		// For mapping(address => struct), storage slot = keccak256(abi.encode(key, slot))
		slotBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(slotBytes[24:], devnet.ALLOCATION_DELAY_INFO_SLOT)
		keyBytes := ethcommon.LeftPadBytes(operatorAddr.Bytes(), 32)

		encoded := append(keyBytes, slotBytes...)
		storageKey := ethcommon.BytesToHash(crypto.Keccak256(encoded))

		// Define struct fields
		var (
			delay        uint32 = 0                    // rightmost 4 bytes
			isSet        byte   = 0x00                 // 1 byte before delay
			pendingDelay uint32 = 0                    // 4 bytes before isSet
			effectBlock  uint32 = uint32(currentBlock) // 4 bytes before pendingDelay
		)

		// Create a 32-byte array (filled with zeros)
		structValue := make([]byte, 32)

		// Offset starts from the right
		offset := 32

		// Set delay (4 bytes)
		offset -= 4
		binary.BigEndian.PutUint32(structValue[offset:], delay)

		// Set isSet (1 byte)
		offset -= 1
		structValue[offset] = isSet

		// Set pendingDelay (4 bytes)
		offset -= 4
		binary.BigEndian.PutUint32(structValue[offset:], pendingDelay)

		// Set effectBlock (4 bytes)
		offset -= 4
		binary.BigEndian.PutUint32(structValue[offset:], effectBlock)

		var setStorageResult interface{}
		err = rpcClient.Call(&setStorageResult, "anvil_setStorageAt",
			allocationManagerAddr,
			storageKey.Hex(),
			hex.EncodeToString(structValue))
		if err != nil {
			logger.Warn("Failed to manipulate AllocationDelayInfo storage for operator %s: %v", op.Address, err)
		} else {
			logger.Info("Manipulated AllocationDelayInfo storage for operator %s effectBlock: %d", op.Address, effectBlock)
		}
	}

	logger.Info("Successfully bypassed allocation configuration delay")

	return nil
}

// ConfigureOpSetCurveType
func ConfigureOpSetCurveTypeAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for configure op set curve type: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	avsAddress := ethcommon.HexToAddress(envCtx.Avs.Address)
	avsPrivateKeyOrGivenPermissionByAvs := envCtx.Avs.AVSPrivateKey
	_, _, _, keyRegistrarAddr, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		avsPrivateKeyOrGivenPermissionByAvs,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(keyRegistrarAddr),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}
	// For each created operator set, configure the curve type
	for _, opSet := range envCtx.OperatorSets {
		logger.Info("Configuring curve type for operator set %s", opSet.OperatorSetID)

		// Configure the curve type
		err = contractCaller.ConfigureOpSetCurveType(cCtx.Context, avsAddress, uint32(opSet.OperatorSetID), uint8(devnet.CURVE_TYPE_KEY_REGISTRAR_BN254))
		if err != nil {
			return fmt.Errorf("failed to configure curve type for operator set %v: %w", opSet.OperatorSetID, err)
		}
		logger.Info("Successfully configured curve type for operator set %s", opSet.OperatorSetID)
	}

	return nil
}

func CreateGenerationReservationAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for request op set generation reservation: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	avsAddress := ethcommon.HexToAddress(envCtx.Avs.Address)
	avsPrivateKeyOrGivenPermissionByAvs := envCtx.Avs.AVSPrivateKey
	_, _, _, keyRegistrarAddr, crossChainRegistryAddr, bn254TableCalculatorAddr, _ := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		avsPrivateKeyOrGivenPermissionByAvs,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(keyRegistrarAddr),
		ethcommon.HexToAddress(crossChainRegistryAddr),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	for _, opSet := range envCtx.OperatorSets {
		err = contractCaller.CreateGenerationReservation(cCtx.Context, uint32(opSet.OperatorSetID), ethcommon.HexToAddress(bn254TableCalculatorAddr), avsAddress)
		if err != nil {
			return fmt.Errorf("failed to request op set generation reservation: %w", err)
		}

	}

	logger.Info("Successfully requested op set generation reservation")

	return nil
}

func WhitelistChainIdInCrossRegistryAction(cCtx *cli.Context, logger iface.Logger) error {
	// Skip this call if funding is disabled
	if os.Getenv("SKIP_DEVNET_FUNDING") == "true" {
		log.Println("🔧 Skipping WhitelistChainIdInCrossRegistry (test mode)")
		return nil
	}

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	crossChainRegistryAddr := ethcommon.HexToAddress(envCtx.EigenLayer.L1.CrossChainRegistry)
	operatorTableUpdater := ethcommon.HexToAddress(envCtx.EigenLayer.L2.OperatorTableUpdater)

	avsPrivateKeyOrGivenPermissionByAvs := envCtx.Avs.AVSPrivateKey

	contractCaller, err := common.NewContractCaller(
		avsPrivateKeyOrGivenPermissionByAvs,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		crossChainRegistryAddr,
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	err = contractCaller.WhitelistChainIdInCrossRegistry(cCtx.Context, operatorTableUpdater, uint64(l1Cfg.ChainID))
	if err != nil {
		return fmt.Errorf("failed to whitelist ChainId in CrossChainRegistry: %w", err)
	}

	logger.Info("Successfully whitelisted chain id in cross registry")
	return nil
}

func RegisterKeyInKeyRegistrarAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for register key in key registrar: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}

	avsAddress := ethcommon.HexToAddress(envCtx.Avs.Address)
	_, _, _, keyRegistrarAddr, _, _, _ := devnet.GetEigenLayerAddresses(cfg)

	for _, op := range envCtx.OperatorRegistrations {

		for _, operator := range envCtx.Operators {

			if op.Address == operator.Address {
				operatorPrivateKey := strings.Trim(operator.ECDSAKey, "0x")
				operatorAddress := ethcommon.HexToAddress(op.Address)
				contractCaller, err := common.NewContractCaller(
					operatorPrivateKey,
					big.NewInt(int64(l1Cfg.ChainID)),
					client,
					ethcommon.HexToAddress(""),
					ethcommon.HexToAddress(""),
					ethcommon.HexToAddress(""),
					ethcommon.HexToAddress(keyRegistrarAddr),
					ethcommon.HexToAddress(""),
					ethcommon.HexToAddress(""),
					logger,
				)
				if err != nil {
					return fmt.Errorf("failed to create contract caller: %w", err)
				}
				blskeystorePath := operator.BlsKeystorePath
				blskeystorePassword := operator.BlsKeystorePassword
				keystoreData, err := keystore.LoadKeystoreFile(blskeystorePath)

				if err != nil {
					return fmt.Errorf("failed to load the keystore file from given path %s error %w", blskeystorePath, err)
				}

				privateKey, err := keystoreData.GetBN254PrivateKey(blskeystorePassword)
				if err != nil {
					return fmt.Errorf("failed to extract the private key from the keystore file")

				}

				keyData, err := contractCaller.EncodeBN254KeyData(privateKey.Public())
				if err != nil {
					return fmt.Errorf("failed to encode key data: %w", err)
				}

				messageHash, err := contractCaller.GetOperatorRegistrationMessageHash(cCtx.Context, operatorAddress, avsAddress, uint32(op.OperatorSetID), keyData)
				if err != nil {
					return fmt.Errorf("failed to get operator registration message hash: %w", err)
				}

				signature, err := privateKey.SignSolidityCompatible(messageHash)
				if err != nil {
					return fmt.Errorf("failed to sign message hash: %w", err)
				}

				bn254Signature := bn254.Signature(*signature)

				err = contractCaller.RegisterKeyInKeyRegistrar(cCtx.Context, operatorAddress, avsAddress, uint32(op.OperatorSetID), keyData, bn254Signature)
				if err != nil {
					return fmt.Errorf("failed to register key in key registrar: %w", err)
				}
				logger.Info("Successfully registered key in key registrar for operator %s", operator.Address)
			}
		}

	}
	logger.Info("Successfully registered keys in key registrar")
	return nil
}
