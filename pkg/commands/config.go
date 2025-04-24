package commands

import (
	"devkit-cli/pkg/common"
	"github.com/pelletier/go-toml"
	"github.com/urfave/cli/v2"
	"log"
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

		if cCtx.Bool("verbose") {
			log.Println("Managing project configuration...")
		}

		// load by default , if --set is not provided
		map_val, err := common.StructToMap(config)
		if err != nil {
			return err
		}
		tree, _ := toml.TreeFromMap(map_val)
		common.PrintStyledConfig(tree.String())

		return nil
	},
}
