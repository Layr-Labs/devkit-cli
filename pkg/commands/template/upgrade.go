package template

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// UpgradeCommand defines the "template upgrade" subcommand
var UpgradeCommand = &cli.Command{
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

		// Get the requested version
		requestedVersion := cCtx.String("version")
		if requestedVersion == "" {
			return fmt.Errorf("template version is required. Use --version to specify")
		}

		// Get template information
		projectName, templateBaseURL, currentVersion, err := GetTemplateInfo()
		if err != nil {
			return err
		}

		if templateBaseURL == "" {
			return fmt.Errorf("no template URL found in config. This project may not have been created with a template")
		}

		// For now, just echo back the information
		log.Info("Project template information:")
		if projectName != "" {
			log.Info("  Project name: %s", projectName)
		}
		log.Info("  Current template URL: %s", templateBaseURL)
		log.Info("  Current version: %s", currentVersion)
		log.Info("  Requested version: %s", requestedVersion)
		log.Info("")
		log.Info("This is a stub command. Template upgrade functionality not yet implemented.")

		return nil
	},
}
