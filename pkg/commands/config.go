package commands

import (
	"devkit-cli/pkg/common"

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
		// Load config
		// config, err := common.LoadBaseConfig()
		// if err != nil {
		// 	return err
		// }

		// if common.IsVerboseEnabled(cCtx, config) {
		// 	log.Printf("Managing project configuration...")
		// }

		// if cCtx.Bool("edit") {
		// 	log.Printf("Opening config file for editing...")
		// 	return editConfig(cCtx)
		// }

		// // list by default, if no flags are provided
		// log.Println("Displaying current configuration...")
		// projectSetting, err := common.LoadProjectSettings()
		// if err != nil {
		// 	log.Printf("failed to load project settings to get telemetry status: %v", err)
		// } else {
		// 	log.Printf("telemetry enabled: %t", projectSetting.TelemetryEnabled)
		// }

		// file, err := os.Open(common.EigenTomlPath)
		// if err != nil {
		// 	return fmt.Errorf("failed to open eigen.toml: %w", err)
		// }
		// defer file.Close()

		// var data map[string]interface{}
		// err = toml.NewDecoder(file).Decode(&data)
		// if err != nil {
		// 	return fmt.Errorf("failed to decode eigen.toml: %w", err)
		// }

		// printTOMLMap(data, 0)
		return nil
	},
}
