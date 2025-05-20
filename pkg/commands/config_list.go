package commands

import (
	"fmt"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

func listConfig(config *common.ConfigWithContextConfig, projectSettings *common.ProjectSettings) error {
	fmt.Printf("Displaying current configuration... \n\n")
	fmt.Printf("Telemetry enabled: %t \n", projectSettings.TelemetryEnabled)

	fmt.Printf("Project: %s\n", config.Config.Project.Name)
	fmt.Printf("Version: %s\n\n", config.Config.Project.Version)

	contextDir := filepath.Join("config", "contexts")
	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return fmt.Errorf("failed to read contexts directory: %w", err)
	}

	fmt.Println("Available Contexts:")

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		filePath := filepath.Join(contextDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("❌ Failed to read %s: %v\n\n", entry.Name(), err)
			continue
		}

		var content map[string]interface{}
		if err := yaml.Unmarshal(data, &content); err != nil {
			fmt.Printf("Failed to parse %s: %v\n\n", entry.Name(), err)
			continue
		}

		fmt.Printf("%s\n", entry.Name())
		fmt.Println(strings.Repeat("-", len(entry.Name())+2))

		yamlContent, err := yaml.Marshal(content)
		if err != nil {
			fmt.Printf("Failed to marshal %s: %v\n\n", entry.Name(), err)
			continue
		}

		fmt.Println(string(yamlContent))
		fmt.Println()
	}

	return nil
}
