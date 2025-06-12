package commands

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/devkit-cli/pkg/hooks"
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
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
	config, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return err
	}

	// Set path for context yamls
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, "devnet.yaml")

	// Load YAML as *yaml.Node
	rootNode, err := common.LoadYAML(yamlPath)
	if err != nil {
		return err
	}

	// YAML is parsed into a DocumentNode:
	//   - rootNode.Content[0] is the top-level MappingNode
	//   - It contains the 'context' mapping we're interested in
	if len(rootNode.Content) == 0 {
		return fmt.Errorf("empty YAML root node")
	}

	// Check for context
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in ./config/contexts/devnet.yaml")
	}

	// Fetch EigenLayer addresses using Zeus if requested
	if useZeus {
		logger.Info("Fetching EigenLayer core addresses from Zeus...")
		err = common.UpdateContextWithZeusAddresses(logger, contextNode, devnet.CONTEXT)
		if err != nil {
			logger.Warn("Failed to fetch addresses from Zeus: %v", err)
			logger.Info("Continuing with addresses from config...")
		} else {
			logger.Info("Successfully updated context with addresses from Zeus")

			// Write yaml back to project directory
			if err := common.WriteYAML(yamlPath, rootNode); err != nil {
				return fmt.Errorf("Failed to save updated context: %v", err)
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
	chainId, err := devnet.GetDevnetChainIdOrDefault(config, devnet.L1)
	if err != nil {
		chainId = common.DefaultAnvilChainId
	}

	// Append config defined details to chainArgs
	chainArgs = fmt.Sprintf("%s --chain-id %d", chainArgs, chainId)
	chainArgs = fmt.Sprintf("%s --block-time %d", chainArgs, blockTime)

	// Run docker compose up for anvil devnet
	cmd := exec.CommandContext(cCtx.Context, "docker", "compose", "-p", config.Config.Project.Name, "-f", composePath, "up", "-d")

	containerName := fmt.Sprintf("devkit-devnet-%s", config.Config.Project.Name)
	l1ChainConfig, found := config.Context[devnet.CONTEXT].Chains["l1"]
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
	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)
	logger.Info("\nDevnet started successfully in %s", elapsed)

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
			logger.Info("AVS registered with EigenLayer successfully.")

			if err := RegisterOperatorsFromConfigAction(cCtx, logger); err != nil {
				return fmt.Errorf("registering operators failed: %w", err)
			}
		} else {
			logger.Info("Skipping AVS setup steps...")
		}
	}

	// Start offchain AVS components after starting devnet and deploying contracts unless skipped
	if !skipDeployContracts && !skipAvsRun {
		// Update project stage to running before starting AVS components (since AVSRun is long-running)
		logger.Info("Updating project stage to running...")
		if err := hooks.UpdateProjectStage(hooks.StageRunning, logger); err != nil {
			logger.Warn("Failed to update project stage: %v", err)
		}

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

	// Set path for context yaml
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, fmt.Sprintf("%s.%s", context, "yaml"))

	// Load YAML as *yaml.Node
	rootNode, err := common.LoadYAML(yamlPath)
	if err != nil {
		return err
	}

	// List of scripts we want to call and curry context through
	scriptNames := []string{
		"deployContracts",
		"getOperatorSets",
		"getOperatorRegistrationMetadata",
	}

	// YAML is parsed into a DocumentNode:
	//   - rootNode.Content[0] is the top-level MappingNode
	//   - It contains the 'context' mapping we're interested in
	if len(rootNode.Content) == 0 {
		return fmt.Errorf("empty YAML root node")
	}

	// Check for context
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in ./config/contexts/%s.yaml", context)
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
		config, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
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
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	uri := cCtx.String("uri")
	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()

	allocationManagerAddr := ethcommon.HexToAddress(devnet.ALLOCATION_MANAGER_ADDRESS)
	delegationManagerAddr := ethcommon.HexToAddress(devnet.DELEGATION_MANAGER_ADDRESS)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		allocationManagerAddr,
		delegationManagerAddr,
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	return contractCaller.UpdateAVSMetadata(cCtx.Context, avsAddr, uri)
}

func SetAVSRegistrarAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()

	allocationManagerAddr := ethcommon.HexToAddress(devnet.ALLOCATION_MANAGER_ADDRESS)
	delegationManagerAddr := ethcommon.HexToAddress(devnet.DELEGATION_MANAGER_ADDRESS)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		allocationManagerAddr,
		delegationManagerAddr,
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
		return fmt.Errorf("AvsRegistrar contract not found in deployed contracts for context '%s'", devnet.CONTEXT)
	}

	return contractCaller.SetAVSRegistrar(cCtx.Context, avsAddr, registrarAddr)
}

func CreateAVSOperatorSetsAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}
	l1ChainCfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s'", devnet.L1, devnet.CONTEXT)
	}
	client, err := ethclient.Dial(l1ChainCfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC at %s: %w", l1ChainCfg.RPCURL, err)
	}
	defer client.Close()

	allocationManagerAddr := ethcommon.HexToAddress(devnet.ALLOCATION_MANAGER_ADDRESS)
	delegationManagerAddr := ethcommon.HexToAddress(devnet.DELEGATION_MANAGER_ADDRESS)

	contractCaller, err := common.NewContractCaller(
		envCtx.Avs.AVSPrivateKey,
		big.NewInt(int64(l1ChainCfg.ChainID)),
		client,
		allocationManagerAddr,
		delegationManagerAddr,
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

func RegisterOperatorsFromConfigAction(cCtx *cli.Context, logger iface.Logger) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for operator registration: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
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
		if err := registerOperatorAVS(cCtx, logger, opReg.Address, uint32(opReg.OperatorSetID), opReg.Payload); err != nil {
			logger.Error("Failed to register operator %s for AVS: %v. Continuing...", opReg.Address, err)
			continue
		}
		logger.Info("Successfully registered operator %s for OperatorSetID %d", opReg.Address, opReg.OperatorSetID)
	}
	logger.Info("Operator registration with EigenLayer completed.")
	return nil
}

func FetchZeusAddressesAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)
	contextName := cCtx.String("context")

	// Set path for context yaml
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, fmt.Sprintf("%s.%s", contextName, "yaml"))

	// Load YAML as *yaml.Node
	rootNode, err := common.LoadYAML(yamlPath)
	if err != nil {
		return err
	}
	// Check for context
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in ./config/contexts/%s.yaml", contextName)
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
		return fmt.Errorf("Found addresses (marshal failed): %w", err)
	}
	logger.Info("Found addresses: %s", b)

	// Update the context with the fetched addresses
	err = common.UpdateContextWithZeusAddresses(logger, contextNode, contextName)
	if err != nil {
		return fmt.Errorf("failed to update context (%s) with Zeus addresses: %w", contextName, err)
	}

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return fmt.Errorf("Failed to save updated context: %v", err)
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

	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}
	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.CONTEXT)
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

	allocationManagerAddr := ethcommon.HexToAddress(devnet.ALLOCATION_MANAGER_ADDRESS)
	delegationManagerAddr := ethcommon.HexToAddress(devnet.DELEGATION_MANAGER_ADDRESS)

	contractCaller, err := common.NewContractCaller(
		operatorPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		allocationManagerAddr,
		delegationManagerAddr,
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

	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}
	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.CONTEXT)
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

	allocationManagerAddr, delegationManagerAddr := devnet.GetEigenLayerAddresses(cfg)

	contractCaller, err := common.NewContractCaller(
		operatorPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(allocationManagerAddr),
		ethcommon.HexToAddress(delegationManagerAddr),
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
