package commands

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// CallCommand allows executing tasks in a running devnet
var CallCommand = &cli.Command{
	Name:  "call",
	Usage: "Call a task on a running devnet instance",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "task-name",
			Usage:    "Name of the task to execute",
			Required: true,
		},
		&cli.StringSliceFlag{
			Name:  "param",
			Usage: "Task parameters in key=value format (can be used multiple times)",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Get logger
		logger := common.LoggerFromContext(cCtx.Context)

		logger.DebugWithActor(iface.ActorAVSDev, "Testing AVS tasks...")

		// Task parameters
		taskName := cCtx.String("task-name")
		params := cCtx.StringSlice("param")

		// Validate parameters
		if taskName == "" {
			return fmt.Errorf("no task name specified")
		}

		paramMap := make(map[string]string)
		for _, param := range params {
			if param != "" {
				// Parse key=value format
				key, value, found := parseKeyValue(param)
				if !found {
					return fmt.Errorf("invalid param format: %s", param)
				}
				paramMap[key] = value
			}
		}

		if len(params) > 0 && len(paramMap) == 0 {
			return fmt.Errorf("no parameters supplied")
		}

		// Run the script from root of project dir
		const dir = ""

		// Set path for .devkit scripts
		scriptPath := filepath.Join(".devkit", "scripts", "call")

		// Set path for context yaml
		contextDir := filepath.Join("config", "contexts")
		yamlPath := path.Join(contextDir, "devnet.yaml")
		contextJSON, err := common.LoadRawContext(yamlPath)
		if err != nil {
			return fmt.Errorf("failed to load context: %w", err)
		}

		// Prepare call parameters
		callParams := map[string]interface{}{
			"task_name":  taskName,
			"parameters": paramMap,
		}

		// Convert parameters to JSON
		callParamsJSON, err := json.Marshal(callParams)
		if err != nil {
			return fmt.Errorf("failed to marshal call parameters: %w", err)
		}

		// Run call on the template call script
		if _, err := common.CallTemplateScript(cCtx.Context, logger, dir, scriptPath, common.ExpectNonJSONResponse, contextJSON, callParamsJSON); err != nil {
			return fmt.Errorf("call failed: %w", err)
		}

		logger.InfoWithActor(iface.ActorAVSDev, "Task execution completed successfully")

		return nil
	},
}

// parseKeyValue parses a key=value string
func parseKeyValue(param string) (key, value string, found bool) {
	for i, r := range param {
		if r == '=' {
			return param[:i], param[i+1:], true
		}
	}
	return "", "", false
}
