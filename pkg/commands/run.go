package commands

import (
	"fmt"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// RunCommand defines the "run" command
var RunCommand = &cli.Command{
	Name:  "run",
	Usage: "Start offchain AVS components",
	Flags: append([]cli.Flag{}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Invoke and return AVSRun
		return AVSRun(cCtx)
	},
}

func AVSRun(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)

	// Print task if verbose
	logger.Debug("Starting offchain AVS components...")

	// Run the script from root of project dir
	// (@TODO (GD): this should always be the root of the project, but we need to do this everywhere (ie reading ctx/config etc))
	const dir = ""

	// Set path for .devkit scripts
	scriptPath := filepath.Join(".devkit", "scripts", "run")

	// Set path for context yaml
	contextJSON, err := common.LoadRawContext("devnet") // @TODO: use selected context name
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// Run init on the template init script
	if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectNonJSONResponse, contextJSON); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	logger.Info("Offchain AVS components started successfully!")

	return nil
}
