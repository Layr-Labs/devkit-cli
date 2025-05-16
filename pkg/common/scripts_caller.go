package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func RunTemplateScript(cmdCtx context.Context, scriptPath string, context map[string]interface{}) (map[string]interface{}, error) {
	inputJSON, err := json.Marshal(map[string]interface{}{"context": context})
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	var stdout bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, scriptPath, string(inputJSON))
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("deployContracts failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("invalid JSON output: %w", err)
	}
	return result, nil
}
