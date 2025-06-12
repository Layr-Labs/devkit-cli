package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// CreateContextCommand defines the "create context" subcommand
var CreateContextCommand = &cli.Command{
	Name:  "create",
	Usage: "Create a new context file template",
	Flags: append([]cli.Flag{
		&cli.BoolFlag{
			Name:  "force",
			Usage: "Force overwrite existing context file",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		// Get first positional argument as context name
		ctxName := cCtx.Args().First()
		if ctxName == "" {
			return fmt.Errorf("context name is required")
		}

		// Check if context directory exists
		contextDir := filepath.Join("config", "contexts")
		if err := os.MkdirAll(contextDir, 0755); err != nil {
			return fmt.Errorf("failed to create contexts directory: %w", err)
		}

		// Set context file path
		ctxPath := filepath.Join(contextDir, ctxName+".yaml")

		// create if missing or forced
		if _, err := os.Stat(ctxPath); err != nil || cCtx.Bool("force") {
			logger.InfoWithActor(iface.ActorConfig, "Creating a new context for %s", ctxName)
			if err := CreateContext(ctxPath, ctxName); err != nil {
				return fmt.Errorf("failed to create new context: %w", err)
			}
		} else {
			return fmt.Errorf("context %s already exists (use --force to overwrite)", ctxName)
		}

		logger.InfoWithActor(iface.ActorConfig, "Context successfully created at %s", ctxPath)
		logger.InfoWithActor(iface.ActorConfig, "")
		logger.InfoWithActor(iface.ActorConfig, "  - To view your new context call: `devkit avs context --list %s`", ctxName)
		logger.InfoWithActor(iface.ActorConfig, "  - To edit your new context call: `devkit avs context --edit %s`", ctxName)
		logger.InfoWithActor(iface.ActorConfig, "")
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
