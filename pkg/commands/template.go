package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// TemplateCommand defines the main "template" command for template operations
var TemplateCommand = &cli.Command{
	Name:  "template",
	Usage: "Manage project templates",
	Subcommands: []*cli.Command{
		{
			Name:  "upgrade",
			Usage: "Upgrade project to a newer template version",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "version",
					Usage:    "Template version (Git ref: tag, branch, or commit) to upgrade to",
					Required: true,
				},
			},
			Action: func(cCtx *cli.Context) error {
				// Get logger
				log, _ := common.GetLogger()

				// Ensure we're in a project directory (check for config/config.yaml)
				configPath := filepath.Join("config", common.BaseConfig)
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					return fmt.Errorf("config/config.yaml not found. Make sure you're in a devkit project directory")
				}

				// Get the requested version
				requestedVersion := cCtx.String("version")
				if requestedVersion == "" {
					return fmt.Errorf("template version is required. Use --version to specify")
				}

				// Read the config file to get the template URL
				configData, err := os.ReadFile(configPath)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}

				var configMap map[string]interface{}
				if err := yaml.Unmarshal(configData, &configMap); err != nil {
					return fmt.Errorf("failed to parse config file: %w", err)
				}

				// Extract template info
				templateBaseURL := ""
				currentVersion := ""

				if configSection, ok := configMap["config"].(map[string]interface{}); ok {
					if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
						if url, ok := projectMap["templateBaseUrl"].(string); ok {
							templateBaseURL = url
						}
						if version, ok := projectMap["templateVersion"].(string); ok {
							currentVersion = version
						}
					}
				}

				if templateBaseURL == "" {
					return fmt.Errorf("no template URL found in config. This project may not have been created with a template")
				}

				// For now, just echo back the information
				log.Info("Project template information:")
				log.Info("  Current template URL: %s", templateBaseURL)
				log.Info("  Current version: %s", currentVersion)
				log.Info("  Requested version: %s", requestedVersion)
				log.Info("")
				log.Info("This is a stub command. Template upgrade functionality not yet implemented.")

				return nil
			},
		},
	},
}
