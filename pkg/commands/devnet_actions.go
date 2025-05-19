package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/devnet"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"

	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
)

func StartDevnetAction(cCtx *cli.Context) error {
	log, _ := common.GetLogger()

	// Extract vars
	skipAvsRun := cCtx.Bool("skip-avs-run")
	skipDeployContracts := cCtx.Bool("skip-deploy-contracts")

	// Load config for devnet
	config, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return err
	}
	port := cCtx.Int("port")
	if !devnet.IsPortAvailable(port) {
		return fmt.Errorf("❌ Port %d is already in use. Please choose a different port using --port", port)
	}
	chainImage := devnet.GetDevnetChainImageOrDefault(config)
	chainArgs := devnet.GetDevnetChainArgsOrDefault(config)

	// Start timer
	startTime := time.Now()

	// If user gives, say, log = "DEBUG" Or "Debug", we normalize it to lowercase
	if common.IsVerboseEnabled(cCtx, config) {
		log.Info("Starting devnet...\n")

		if cCtx.Bool("reset") {
			log.Info("Resetting devnet...")
		}
		if fork := cCtx.String("fork"); fork != "" {
			log.Info("Forking from chain: %s", fork)
		}
		if cCtx.Bool("headless") {
			log.Info("Running in headless mode")
		}
	}

	// Docker-compose for anvil devnet
	composePath := devnet.WriteEmbeddedArtifacts()
	forkURL, err := devnet.GetDevnetForkUrlDefault(cfg, devnet.L1)
	if err != nil {
		return fmt.Errorf("getting fork URL: %w", err)
	}

	envCtxForStart, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration for StartDevnetAction", devnet.CONTEXT)
	}
	l1ChainConfigForStart, ok := envCtxForStart.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("L1 chain configuration ('%s') not found in context '%s' for StartDevnetAction", devnet.L1, devnet.CONTEXT)
	}

	dockerCmd := exec.CommandContext(cCtx.Context, "docker", "compose", "-p", cfg.Config.Project.Name, "-f", composePath, "up", "-d")
	containerName := fmt.Sprintf("devkit-devnet-%s", cfg.Config.Project.Name)
	dockerCmd.Env = append(os.Environ(),
		"FOUNDRY_IMAGE="+chainImage,
		"ANVIL_ARGS="+chainArgs,
		fmt.Sprintf("DEVNET_PORT=%d", port),
		"FORK_RPC_URL="+forkURL,
		fmt.Sprintf("FORK_BLOCK_NUMBER=%d", l1ChainConfigForStart.Fork.Block),
		"AVS_CONTAINER_NAME="+containerName,
	)
	if err := dockerCmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet containers: %w", err)
	}
	rpcURL := fmt.Sprintf("http://localhost:%d", port)
	log.Info("Waiting for devnet to be ready...")
	time.Sleep(4 * time.Second)

	// Fund the wallets defined in config
	devnet.FundWalletsDevnet(config, rpcUrl)
	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)
	log.Info("\nDevnet started successfully in %s", elapsed)

	// Deploy the contracts after starting devnet unless skipped
	if !skipDeployContracts {
		if err := DeployContractsAction(cCtx); err != nil { // Assumes DeployContractsAction remains as is or is also refactored if needed
			return fmt.Errorf("deploy-contracts failed: %w", err)
		}

		log.Info("Registering AVS with EigenLayer...")

		if err := UpdateAVSMetadataAction(cCtx); err != nil {
			return fmt.Errorf("updating AVS metadata failed: %w", err)
		}
		if err := SetAVSRegistrarAction(cCtx); err != nil {
			return fmt.Errorf("setting AVS registrar failed: %w", err)
		}
		if err := CreateAVSOperatorSetsAction(cCtx); err != nil {
			return fmt.Errorf("creating AVS operator sets failed: %w", err)
		}
		log.Info("AVS registered with EigenLayer successfully.")

		if err := registerOperatorsFromConfig(cCtx, cfg); err != nil {
			return fmt.Errorf("registering operators failed: %w", err)
		}
	}

	// Start offchain AVS components after starting devnet and deploying contracts unless skipped
	if !skipDeployContracts && !skipAvsRun {
		if err := AVSRun(cCtx); err != nil {
			return fmt.Errorf("avs run failed: %w", err)
		}
	}

	return nil
}

func DeployContractsAction(cCtx *cli.Context) error {
	log, _ := common.GetLogger()
	startTime := time.Now()

	// Run scriptPath from cwd
	const dir = ""

	// Set path for .devkit scripts
	scriptsDir := filepath.Join(".devkit", "scripts")
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, "devnet.yaml") // @TODO: use selected context name
	rootNode, err := common.LoadYAML(yamlPath)
	if err != nil {
		return err
	}
	scriptNames := []string{"deployContracts", "getOperatorSets", "getOperatorRegistrationMetadata"}
	if len(rootNode.Content) == 0 {
		return fmt.Errorf("empty YAML root node")
	}
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in ./config/contexts/devnet.yaml")
	}
	for _, name := range scriptNames {
		clonedCtxNode := common.CloneNode(contextNode)
		ctxInterface, err := common.NodeToInterface(clonedCtxNode)
		if err != nil {
			return fmt.Errorf("context decode failed: %w", err)
		}
		ctxMap, ok := ctxInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cloned context is not a map")
		}
		inputJSON, err := json.Marshal(map[string]interface{}{"context": ctxMap})
		if err != nil {
			return fmt.Errorf("marshal context: %w", err)
		}

		// Set path in scriptsDir
		scriptPath := filepath.Join(scriptsDir, name)
		// Expect a JSON response which we will curry to the next call and later save to context
		outMap, err := common.CallTemplateScript(cCtx.Context, dir, scriptPath, common.ExpectJSONResponse, inputJSON)
		if err != nil {
			return fmt.Errorf("%s failed: %w", name, err)
		}
		outNode, err := common.InterfaceToNode(outMap)
		if err != nil {
			return fmt.Errorf("%s output invalid: %w", name, err)
		}
		common.DeepMerge(contextNode, outNode)
	}
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return err
	}
	log.Info("Devnet contracts deployed successfully in %s", time.Since(startTime).Round(time.Second))
	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {
	log, _ := common.GetLogger()
	stopAllContainers := cCtx.Bool("all")
	if stopAllContainers {
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
	if !(projectName == "") || !(projectPort == 0) {
		if projectName != "" {
			container := fmt.Sprintf("devkit-devnet-%s", projectName)
			devnet.StopAndRemoveContainer(cCtx, container)
		} else {
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
					projectNameFromContainer := strings.TrimPrefix(containerName, "devkit-devnet-")
					devnet.StopAndRemoveContainer(cCtx, containerName)
					log.Info("Stopped devnet container running on port %d, project.name %s", projectPort, projectNameFromContainer)
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

func extractHostPort(portStr string) string {
	if strings.Contains(portStr, "->") {
		beforeArrow := strings.Split(portStr, "->")[0]
		hostPort := strings.Split(beforeArrow, ":")
		return hostPort[len(hostPort)-1]
	}
	return portStr
}

// UpdateAVSMetadataAction handles the CLI command for updating AVS metadata.
func UpdateAVSMetadataAction(cCtx *cli.Context) error {
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
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	return contractCaller.UpdateAVSMetadata(cCtx.Context, avsAddr, uri)
}

// SetAVSRegistrarAction handles the CLI command for setting the AVS registrar.
func SetAVSRegistrarAction(cCtx *cli.Context) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}
	address := cCtx.String("address")

	log, _ := common.GetLogger()
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
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	var registrarAddr ethcommon.Address
	if address == "" {
		log.Info("Registrar address not provided, attempting to find in deployed contracts...")
		foundInDeployed := false
		for _, contract := range envCtx.DeployedContracts {
			if strings.Contains(strings.ToLower(contract.Name), "avsregistrar") {
				registrarAddr = ethcommon.HexToAddress(contract.Address)
				log.Info("Found AvsRegistrar: '%s' at address %s", contract.Name, registrarAddr.Hex())
				foundInDeployed = true
				break
			}
		}
		if !foundInDeployed {
			return fmt.Errorf("AvsRegistrar contract not found in deployed contracts for context '%s' and no address provided", devnet.CONTEXT)
		}
	} else {
		registrarAddr = ethcommon.HexToAddress(address)
	}

	return contractCaller.SetAVSRegistrar(cCtx.Context, avsAddr, registrarAddr)
}

func CreateAVSOperatorSetsAction(cCtx *cli.Context) error {
	cfg, err := common.LoadConfigWithContextConfig(devnet.CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	log, _ := common.GetLogger()
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
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	avsAddr := ethcommon.HexToAddress(envCtx.Avs.Address)
	if len(envCtx.OperatorSets) == 0 {
		log.Info("No operator sets to create.")
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

func RegisterOperatorELAction(cCtx *cli.Context) error {
	operatorAddress := cCtx.String("operator-address")
	if operatorAddress == "" {
		return fmt.Errorf("operator-address flag is required")
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
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	return contractCaller.RegisterAsOperator(cCtx.Context, ethcommon.HexToAddress(operatorAddress), 0, "test")
}

func RegisterOperatorAVSAction(cCtx *cli.Context) error {
	operatorAddress := cCtx.String("operator-address")
	operatorSetID := uint32(cCtx.Uint("operator-set-id"))
	payloadHex := cCtx.String("payload-hex")

	if operatorAddress == "" {
		return fmt.Errorf("operator-address flag is required")
	}
	if operatorSetID == 0 {
		return fmt.Errorf("operator-set-id flag is required")
	}
	if payloadHex == "" {
		return fmt.Errorf("payload-hex flag is required")
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

func registerOperatorsFromConfig(cCtx *cli.Context, cfg *common.ConfigWithContextConfig) error {
	log, _ := common.GetLogger()
	envCtx, ok := cfg.Context[devnet.CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.CONTEXT)
	}

	log.Info("Registering operators with EigenLayer...")
	if len(envCtx.OperatorRegistrations) == 0 {
		log.Info("No operator registrations found in context, skipping operator registration.")
		return nil
	}

	for _, opReg := range envCtx.OperatorRegistrations {
		log.Info("Processing registration for operator at address %s", opReg.Address)
		if err := RegisterOperatorELAction(cCtx); err != nil {
			log.Error("Failed to register operator %s with EigenLayer: %v. Continuing...", opReg.Address, err)
			continue
		}
		if err := RegisterOperatorAVSAction(cCtx); err != nil {
			log.Error("Failed to register operator %s for AVS: %v. Continuing...", opReg.Address, err)
			continue
		}
		log.Info("Successfully registered operator %s for OperatorSetID %d", opReg.Address, opReg.OperatorSetID)
	}
	log.Info("Operator registration with EigenLayer completed.")
	return nil
}
