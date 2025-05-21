package template

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// InfoCommand defines the "template info" subcommand
var InfoCommand = &cli.Command{
	Name:  "info",
	Usage: "Display information about the current project template",
	Action: func(cCtx *cli.Context) error {
		// Get logger
		log, _ := common.GetLogger()

		// Get template information
		projectName, templateBaseURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			return err
		}

		// Display template information
		log.Info("Project template information:")
		if projectName != "" {
			log.Info("  Project name: %s", projectName)
		}
		log.Info("  Template URL: %s", templateBaseURL)
		log.Info("  Version: %s", templateVersion)

		return nil
	},
}
