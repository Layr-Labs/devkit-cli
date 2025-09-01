package commands

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
)

func StartDevnetAction(cCtx *cli.Context) error {
	// Check if docker is running, else try to start it
	if err := common.EnsureDockerIsRunning(cCtx); err != nil {

		if errors.Is(err, context.Canceled) {
			return err // propagate the cancellation directly
		}
		return cli.Exit(err.Error(), 1)
	}

	// Get logger
	logger := common.LoggerFromContext(cCtx)

	// Extract vars
	contextName := cCtx.String("context")
	skipAvsRun := cCtx.Bool("skip-avs-run")
	skipDeployContracts := cCtx.Bool("skip-deploy-contracts")
	skipTransporter := cCtx.Bool("skip-transporter")
	useZeus := cCtx.Bool("use-zeus")
	persist := cCtx.Bool("persist")

	// Migrate config
	configsMigratedCount, err := configs.MigrateConfig(logger)
	if err != nil {
		logger.Error("config migration failed: %w", err)
	}
	if configsMigratedCount > 0 {
		logger.Info("configs migrated: %d", configsMigratedCount)
	}

	// Migrate contexts
	contextsMigratedCount, err := contexts.MigrateContexts(logger)
	if err != nil {
		logger.Error("context migrations failed: %w", err)
	}
	if contextsMigratedCount > 0 {
		logger.Info("contexts migrated: %d", contextsMigratedCount)
	}

	// Load config for selected context
	var config *common.ConfigWithContextConfig
	if contextName == "" {
		config, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		config, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("loading config and context failed: %w", err)
	}

	// Prevent runs when context is not devnet
	if contextName != devnet.DEVNET_CONTEXT {
		return fmt.Errorf("devnet start failed: `devkit avs devnet start` only available on devnet - please run `devkit avs devnet start --context devnet`")
	}

	// Load the context nodes
	yamlPath, rootNode, contextNode, contextName, err := common.LoadContext(contextName)
	if err != nil {
		return fmt.Errorf("loading context nodes failed: %w", err)
	}

	// Extract context details
	envCtx, ok := config.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	// Fetch EigenLayer addresses using Zeus if requested
	if useZeus {
		err = common.UpdateContextWithZeusAddresses(cCtx.Context, logger, contextNode, contextName)
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
	l1Port := cCtx.Int("l1-port")
	l2Port := cCtx.Int("l2-port")

	if !devnet.IsPortAvailable(l2Port) {
		return fmt.Errorf("❌ Port %d is already in use. Please choose a different port using --l2-port", l2Port)
	}

	if !devnet.IsPortAvailable(l1Port) {
		return fmt.Errorf("❌ Port %d is already in use. Please choose a different port using --l1-port", l1Port)
	}
	if !devnet.IsPortAvailable(l2Port) {
		return fmt.Errorf("❌ L2 port %d is already in use. Please choose a different port using --port", l2Port)
	}

	chainImage := devnet.GetDevnetChainImageOrDefault(config)
	l1ChainArgs := devnet.GetL1DevnetChainArgsOrDefault(config)
	l2ChainArgs := devnet.GetL2DevnetChainArgsOrDefault(config)

	// Start timer
	startTime := time.Now()

	logger.Info("Starting L1 and L2 devnets...\n")

	// Docker-compose for anvil devnet
	composePath := devnet.WriteEmbeddedArtifacts()
	l1ForkUrl, err := common.GetForkUrlDefault(contextName, config, common.L1)
	if err != nil {
		return fmt.Errorf("L1 fork URL error %w", err)
	}
	l2ForkUrl, err := common.GetForkUrlDefault(contextName, config, common.L2)
	if err != nil {
		return fmt.Errorf("L2 fork URL error: %w", err)
	}

	// Error if the l1ForkUrl has not been modified
	if l1ForkUrl == "" {
		return fmt.Errorf("l1 fork-url not set; set l1 fork-url in ./config/context/devnet.yaml or .env and consult README for guidance")
	}
	// Error if the l2ForkUrl has not been modified
	if l2ForkUrl == "" {
		return fmt.Errorf("l2 fork-url not set; set l2 fork-url in ./config/context/devnet.yaml or .env and consult README for guidance")
	}

	// Ensure fork URL uses appropriate Docker host for container environments
	l1DockerForkUrl := devnet.EnsureDockerHost(l1ForkUrl)
	l2DockerForkUrl := devnet.EnsureDockerHost(l2ForkUrl)
	// Get the l1 block_time from env/config
	l1BlockTime, err := devnet.GetDevnetBlockTimeOrDefault(config, common.L1)
	if err != nil {
		l1BlockTime = 12
	}

	// Get the l2 block_time from env/config
	l2BlockTime, err := devnet.GetDevnetBlockTimeOrDefault(config, common.L2)
	if err != nil {
		l2BlockTime = 12
	}

	// Get the l1 chain_id from env/config
	l1ChainId, err := devnet.GetDevnetChainIdOrDefault(config, common.L1, logger)
	if err != nil {
		l1ChainId = devnet.DEFAULT_L1_ANVIL_CHAINID
	}

	// Get the l2 chain_id from env/config
	l2ChainId, err := devnet.GetDevnetChainIdOrDefault(config, common.L2, logger)
	if err != nil {
		l2ChainId = devnet.DEFAULT_L2_ANVIL_CHAINID
	}

	// Append config defined details to chainArgs for l1
	l1ChainArgs = fmt.Sprintf("%s --chain-id %d", l1ChainArgs, l1ChainId)
	l1ChainArgs = fmt.Sprintf("%s --block-time %d", l1ChainArgs, l1BlockTime)

	// Append config defined details to chainArgs for l2
	l2ChainArgs = fmt.Sprintf("%s --chain-id %d", l2ChainArgs, l2ChainId)
	l2ChainArgs = fmt.Sprintf("%s --block-time %d", l2ChainArgs, l2BlockTime)

	// Run docker compose up for anvil devnet
	cmd := exec.CommandContext(cCtx.Context, "docker", "compose", "-p", config.Config.Project.Name, "-f", composePath, "up", "-d")

	l1ContainerName := fmt.Sprintf("devkit-devnet-l1-%s", config.Config.Project.Name)
	l2ContainerName := fmt.Sprintf("devkit-devnet-l2-%s", config.Config.Project.Name)
	l1ChainConfig, found := envCtx.Chains[common.L1]
	if !found {
		return fmt.Errorf("failed to find a chain with name: l1 in devnet.yaml")
	}
	l2ChainConfig, found := envCtx.Chains[common.L2]
	if !found {
		return fmt.Errorf("failed to find a chain with name: l2 in devnet.yaml")
	}

	cmd.Env = append(os.Environ(),
		"FOUNDRY_IMAGE="+chainImage,
		"L1_ANVIL_ARGS="+l1ChainArgs,
		"L2_ANVIL_ARGS="+l2ChainArgs,
		fmt.Sprintf("L1_DEVNET_PORT=%d", l1Port),
		fmt.Sprintf("L2_DEVNET_PORT=%d", l2Port),
		"L1_FORK_RPC_URL="+l1DockerForkUrl,
		"L2_FORK_RPC_URL="+l2DockerForkUrl,
		fmt.Sprintf("L1_FORK_BLOCK_NUMBER=%d", l1ChainConfig.Fork.Block),
		fmt.Sprintf("L2_FORK_BLOCK_NUMBER=%d", l2ChainConfig.Fork.Block),
		"L1_AVS_CONTAINER_NAME="+l1ContainerName,
		"L2_AVS_CONTAINER_NAME="+l2ContainerName,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}

	// On cancel, stop the containers if we're not skipping deployContracts/avsRun and we're not persisting
	if !skipDeployContracts && !skipAvsRun && !persist {
		defer func() {
			logger.Info("Stopping containers")
			// Use background context to avoid cancellation issues during cleanup
			bgCtx := context.Background()

			l1Container := fmt.Sprintf("devkit-devnet-l1-%s", config.Config.Project.Name)
			l2Container := fmt.Sprintf("devkit-devnet-l2-%s", config.Config.Project.Name)

			logger.Info("Stopping individual containers: %s, %s", l1Container, l2Container)
			devnet.StopAndRemoveContainer(&cli.Context{Context: bgCtx}, l1Container)
			devnet.StopAndRemoveContainer(&cli.Context{Context: bgCtx}, l2Container)
		}()
	}

	// Construct RPC url to pass to scripts for l1 and l2
	l1RpcUrl := devnet.GetRPCURL(l1Port)
	l2RpcUrl := devnet.GetRPCURL(l2Port)
	logger.Info("Waiting for devnet to be ready...")

	// Get chains node
	chainsNode := common.GetChildByKey(contextNode, "chains")
	if chainsNode == nil {
		return fmt.Errorf("missing 'chains' key in context")
	}

	// Update RPC URLs for L1 chain
	l1ChainNode := common.GetChildByKey(chainsNode, common.L1)
	if l1ChainNode != nil {
		l1RpcUrlNode := common.GetChildByKey(l1ChainNode, "rpc_url")
		if l1RpcUrlNode != nil {
			l1RpcUrlNode.Value = l1RpcUrl
		}
	}

	// Update RPC URLs for L2 chain
	l2ChainNode := common.GetChildByKey(chainsNode, common.L2)
	if l2ChainNode != nil {
		l2RpcUrlNode := common.GetChildByKey(l2ChainNode, "rpc_url")
		if l2RpcUrlNode != nil {
			l2RpcUrlNode.Value = l2RpcUrl
		}
	}

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return err
	}

	// Sleep for 4 second to ensure the devnet is fully started
	time.Sleep(4 * time.Second)

	// Fund the wallets defined in config on L1
	logger.Info("Funding wallets on L1...")
	err = devnet.FundWalletsDevnet(config, l1RpcUrl)
	if err != nil {
		return fmt.Errorf("funding L1 devnet wallets failed: %w", err)
	}

	// Fund the wallets defined in config on L2
	logger.Info("Funding wallets on L2...")
	err = devnet.FundWalletsDevnet(config, l2RpcUrl)
	if err != nil {
		return fmt.Errorf("failed L2 devnet wallets failed: %w", err)
	}

	// Fund stakers with strategy tokens
	if contextName == devnet.DEVNET_CONTEXT {
		logger.Info("Funding stakers with strategy tokens...")

		var tokenAddresses []string
		var tokenErr error
		tokenAddresses, tokenErr = devnet.GetUnderlyingTokenAddressesFromStrategies(config, l1RpcUrl, logger)
		if tokenErr != nil {
			logger.Warn("Failed to get underlying token addresses from strategies: %v", tokenErr)
			logger.Info("Continuing with devnet startup...")
		}

		if len(tokenAddresses) > 0 {
			err = devnet.FundStakersWithStrategyTokens(config, l1RpcUrl, tokenAddresses)
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
	logger.Info("\nL1 devnet started successfully on port %d", l1Port)
	logger.Info("L2 devnet started successfully on port %d", l2Port)
	logger.Info("Total startup time: %s", elapsed)

	if err := WhitelistChainIdInCrossRegistryAction(cCtx, logger); err != nil {
		return fmt.Errorf("whitelisting chain id in cross registry failed: %w", err)
	}

	// Deploy the contracts after starting devnet unless skipped
	if !skipDeployContracts {
		if err := DeployL1ContractsAction(cCtx); err != nil { // Assumes DeployContractsAction remains as is or is also refactored if needed
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
				return fmt.Errorf("failed to configure OpSet in KeyRegistrar: %w", err)
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

	// Create a context that cancels on Ctrl-C (SIGINT) or docker/systemd stop (SIGTERM)
	ctx, stop := signal.NotifyContext(cCtx.Context, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Set up waitGroup to handle bg scheduler
	var wg sync.WaitGroup
	wg.Add(1)

	// Run Transport against schedule - exit when AVSRun exits
	if !skipTransporter {
		// Post initial stake roots to L1
		if err := Transport(cCtx, true); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("transport run failed: %w", err)
		}

		// Shallow-copy cli.Context so that ScheduleTransport sees the new ctx
		childCtx := *cCtx
		childCtx.Context = ctx

		// Run scheduler in the background
		go func() {
			if err := ScheduleTransport(&childCtx, envCtx.Transporter.Schedule); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("ScheduleTransport failed: %v", err)
				stop()
			}
		}()
	}

	// Sleep for 2 seconds
	time.Sleep(2 * time.Second)

	// Deploy L2 contracts only if L1 contracts were also deployed
	if !skipDeployContracts {
		if err := DeployL2ContractsAction(cCtx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("deploy-l2-contracts failed: %v", err)
			return fmt.Errorf("deploy-l2-contracts failed: %w", err)
		}
	}

	// Start offchain AVS components after starting devnet and deploying contracts unless skipped
	if !skipDeployContracts && !skipAvsRun {
		if err := AVSRun(cCtx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("avs run failed: %w", err)
		}
	}

	// Wait for the scheduler and close on interrupt
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-ctx.Done(): // user interrupt -> stop scheduler, exit
	case <-done: // scheduler ended on its own -> exit
	}

	return ctx.Err()
}

func StopDevnetAction(cCtx *cli.Context) error {
	// Get logger
	log := common.LoggerFromContext(cCtx)

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

	// Extract vars
	contextName := cCtx.String("context")
	projectName := cCtx.String("project.name")
	projectPort := cCtx.Int("port")
	l1Port := cCtx.Int("l1-port")
	l2Port := cCtx.Int("l2-port")

	// Check if any of the args are provided
	if !(projectName == "") || !(projectPort == 0) || !(l1Port == 0) || !(l2Port == 0) {
		if projectName != "" {
			// Stop both L1 and L2 containers
			l1Container := fmt.Sprintf("devkit-devnet-l1-%s", projectName)
			l2Container := fmt.Sprintf("devkit-devnet-l2-%s", projectName)

			devnet.StopAndRemoveContainer(cCtx, l1Container)
			devnet.StopAndRemoveContainer(cCtx, l2Container)
		} else if l1Port != 0 {
			// Stop only L1 container matching the port
			stopContainerByPort(cCtx, log, l1Port, "l1")
		} else if l2Port != 0 {
			// Stop only L2 container matching the port
			stopContainerByPort(cCtx, log, l2Port, "l2")
		} else if projectPort != 0 {
			// Stop both L1 and L2 containers for the project found on this port
			stopBothContainersByPort(cCtx, log, projectPort)
		}
		return nil
	}

	if devnet.FileExistsInRoot(filepath.Join(common.DefaultConfigWithContextConfigPath, common.BaseConfig)) {
		// Load config
		var err error
		var config *common.ConfigWithContextConfig
		if contextName == "" {
			config, _, err = common.LoadDefaultConfigWithContextConfig()
		} else {
			config, _, err = common.LoadConfigWithContextConfig(contextName)
		}
		if err != nil {
			return fmt.Errorf("loading config and context failed: %w", err)
		}

		// Stop both L1 and L2 containers
		l1Container := fmt.Sprintf("devkit-devnet-l1-%s", config.Config.Project.Name)
		l2Container := fmt.Sprintf("devkit-devnet-l2-%s", config.Config.Project.Name)

		devnet.StopAndRemoveContainer(cCtx, l1Container)
		devnet.StopAndRemoveContainer(cCtx, l2Container)
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

func DepositIntoStrategiesAction(cCtx *cli.Context, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	// Load config for selected context
	var cfg *common.ConfigWithContextConfig
	var err error
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for deposit into strategies: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
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

func DelegateToOperatorsAction(cCtx *cli.Context, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for delegate to operators: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
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

func extractHostPort(portStr string) string {
	if strings.Contains(portStr, "->") {
		beforeArrow := strings.Split(portStr, "->")[0]
		hostPort := strings.Split(beforeArrow, ":")
		return hostPort[len(hostPort)-1]
	}
	return portStr
}

func registerOperatorEL(cCtx *cli.Context, operatorAddress string, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	if operatorAddress == "" {
		return fmt.Errorf("operatorAddress parameter is required and cannot be empty")
	}

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}
	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	var operatorPrivateKey string
	var foundOperator bool
	for _, op := range envCtx.Operators {
		// Try to load the ECDSA key
		keyHex, err := loadOperatorECDSAKey(op)
		if err != nil {
			continue
		}

		key, keyErr := crypto.HexToECDSA(keyHex)
		if keyErr != nil {
			continue
		}
		if strings.EqualFold(crypto.PubkeyToAddress(key.PublicKey).Hex(), operatorAddress) {
			operatorPrivateKey = keyHex
			foundOperator = true
			break
		}
	}
	if !foundOperator {
		return fmt.Errorf("operator with address %s not found in config", operatorAddress)
	}
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)

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
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	return contractCaller.RegisterAsOperator(cCtx.Context, ethcommon.HexToAddress(operatorAddress), 0, "test")
}

func registerOperatorAVS(cCtx *cli.Context, logger iface.Logger, operatorAddress string, operatorSetID uint32, payloadHex string) error {
	// Extract vars
	contextName := cCtx.String("context")

	if operatorAddress == "" {
		return fmt.Errorf("operatorAddress parameter is required and cannot be empty")
	}
	if payloadHex == "" {
		return fmt.Errorf("payloadHex parameter is required and cannot be empty")
	}

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}
	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	var operatorPrivateKey string
	var foundOperator bool
	for _, op := range envCtx.Operators {
		// Try to load the ECDSA key
		keyHex, err := loadOperatorECDSAKey(op)
		if err != nil {
			continue
		}

		key, keyErr := crypto.HexToECDSA(keyHex)
		if keyErr != nil {
			continue
		}
		if strings.EqualFold(crypto.PubkeyToAddress(key.PublicKey).Hex(), operatorAddress) {
			operatorPrivateKey = keyHex
			foundOperator = true
			break
		}
	}
	if !foundOperator {
		return fmt.Errorf("operator with address %s not found in config", operatorAddress)
	}

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)

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
	// Extract vars
	contextName := cCtx.String("context")

	if stakerSpec.StakerAddress == "" {
		return fmt.Errorf("staker address parameter is required and cannot be empty")
	}

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)
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
	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)
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
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}
	// After depositing, delegate to the operator
	// Extract the private key of the operator we are delegating to in order to create an approval signature
	var operatorPrivateKey string
	var foundOperator bool
	for _, op := range envCtx.Operators {
		if strings.EqualFold(op.Address, operator.Hex()) {
			keyHex, err := loadOperatorECDSAKey(op)
			if err != nil {
				return fmt.Errorf("failed to load ECDSA key for operator %s: %w", operator, err)
			}
			operatorPrivateKey = keyHex
			foundOperator = true
			break
		}
	}
	if !foundOperator {
		return fmt.Errorf("ECDSA key not found for operator %s in operators in config. This means we cannot create an approval signature for this delegation", operator)
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

func ModifyAllocationsAction(cCtx *cli.Context, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for modify allocations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	for _, op := range envCtx.Operators {
		logger.Info("Modifying allocations for operator %s", op.Address)
		if len(op.Allocations) == 0 {
			logger.Info("Operator %s has no allocations specified, skipping allocation modification", op.Address)
			continue
		}
		// Load ECDSA key for operator
		operatorKey, err := loadOperatorECDSAKey(op)
		if err != nil {
			logger.Debug("Failed to load ECDSA key for operator %s: %v. Continuing...", op.Address, err)
			continue
		}
		if err := modifyAllocations(cCtx, op.Address, operatorKey, logger); err != nil {
			logger.Debug("Failed to modify allocations for operator %s: %v. Continuing...", op.Address, err)
			continue
		}
	}
	logger.Info("Modifying allocations completed.")
	return nil
}

func modifyAllocations(cCtx *cli.Context, operatorAddress string, operatorPrivateKey string, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	if operatorAddress == "" {
		return fmt.Errorf("modifyAllocations:operatorAddress parameter is required and cannot be empty")
	}

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
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

			allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)

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
	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for set allocation delay: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	// Instead of mining blocks(because it's infeasible for 126000 blocks(for mainnet) or 30 on sepolia), use anvil_setStorageAt to bypass ALLOCATION_CONFIGURATION_DELAY
	// We need to manipulate the storage that tracks when allocation delays were set for each operator by modifying
	// the effectBlock field in the AllocationDelayInfo struct.
	logger.Info("Bypassing allocation configuration delay using anvil_setStorageAt...")

	allocationManagerAddr, _, _, _, _, _, _, _ := common.GetEigenLayerAddresses(contextName, cfg)
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

func WhitelistChainIdInCrossRegistryAction(cCtx *cli.Context, logger iface.Logger) error {
	// Extract vars
	contextName := cCtx.String("context")

	// Skip this call if funding is disabled
	if os.Getenv("SKIP_DEVNET_FUNDING") == "true" {
		log.Println("🔧 Skipping WhitelistChainIdInCrossRegistry (test mode)")
		return nil
	}

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	l2Cfg, ok := envCtx.Chains[common.L2]
	if !ok {
		return fmt.Errorf("failed to get l2 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	crossChainRegistryAddr := ethcommon.HexToAddress(envCtx.EigenLayer.L1.CrossChainRegistry)
	l1OperatorTableUpdater := ethcommon.HexToAddress(envCtx.EigenLayer.L1.OperatorTableUpdater)
	l2OperatorTableUpdater := ethcommon.HexToAddress(envCtx.EigenLayer.L2.OperatorTableUpdater)

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
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}
	// whitelist l1 chain id in cross registry
	err = contractCaller.WhitelistChainIdInCrossRegistry(cCtx.Context, l1OperatorTableUpdater, uint64(l1Cfg.ChainID))
	if err != nil {
		return fmt.Errorf("failed to whitelist l1 ChainId in CrossChainRegistry: %w", err)
	}

	// whitelist l2 chain id in cross registry
	err = contractCaller.WhitelistChainIdInCrossRegistry(cCtx.Context, l2OperatorTableUpdater, uint64(l2Cfg.ChainID))
	if err != nil {
		return fmt.Errorf("failed to whitelist l2 ChainId in CrossChainRegistry: %w", err)
	}

	logger.Info("Successfully whitelisted l1 chain id in cross registry")
	return nil
}

// stopContainerByPort stops a specific container (L1 or L2) running on the given port
func stopContainerByPort(cCtx *cli.Context, log iface.Logger, targetPort int, containerType string) {
	cmd := exec.CommandContext(cCtx.Context, "docker", devnet.GetDockerPsDevnetArgs()...)
	output, err := cmd.Output()
	if err != nil {
		log.Warn("Failed to list running devnet containers: %v", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	containerFound := false

	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}
		containerName := parts[0]
		port := parts[1]
		hostPort := extractHostPort(port)

		if hostPort == fmt.Sprintf("%d", targetPort) {
			// Check if this is the right container type (l1 or l2)
			if (containerType == devnet.L1_CONTAINER_TYPE && strings.Contains(containerName, devnet.L1_CONTAINER_NAME_PREFIX)) ||
				(containerType == devnet.L2_CONTAINER_TYPE && strings.Contains(containerName, devnet.L2_CONTAINER_NAME_PREFIX)) ||
				(containerType == devnet.L1_CONTAINER_TYPE && !strings.Contains(containerName, devnet.L1_CONTAINER_TYPE) && !strings.Contains(containerName, devnet.L2_CONTAINER_TYPE)) { // fallback for old naming

				devnet.StopAndRemoveContainer(cCtx, containerName)
				log.Info("Stopped %s devnet container %s running on port %d", strings.ToUpper(containerType), containerName, targetPort)
				containerFound = true
				break
			}
		}
	}

	if !containerFound {
		log.Info("No %s container found running on port %d. Try %sdevkit avs devnet list%s to get a list of running devnet containers",
			strings.ToUpper(containerType), targetPort, devnet.Cyan, devnet.Reset)
	}
}

// stopBothContainersByPort stops both L1 and L2 containers for the project found on the given port
func stopBothContainersByPort(cCtx *cli.Context, log iface.Logger, targetPort int) {
	cmd := exec.CommandContext(cCtx.Context, "docker", devnet.GetDockerPsDevnetArgs()...)
	output, err := cmd.Output()
	if err != nil {
		log.Warn("Failed to list running devnet containers: %v", err)
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	containerFound := false
	projectsToStop := make(map[string]bool) // Track projects we've already stopped

	for _, line := range lines {
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			continue
		}
		containerName := parts[0]
		port := parts[1]
		hostPort := extractHostPort(port)

		if hostPort == fmt.Sprintf("%d", targetPort) {
			// Extract project name from container name
			var projectName string
			if strings.HasPrefix(containerName, "devkit-devnet-l1-") {
				projectName = strings.TrimPrefix(containerName, "devkit-devnet-l1-")
			} else if strings.HasPrefix(containerName, "devkit-devnet-l2-") {
				projectName = strings.TrimPrefix(containerName, "devkit-devnet-l2-")
			} else {
				// Fallback for old naming convention
				projectName = strings.TrimPrefix(containerName, "devkit-devnet-")
			}

			// If we haven't stopped this project yet, stop both L1 and L2 containers
			if !projectsToStop[projectName] {
				l1Container := fmt.Sprintf("devkit-devnet-l1-%s", projectName)
				l2Container := fmt.Sprintf("devkit-devnet-l2-%s", projectName)

				devnet.StopAndRemoveContainer(cCtx, l1Container)
				devnet.StopAndRemoveContainer(cCtx, l2Container)

				log.Info("Stopped both L1 and L2 devnet containers for project %s (found port %d)", projectName, targetPort)
				projectsToStop[projectName] = true
				containerFound = true
			}
		}
	}

	if !containerFound {
		log.Info("No container found with port %d. Try %sdevkit avs devnet list%s to get a list of running devnet containers",
			targetPort, devnet.Cyan, devnet.Reset)
	}
}

// loadOperatorECDSAKey loads an operator's ECDSA private key from keystore or plaintext
func loadOperatorECDSAKey(operator common.OperatorSpec) (string, error) {
	// Check if ECDSA keystore is configured
	if len(operator.Keystores) > 0 && operator.Keystores[0].ECDSAKeystorePath != "" && operator.Keystores[0].ECDSAKeystorePassword != "" {
		// Load from keystore
		keystoreData, err := os.ReadFile(operator.Keystores[0].ECDSAKeystorePath)
		if err != nil {
			return "", fmt.Errorf("failed to read ECDSA keystore file %s: %w", operator.Keystores[0].ECDSAKeystorePath, err)
		}

		key, err := ethkeystore.DecryptKey(keystoreData, operator.Keystores[0].ECDSAKeystorePassword)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt ECDSA keystore: %w", err)
		}

		return hex.EncodeToString(crypto.FromECDSA(key.PrivateKey)), nil
	}

	// Fall back to plaintext key
	if operator.ECDSAKey != "" {
		return strings.ToLower(strings.TrimPrefix(operator.ECDSAKey, "0x")), nil
	}

	return "", fmt.Errorf("no ECDSA key configuration found for operator %s", operator.Address)
}
