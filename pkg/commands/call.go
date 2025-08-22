package commands

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"

	"github.com/urfave/cli/v2"
)

// CallCommand defines the "call" command
var CallCommand = &cli.Command{
	Name:  "call",
	Usage: "Submits tasks to the local devnet, triggers off-chain execution, and aggregates results",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
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
			cmdParams := reconstructCommandParams(cCtx.Args().Slice())

			return fmt.Errorf(
				"call failed: `devkit avs call` only available on devnet - please run `devkit avs call --context devnet %s`",
				cmdParams,
			)
		}

		// Print task if verbose
		logger.Debug("Testing AVS tasks...")

		// Check that args are provided
		parts := cCtx.Args().Slice()
		if len(parts) == 0 {
			return fmt.Errorf("no parameters supplied")
		}

		// Run scriptPath from cwd
		const dir = ""

		// Set path for .devkit scripts
		scriptPath := filepath.Join(".devkit", "scripts", "call")

		// Parse the params from the provided args
		paramsMap, err := parseParams(strings.Join(parts, " "))
		if err != nil {
			return err
		}
		paramsJSON, err := json.Marshal(paramsMap)
		if err != nil {
			return err
		}

		// Run init on the template init script
		if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectNonJSONResponse, contextJSON, paramsJSON); err != nil {
			return fmt.Errorf("call failed: %w", err)
		}

		logger.Info("Task execution completed successfully")
		return nil
	},
}

func parseParams(input string) (map[string]string, error) {
	result := make(map[string]string)
	pairs := strings.Fields(input)

	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid param: %s", pair)
		}
		key := kv[0]
		val := strings.Trim(kv[1], `"'`)
		result[key] = val
	}

	return result, nil
}

func reconstructQuotes(val string) string {
	if strings.Contains(val, `"`) {
		return "'" + val + "'"
	}
	return `"` + val + `"`
}

func reconstructCommandParams(argv []string) string {
	var out []string
	for _, arg := range argv {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) == 2 {
			k, v := parts[0], parts[1]
			out = append(out, fmt.Sprintf("%s=%s", k, reconstructQuotes(v)))
		} else {
			out = append(out, arg)
		}
	}
	return strings.Join(out, " ")
}
