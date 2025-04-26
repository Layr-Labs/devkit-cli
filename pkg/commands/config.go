package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"
	"strings"

	"github.com/urfave/cli/v2"
)

// ConfigCommand defines the "config" command
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
		if cCtx.Bool("verbose") {
			log.Printf("Managing project configuration...")
		}

		if cCtx.Bool("list") {
			log.Printf("Listing all configuration settings...")
			// Placeholder for future implementation
			return nil
		}

		if setValue := cCtx.String("set"); setValue != "" {
			log.Printf("Setting configuration: %s", setValue)
		
			parts := strings.SplitN(setValue, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid format for --set. Expected key=value")
			}
			keyPath := parts[0]
			rawValue := parts[1]
		
			var value interface{}
			if strings.Contains(rawValue, ",") {
				value = strings.Split(rawValue, ",")
			} else {
				value = rawValue
			}
		
			tree, err := common.LoadEigenTree()
			if err != nil {
				return err
			}
		
			if err := common.SetKey(tree, keyPath, value); err != nil {
				return err
			}
		
			if err := common.SaveEigenTree(tree); err != nil {
				return err
			}
		
			log.Printf("✅ Updated %s in eigen.toml", keyPath)
			return nil
		}
		

		// If no flags are provided, show current config
		log.Printf("Displaying current configuration...")
		return nil
	},
}
