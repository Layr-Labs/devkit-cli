package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func listConfig(config *common.ConfigWithContextConfig, projectSettings *common.ProjectSettings) error {
	fmt.Printf("Displaying current configuration... \n\n")
	fmt.Printf("telemetry enabled: %t \n", projectSettings.TelemetryEnabled)

	fmt.Printf("Project: %s \n", config.Config.Project.Name)
	fmt.Printf("Version: %s \n", config.Config.Project.Version)

	// Read all files from config/contexts/
	contextDir := filepath.Join("config", "contexts")
	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return fmt.Errorf("failed to read contexts directory: %w", err)
	}

	fmt.Println("Available Contexts: ")
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		ctxPath := filepath.Join(contextDir, entry.Name())

		data, err := os.ReadFile(ctxPath)
		if err != nil {
			fmt.Printf("  %s: failed to read (%v)\n", name, err)
			continue
		}

		var wrapper struct {
			Context common.ChainContextConfig `yaml:"context"`
		}
		if err := yaml.Unmarshal(data, &wrapper); err != nil {
			fmt.Printf("  %s: failed to parse (%v)\n", name, err)
			continue
		}

		ctx := wrapper.Context
		fmt.Printf("  - %s:\n", name)
		fmt.Printf("      Name: %s\n\n", ctx.Name)
		fmt.Printf("      Chains: \n")
		for _, chain := range ctx.Chains {
			fmt.Printf("        Chain Name: %s\n", chain.Name)
			fmt.Printf("        Chain ID: %d\n", chain.ChainID)
			fmt.Printf("        Rpc Url: %s\n", chain.RPCURL)
			fmt.Printf("        Fork Mainnet Url: %s\n", chain.Fork.Url)
			fmt.Printf("        Fork Block Number: %d\n\n", chain.Fork.Block)
		}
		fmt.Printf("        Deployer Private Key: %s\n\n", ctx.DeployerPrivateKey)
		fmt.Printf("        App Private Key: %s\n\n", ctx.AppDeployerPrivateKey)
		fmt.Printf("        Operators: \n")
		for _, operator := range ctx.Operators {
			fmt.Printf("        Ecdsa key: %s\n", operator.ECDSAKey)
			fmt.Printf("        Bls keystore path: %s\n", operator.BlsKeystorePath)
			fmt.Printf("        Bls keystore password: %s\n", operator.BlsKeystorePassword)
			fmt.Printf("        Stake: %s\n\n", operator.Stake)
		}
		fmt.Printf("        Avs: \n")
		fmt.Printf("        Avs Address: %s\n", ctx.Avs.Address)
		fmt.Printf("        Avs Metadata Url: %s\n", ctx.Avs.MetadataUri)
		fmt.Printf("        Avs Registrar Address: %s\n", ctx.Avs.RegistrarAddress)

	}

	return nil

}
