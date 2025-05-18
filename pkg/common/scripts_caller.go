package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func CallTemplateScript(cmdCtx context.Context, scriptPath string, expectJSONResponse bool, params ...[]byte) (map[string]interface{}, error) {
	// Get logger
	log, _ := GetLogger()

	// Convert byte params to strings
	stringParams := make([]string, len(params))
	for i, b := range params {
		stringParams[i] = string(b)
	}

	// Prepare the command
	var stdout bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, scriptPath, stringParams...)
	cmd.Dir = ""
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	// Exec the command
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("script %s exited with code %d", scriptPath, exitErr.ExitCode())
		}
		return nil, fmt.Errorf("failed to run script %s: %w", scriptPath, err)
	}

	// Clean and validate stdout
	raw := bytes.TrimSpace(stdout.Bytes())
	if len(raw) == 0 {
		log.Warn("Empty output from %s; returning empty result", scriptPath)
		return map[string]interface{}{}, nil
	}

	// Return the result as JSON if expected
	if expectJSONResponse {
		var result map[string]interface{}
		if err := json.Unmarshal(raw, &result); err != nil {
			log.Warn("Invalid or non-JSON script output: %s; returning empty result: %v", string(raw), err)
			return map[string]interface{}{}, nil
		}
		return result, nil
	}

	// Log the raw stdout
	log.Info("%s", string(raw))

	return nil, nil
}
