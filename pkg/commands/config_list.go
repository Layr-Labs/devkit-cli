package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"gopkg.in/yaml.v3"
)

func listConfig(config *common.ConfigWithContextConfig, projectSettings *common.ProjectSettings) error {
	log, _ := common.GetLogger()

	log.Info("Displaying current configuration... \n\n")
	log.Info("Telemetry enabled: %t \n", projectSettings.TelemetryEnabled)

	log.Info("Project: %s\n", config.Config.Project.Name)
	log.Info("Version: %s\n\n", config.Config.Project.Version)

	// set the config location
	configDir := filepath.Join("config")
	filePath := filepath.Join(configDir, common.BaseConfig)

	// load the raw YAML node tree so we preserve ordering
	rootNode, err := common.LoadYAML(filePath)
	if err != nil {
		return fmt.Errorf("❌ Failed to read or parse %s: %v\n\n", common.BaseConfig, err)
	}

	// mark what is being printed
	log.Info("%s\n", filePath)
	log.Info(strings.Repeat("-", len(filePath)+2))

	// encode the node back to YAML on stdout, preserving order & comments
	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	if err := enc.Encode(rootNode); err != nil {
		enc.Close()
		return fmt.Errorf("Failed to emit %s: %v\n\n", common.BaseConfig, err)
	}
	enc.Close()
	log.Info("")

	// @TODO: This needs to be moved into a seperate context_list.go
	// // display contexts in original order, with comments
	// contextDir := filepath.Join("config", "contexts")
	// entries, err := os.ReadDir(contextDir)
	// if err != nil {
	// 	return fmt.Errorf("failed to read contexts directory: %w", err)
	// }

	// log.Info("Available Contexts:\n")
	// for _, entry := range entries {
	// 	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
	// 		continue
	// 	}
	// 	filePath := filepath.Join(contextDir, entry.Name())
	// 	rootNode, err := common.LoadYAML(filePath)
	// 	if err != nil {
	// 		log.Info("❌ Failed to load %s: %v\n\n", entry.Name(), err)
	// 		continue
	// 	}

	// 	log.Info("%s\n", filePath)
	// 	log.Info(strings.Repeat("-", len(filePath)+2))

	// 	enc := yaml.NewEncoder(os.Stdout)
	// 	enc.SetIndent(2)
	// 	if err := enc.Encode(rootNode); err != nil {
	// 		log.Info("Failed to emit %s: %v\n\n", filePath, err)
	// 		enc.Close()
	// 		continue
	// 	}
	// 	enc.Close()
	// 	log.Info("")
	// }

	return nil
}
