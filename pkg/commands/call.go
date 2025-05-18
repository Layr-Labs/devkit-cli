package commands

import (
	"devkit-cli/pkg/common"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

// CallCommand defines the "call" command
var CallCommand = &cli.Command{
	Name:  "call",
	Usage: "Submits tasks to the local devnet, triggers off-chain execution, and aggregates results",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "params",
			Usage:    "parameters for the call (e.g., payload=\"<payload>\")",
			Required: true,
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Get logger
		log, _ := common.GetLogger()

		// Print task if verbose
		if cCtx.Bool("verbose") {
			log.Info("Testing AVS tasks...")
		}

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

		// Extract params from flag
		paramsStr := cCtx.String("params")
		paramsMap, err := parseParams(paramsStr)
		if err != nil {
			return err
		}
		paramsJSON, err := json.Marshal(paramsMap)
		if err != nil {
			return err
		}

		// Run init on the template init script
		const expectJSONResponse = true
		if _, err := common.CallTemplateScript(cCtx.Context, dir, scriptPath, expectJSONResponse, contextJSON, paramsJSON); err != nil {
			return fmt.Errorf("call failed: %w", err)
		}

		log.Info("Task execution completed successfully")
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
