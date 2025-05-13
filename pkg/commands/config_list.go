package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func listConfig(config *common.BaseConfig, projectSettings *common.ProjectSettings) error {
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

	fmt.Println("Available Contexts: \n")
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
		fmt.Printf("      Name: %s\n", ctx.Name)
		fmt.Printf("      Chain ID: %d\n", ctx.ChainID)
		fmt.Printf("      RPC URL: %s\n", ctx.RPCURL)
	}

	return nil

}
