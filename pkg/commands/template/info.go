package template

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// InfoCommand defines the "info" command for templates
var InfoCommand = &cli.Command{
	Name:  "info",
	Usage: "Display template information for current project",
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		// Get template information
		projectName, templateBaseURL, templateVersion, err := GetTemplateInfo()
		if err != nil {
			return err
		}

		// Display template information
		logger.InfoWithActor(iface.ActorAVSDev, "Project template information:")
		if projectName != "" {
			logger.InfoWithActor(iface.ActorAVSDev, "  Project name: %s", projectName)
		}
		logger.InfoWithActor(iface.ActorAVSDev, "  Template URL: %s", templateBaseURL)
		logger.InfoWithActor(iface.ActorAVSDev, "  Version: %s", templateVersion)

		return nil
	},
}
