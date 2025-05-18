package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/devnet"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func StartDevnetAction(cCtx *cli.Context) error {
	// Get logger
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
	fork_url, err := devnet.GetDevnetForkUrlDefault(config, devnet.L1)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

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
		"FORK_RPC_URL="+fork_url,
		fmt.Sprintf("FORK_BLOCK_NUMBER=%d", l1ChainConfig.Fork.Block),
		"AVS_CONTAINER_NAME="+containerName,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}
	rpcUrl := fmt.Sprintf("http://localhost:%d", port)

	// Sleep for 4 second to ensure the devnet is fully started
	time.Sleep(4 * time.Second)

	// Fund the wallets defined in config
	devnet.FundWalletsDevnet(config, rpcUrl)
	elapsed := time.Since(startTime).Round(time.Second)

	// Sleep for 1 second to make sure wallets are funded
	time.Sleep(1 * time.Second)
	log.Info("\nDevnet started successfully in %s", elapsed)

	// Deploy the contracts after starting devnet unless skipped
	if !skipDeployContracts {
		if err := DeployContractsAction(cCtx); err != nil {
			return fmt.Errorf("deploy-contracts failed: %w", err)
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
	// Get logger
	log, _ := common.GetLogger()

	// Start timing execution runtime
	startTime := time.Now()

	// Run scriptPath from cwd
	const dir = ""

	// Set path for .devkit scripts
	scriptsDir := filepath.Join(".devkit", "scripts")

	// Set path for context yaml
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, "devnet.yaml") // @TODO: use selected context name

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
		return fmt.Errorf("missing 'context' key in ./config/contexts/devnet.yaml")
	}

	// Loop scripts with cloned context
	for _, name := range scriptNames {
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
		outMap, err := common.CallTemplateScript(cCtx.Context, dir, scriptPath, common.ExpectJSONResponse, inputJSON)
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

	// Write yaml back to project directory
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return err
	}

	// Measure how long we ran for
	elapsed := time.Since(startTime).Round(time.Second)
	log.Info("\nDevnet contracts deployed successfully in %s", elapsed)
	return nil
}

func StopDevnetAction(cCtx *cli.Context) error {
	// Get logger
	log, _ := common.GetLogger()

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

func extractHostPort(portStr string) string {
	if strings.Contains(portStr, "->") {
		beforeArrow := strings.Split(portStr, "->")[0]
		hostPort := strings.Split(beforeArrow, ":")
		return hostPort[len(hostPort)-1]
	}
	return portStr
}
