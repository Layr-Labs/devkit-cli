package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

type ResponseExpectation int

const (
	ExpectNonJSONResponse ResponseExpectation = iota
	ExpectJSONResponse
)

func CallTemplateScript(cmdCtx context.Context, logger iface.Logger, dir string, scriptPath string, expect ResponseExpectation, params ...[]byte) (map[string]interface{}, error) {
	// Convert byte params to strings
	stringParams := make([]string, len(params))
	for i, b := range params {
		stringParams[i] = string(b)
	}

	// Prepare the command
	var stdout bytes.Buffer
	cmd := exec.CommandContext(cmdCtx, scriptPath, stringParams...)
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	// Run the command in its own group
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// When context is canceled, forward SIGINT (but only if the process is running)
	go func() {
		<-cmdCtx.Done()
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		}
	}()

	// Exec the command
	if err := cmd.Run(); err != nil {
		// if it’s an ExitError, check if it was killed by a signal
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			if ws, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				// killed by signal -> treat as cancellation
				return nil, cmdCtx.Err()
			}
			// nonzero exit code
			return nil, fmt.Errorf("script %s exited with code %d", scriptPath, exitErr.ExitCode())
		}
		return nil, fmt.Errorf("failed to run script %s: %w", scriptPath, err)
	}

	// Clean and validate stdout
	raw := bytes.TrimSpace(stdout.Bytes())

	// Return the result as JSON if expected
	if expect == ExpectJSONResponse {
		// End early for empty response
		if len(raw) == 0 {
			logger.Warn("Empty output from %s; returning empty result", scriptPath)
			return map[string]interface{}{}, nil
		}

		// Unmarshal response and return unless err
		var result map[string]interface{}
		if err := json.Unmarshal(raw, &result); err != nil {
			logger.Warn("Invalid or non-JSON script output: %s; returning empty result: %v", string(raw), err)
			return map[string]interface{}{}, nil
		}
		return result, nil
	}

	// Log the raw stdout
	if len(raw) > 0 {
		logger.Info("%s", string(raw))
	}

	return nil, nil
}
