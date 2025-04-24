package commands

import (
	"devkit-cli/pkg/common"
	"log"

	"github.com/pelletier/go-toml"
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
		&cli.StringFlag{
			Name:  "set",
			Usage: "Set or update a specific configuration key in eigen.toml",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Load config
		config, err := common.LoadEigenConfig()
		if err != nil {
			return err
		}
		if setValue := cCtx.String("set"); setValue != "" {
			log.Printf("Setting configuration: %s", setValue)
			// TODO: Parse and apply the key=value update
			return nil
		}

		// load by default , if --set is not provided
		// dev: If any other subcommand needs to be added in ConfigCommand apart from set and list, handle it above this line.
		log.Println("Displaying current configuration...")
		projectSetting, err := common.LoadProjectSettings()
		if err != nil {
			log.Printf("failed to load project settings to get telemetry status: %w", err)
		} else {
			log.Printf("telemetry enabled %s", projectSetting.TelemetryEnabled)
		}
		map_val, err := common.StructToMap(config)
		if err != nil {
			return err
		}
		tree, _ := toml.TreeFromMap(map_val)
		common.PrintStyledConfig(tree.String())

		return nil
	},
}
