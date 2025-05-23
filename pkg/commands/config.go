package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"gopkg.in/yaml.v3"

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
		&cli.BoolFlag{
			Name:  "edit",
			Usage: "Open config file in a text editor for manual editing",
		},
		&cli.StringSliceFlag{
			Name:  "set",
			Usage: "Set a value into the current projects configuration settings (--set project.name=value)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		// Open editor for the project level config
		if cCtx.Bool("edit") {
			logger.Info("Opening config file for editing...")
			return editConfig(cCtx, filepath.Join("config", common.BaseConfig))
		}

		// Get the sets
		items := cCtx.StringSlice("set")
		// Slice any position args to the items list
		items = append(items, cCtx.Args().Slice()...)

		// Set values using dot.delim to navigate keys
		if len(items) > 0 {
			cfgPath := filepath.Join("config", common.BaseConfig)
			rootDoc, err := common.LoadYAML(cfgPath)
			if err != nil {
				return fmt.Errorf("read config YAML: %w", err)
			}
			root := rootDoc.Content[0]
			configNode := common.GetChildByKey(root, "config")
			if configNode == nil {
				configNode = &yaml.Node{Kind: yaml.MappingNode}
				root.Content = append(root.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "config"},
					configNode,
				)
			}
			for _, item := range items {
				// Split into "key.path.to.field" and "value"
				parts := strings.SplitN(item, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --set syntax %q (want key=val)", item)
				}

				// Break the key path into segments
				path := strings.Split(parts[0], ".")
				val := parts[1]

				// Set val at path
				configNode, err = common.WriteToPath(configNode, path, val)
				if err != nil {
					return fmt.Errorf("setting value %s failed: %w", item, err)
				}
				logger.Info("Set %s = %s", parts[0], val)

			}
			if err := common.WriteYAML(cfgPath, rootDoc); err != nil {
				return fmt.Errorf("write config YAML: %w", err)
			}
			return nil
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
