package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// CreateCommand defines the "create context" subcommand
var CreateContextCommand = &cli.Command{
	Name:  "create",
	Usage: "Create a new context",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to work over",
		},
		&cli.BoolFlag{
			Name:  "force",
			Usage: "Force context to be overwritten",
		},
	},
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		ctxName := cCtx.String("context")
		if args := cCtx.Args().Slice(); len(args) > 0 {
			ctxName = args[0]
		}

		// path + ensure dir
		ctxPath := filepath.Join("config", "contexts", fmt.Sprintf("%s.yaml", ctxName))
		if err := os.MkdirAll(filepath.Dir(ctxPath), 0755); err != nil {
			return fmt.Errorf("failed to make contexts dir: %w", err)
		}

		// create if missing or forced
		if _, err := os.Stat(ctxPath); err != nil || cCtx.Bool("force") {
			logger.Info("Creating a new context for %s", ctxName)
			if err := CreateContext(ctxPath, ctxName); err != nil {
				return fmt.Errorf("failed to create new context: %w", err)
			}
		} else {
			return fmt.Errorf("context already exists, if you want to recreate try `devkit avs context create --force %s`", ctxName)
		}

		logger.Info("Context successfully created at %s", ctxPath)
		logger.Info("")
		logger.Info("  - To view your new context call: `devkit avs context --list %s`", ctxName)
		logger.Info("  - To edit your new context call: `devkit avs context --edit %s`", ctxName)
		logger.Info("")
		return nil
	},
}

func CreateContext(contextPath, context string) error {
	// Pull the latest context and set name
	content := contexts.ContextYamls[contexts.LatestVersion]
	entryName := fmt.Sprintf("%s.yaml", context)

	// Place the context name in place
	contentString := strings.ReplaceAll(string(content), "devnet", context)

	// Write the new context
	err := os.WriteFile(contextPath, []byte(contentString), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", entryName, err)
	}

	return nil
}
