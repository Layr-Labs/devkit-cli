package commands

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// CallCommand defines the "call" command
var CallCommand = &cli.Command{
	Name:  "call",
	Usage: "Submits tasks to the local devnet, triggers off-chain execution, and aggregates results",
	Flags: common.GlobalFlags,
	Action: func(cCtx *cli.Context) error {
		// Get logger
		logger, _ := common.GetLogger(cCtx.Bool("verbose"))

		logger.Debug("Testing AVS tasks...")

		// Set path for context yaml
		contextDir := filepath.Join("config", "contexts")
		yamlPath := path.Join(contextDir, "devnet.yaml") // @TODO: use selected context name
		contextJSON, err := common.LoadContext(yamlPath)
		if err != nil {
			return fmt.Errorf("failed to load context %w", err)
		}

		// Run scriptPath from cwd
		const dir = ""

		// Set path for .devkit scripts
		scriptPath := filepath.Join(".devkit", "scripts", "call")

		// Check that args are provided
		parts := cCtx.Args().Slice()
		if len(parts) == 0 {
			return fmt.Errorf("no parameters supplied")
		}

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
		if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectJSONResponse, contextJSON, paramsJSON); err != nil {
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
