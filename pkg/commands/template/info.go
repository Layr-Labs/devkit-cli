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
		logger := common.LoggerFromContext(cCtx.Context)

		// Get template information
		projectName, templateBaseURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			return err
		}

		// Display template information
		logger.InfoWithActor("User", "Project template information:")
		if projectName != "" {
			logger.InfoWithActor("User", "  Project name: %s", projectName)
		}
		logger.InfoWithActor("User", "  Template URL: %s", templateBaseURL)
		logger.InfoWithActor("User", "  Version: %s", templateVersion)

		return nil
	},
}
