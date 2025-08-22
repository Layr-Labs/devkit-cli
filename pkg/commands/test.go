package commands

import (
	"fmt"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// TestCommand defines the "test" command
var TestCommand = &cli.Command{
	Name:  "test",
	Usage: "Run AVS tests",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Invoke and return AVSTest
		return AVSTest(cCtx)
	},
}

func AVSTest(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx)

	// Print task if verbose
	logger.Debug("Running AVS tests...")

	// Run the script from root of project dir
	const dir = ""

	// Set path for .devkit scripts
	scriptPath := filepath.Join(".devkit", "scripts", "test")

	// Check for flagged contextName
	contextName := cCtx.String("context")

	// Set path for context yaml
	var err error
	var contextJSON []byte
	if contextName == "" {
		contextJSON, _, err = common.LoadDefaultRawContext()
	} else {
		contextJSON, _, err = common.LoadRawContext(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}

	// Run test on the template test script
	if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectNonJSONResponse, contextJSON); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	logger.Info("AVS tests completed successfully!")

	return nil
}
