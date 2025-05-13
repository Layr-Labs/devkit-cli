package commands

import (
	"devkit-cli/pkg/common"
	"fmt"

	"github.com/urfave/cli/v2"
)

var ConfigCommand = &cli.Command{
	Name:  "config",
	Usage: "Views or manages project-specific configuration (stored in eigen.toml)",
	Flags: append([]cli.Flag{
		&cli.BoolFlag{
			Name:  "list",
			Usage: "Display all current project configuration settings",
		},
		&cli.BoolFlag{
			Name:  "edit",
			Usage: "Open eigen.toml in a text editor for manual editing",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {

		projectSetting, err := common.LoadProjectSettings()

		if err != nil {
			return fmt.Errorf("failed to load project settings to get telemetry status: %v", err)
		}

		// Load config
		config, err := common.LoadBaseConfigWithoutContext()
		if err != nil {
			return fmt.Errorf("failed to load base config: %w", err)
		}

		listConfig(config, projectSetting)

		// if common.IsVerboseEnabled(cCtx, config) {
		// 	log.Info("Managing project configuration...")
		// }

		// if cCtx.Bool("edit") {
		// 	log.Printf("Opening config file for editing...")
		// 	return editConfig(cCtx)
		// }

		// list by default, if no flags are provided

		return nil
	},
}
