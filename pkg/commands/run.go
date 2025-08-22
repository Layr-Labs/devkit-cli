package commands

import (
	"fmt"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"

	"github.com/urfave/cli/v2"
)

// RunCommand defines the "run" command
var RunCommand = &cli.Command{
	Name:  "run",
	Usage: "Start offchain AVS components",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Invoke and return AVSRun
		return AVSRun(cCtx)
	},
}

func AVSRun(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx)

	// Check for flagged contextName
	contextName := cCtx.String("context")

	// Set path for context yaml
	var err error
	var contextJSON []byte
	if contextName == "" {
		contextJSON, contextName, err = common.LoadDefaultRawContext()
	} else {
		contextJSON, contextName, err = common.LoadRawContext(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// Prevent runs when context is not devnet
	if contextName != devnet.DEVNET_CONTEXT {
		return fmt.Errorf("run failed: `devkit avs run` only available on devnet - please run `devkit avs run --context devnet`")
	}

	// Print task if verbose
	logger.Debug("Starting offchain AVS components...")

	// Load the config fetch templateLanguage
	cfg, _, err := common.LoadConfigWithContextConfig(contextName)
	if err != nil {
		return err
	}

	// Pull template language from config
	language := cfg.Config.Project.TemplateLanguage
	if language == "" {
		language = "go"
	}

	// Log the type of project being ran
	logger.Info("Running %s AVS project", language)

	// Run the script from root of project dir
	// (@TODO (GD): this should always be the root of the project, but we need to do this everywhere (ie reading ctx/config etc))
	const dir = ""

	// Set path for .devkit scripts
	scriptPath := filepath.Join(".devkit", "scripts", "run")

	// Run init on the template init script
	if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectNonJSONResponse, contextJSON, []byte(language)); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	logger.Info("Offchain AVS components started successfully!")

	return nil
}
