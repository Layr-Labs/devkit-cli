package common

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
)

func TestCallTemplateScript(t *testing.T) {
	logger := logger.NewNoopLogger()
	// JSON response case
	scriptJSON := `#!/bin/bash
input=$1
echo '{"status": "ok", "received": '"$input"'}'`

	tmpDir := t.TempDir()
	jsonScriptPath := filepath.Join(tmpDir, "json_echo.sh")
	if err := os.WriteFile(jsonScriptPath, []byte(scriptJSON), 0755); err != nil {
		t.Fatalf("failed to write JSON test script: %v", err)
	}

	// Parse the provided params
	inputJSON, err := json.Marshal(map[string]interface{}{"context": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		t.Fatalf("marshal context: %v", err)
	}

	// Run the json_echo script
	out, err := CallTemplateScript(context.Background(), logger, "", jsonScriptPath, ExpectJSONResponse, inputJSON)
	if err != nil {
		t.Fatalf("CallTemplateScript (JSON) failed: %v", err)
	}

	// Assert known structure
	if out["status"] != "ok" {
		t.Errorf("expected status ok, got %v", out["status"])
	}

	received, ok := out["received"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map under 'received'")
	}

	expected := map[string]interface{}{"foo": "bar"}
	if !reflect.DeepEqual(received["context"], expected) {
		t.Errorf("expected context %v, got %v", expected, received["context"])
	}

	// Non-JSON response case
	scriptText := `#!/bin/bash
echo "This is plain text output"`

	textScriptPath := filepath.Join(tmpDir, "text_echo.sh")
	if err := os.WriteFile(textScriptPath, []byte(scriptText), 0755); err != nil {
		t.Fatalf("failed to write text test script: %v", err)
	}

	// Run the text_echo script
	out, err = CallTemplateScript(context.Background(), logger, "", textScriptPath, ExpectNonJSONResponse)
	if err != nil {
		t.Fatalf("CallTemplateScript (non-JSON) failed: %v", err)
	}
	if out != nil {
		t.Errorf("expected nil output for non-JSON response, got: %v", out)
	}

	// Empty response case
	empty := `#!/bin/bash
exit 0`

	emptyPath := filepath.Join(tmpDir, "empty.sh")
	if err := os.WriteFile(emptyPath, []byte(empty), 0755); err != nil {
		t.Fatalf("failed to write empty test script: %v", err)
	}

	// Run the empty script expecting JSON (this should generate a warning)
	out, err = CallTemplateScript(context.Background(), logger, "", emptyPath, ExpectJSONResponse)
	if err != nil {
		t.Fatalf("CallTemplateScript (empty JSON) failed: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty map for empty JSON response, got: %v", out)
	}

	// Check logger buffer for warning instead of capturing stdout
	if !logger.Contains("returning empty result") {
		t.Errorf("expected warning 'returning empty result' in logger buffer, but not found")
	}
}
