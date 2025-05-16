package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

func RunTemplateScript(cmdCtx context.Context, scriptPath string, params []byte) (map[string]interface{}, error) {
	// Get logger
	log, _ := GetLogger()

	// Prepare the command
	var stdout bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, scriptPath, string(params))
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	// Exec the command
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("Call to %s failed: %w", scriptPath, err)
	}

	// Clean and validate stdout
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		log.Warn("Empty output from %s; returning empty result", scriptPath)
		return map[string]interface{}{}, nil
	}

	// Collect the result as JSON
	var result map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		log.Warn("Invalid or non-JSON script output; returning empty result")
		return map[string]interface{}{}, nil
	}

	return result, nil
}
