package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/commands/config"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"gopkg.in/yaml.v3"

	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:  "context",
	Usage: "Views or manages context-specific configuration (stored in config/contexts directory)",
	Subcommands: []*cli.Command{
		CreateContextCommand,
	},
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to work over",
		},
		&cli.BoolFlag{
			Name:  "list",
			Usage: "Display all current context settings",
		},
		&cli.BoolFlag{
			Name:  "edit",
			Usage: "Open selected context file in a text editor for manual editing",
		},
		&cli.StringSliceFlag{
			Name:  "set",
			Usage: "Set a value into the current context settings (--set project.name=value)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		// Identify the context we are working against
		context := cCtx.String("context")
		// Locate the context directory
		contextDir := filepath.Join("config", "contexts")
		// Pull positional args
		args := cCtx.Args().Slice()
		// Get the sets
		items := cCtx.StringSlice("set")

		// Pull available contexts
		if cCtx.String("context") == "" && (len(args) == 0 || len(items) > 0) {
			// List available contexts
			ctx, err := ListContexts(contextDir, cCtx.Bool("list"))
			if err != nil {
				return fmt.Errorf("failed to list contexts %w", err)
			}
			// Place the selection
			context = ctx[0]
			// add empty line
			fmt.Println()
		} else if cCtx.String("context") == "" && len(cCtx.Args().Slice()) > 0 {
			// Select the last arg
			last := len(args) - 1
			// Only treat as context if it’s not a key=value
			if !strings.Contains(args[last], "=") {
				context = args[last]
				args = args[:last]
			}
		}

		// No context provided
		if context == "" {
			return fmt.Errorf("cannot proceed without a selected context")
		}

		// Path to the context.yaml file
		contextPath := filepath.Join(contextDir, fmt.Sprintf("%s.yaml", context))

		// Open editor for the context level config
		if cCtx.Bool("edit") {
			logger.Info("Opening context file for editing...")
			return config.EditConfig(cCtx, contextPath, config.Context, context)
		}

		// Set values using dot.delim to navigate keys
		if len(items) > 0 {
			// Slice any position args to the items list
			items = append(items, args...)

			// Load the context yaml
			rootDoc, err := common.LoadYAML(contextPath)
			if err != nil {
				return fmt.Errorf("read context YAML: %w", err)
			}
			root := rootDoc.Content[0]
			configNode := common.GetChildByKey(root, "context")
			if configNode == nil {
				configNode = &yaml.Node{Kind: yaml.MappingNode}
				root.Content = append(root.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "context"},
					configNode,
				)
			}
			for _, item := range items {
				// Split into "key.path.to.field" and "value"
				idx := strings.LastIndex(item, "=")
				if idx < 0 {
					return fmt.Errorf("invalid --set syntax %q (want key=val)", item)
				}
				pathStr := item[:idx]
				val := item[idx+1:]

				// Break the key path into segments
				path := strings.Split(pathStr, ".")

				// Set val at path
				configNode, err = common.WriteToPath(configNode, path, val)
				if err != nil {
					return fmt.Errorf("setting value %s failed: %w", item, err)
				}
				logger.Info("Set %s = %s", pathStr, val)
			}
			if err := common.WriteYAML(contextPath, rootDoc); err != nil {
				return fmt.Errorf("write context YAML: %w", err)
			}
			return nil
		}

		// Persist the chosen context into base config.yaml
		if !cCtx.Bool("list") {
			// Verify context file exists
			if _, err := os.Stat(contextPath); os.IsNotExist(err) {
				return fmt.Errorf("this context does not exist, create it with `devkit avs context create %s`", context)
			}
			cfgPath := filepath.Join("config", common.BaseConfig)

			// synthesize a single project.context assignment
			items := []string{"project.context=" + context}
			doc, err := common.LoadYAML(cfgPath)
			if err != nil {
				return fmt.Errorf("read base config: %w", err)
			}
			root := doc.Content[0]
			cfgNode := common.GetChildByKey(root, "config")
			if cfgNode == nil {
				cfgNode = &yaml.Node{Kind: yaml.MappingNode}
				root.Content = append(
					root.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: "config"},
					cfgNode,
				)
			}
			for _, it := range items {
				parts := strings.SplitN(it, "=", 2)
				cfgNode, err = common.WriteToPath(cfgNode, strings.Split(parts[0], "."), parts[1])
				if err != nil {
					return fmt.Errorf("failed to set %s: %w", it, err)
				}
			}

			// Write the base config.yaml back to disk
			if err := common.WriteYAML(cfgPath, doc); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
			logger.Info("Global context successfully set to %s", context)
			return nil
		}

		// List the context
		contextPath = filepath.Join(contextDir, fmt.Sprintf("%s.yaml", context))
		err := common.ListYaml(contextPath, logger)
		if err != nil {
			return fmt.Errorf("this context does not exist, create it with `devkit avs context create %s`", context)
		}

		return nil
	},
}
