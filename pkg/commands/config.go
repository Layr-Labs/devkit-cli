package commands

import (
	"fmt"
	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

var ConfigCommand = &cli.Command{
	Name:  "config",
	Usage: "Views or manages project-specific configuration (stored in config directory)",
	Flags: append([]cli.Flag{
		&cli.BoolFlag{
			Name:  "list",
			Usage: "Display all current project configuration settings",
		},
		&cli.StringFlag{
			Name:  "edit",
			Usage: "Open config file in a text editor for manual editing",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		log, _ := common.GetLogger()

		if path := cCtx.String("edit"); path != "" {
			log.Info("Opening config file for editing...")
			return editConfig(cCtx, path)
		}

		// list by default, if no flags are provided
		projectSetting, err := common.LoadProjectSettings()

		if err != nil {
			return fmt.Errorf("failed to load project settings to get telemetry status: %v", err)
		}

		// Load config
		config, err := common.LoadConfigWithContextConfigWithoutContext()
		if err != nil {
			return fmt.Errorf("failed to load config and context config: %w", err)
		}

		err = listConfig(config, projectSetting)
		if err != nil {
			return fmt.Errorf("failed to list config %w", err)
		}
		return nil
	},
}
