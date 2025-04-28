package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"github.com/naoina/toml"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strings"
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
			log.Printf("failed to load project settings to get telemetry status: %v", err)
		} else {
			log.Printf("telemetry enabled: %t", projectSetting.TelemetryEnabled)
		}

		file, err := os.Open(common.EigenTomlPath)
		if err != nil {
			return fmt.Errorf("failed to open eigen.toml: %w", err)
		}
		defer file.Close()

		var data map[string]interface{}
		err = toml.NewDecoder(file).Decode(&data)
		if err != nil {
			return fmt.Errorf("failed to decode eigen.toml: %w", err)
		}

		printTOMLMap(data, 0)
		return nil
	},
}

func printTOMLMap(m map[string]interface{}, indent int) {
	pad := strings.Repeat("  ", indent)
	for key, val := range m {
		switch v := val.(type) {
		case map[string]interface{}:
			fmt.Printf("%s[%s]\n", pad, key)
			printTOMLMap(v, indent+1)
		default:
			fmt.Printf("%s%s = %v\n", pad, key, v)
		}
	}
}
