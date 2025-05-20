package common

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCallTemplateScript(t *testing.T) {
	logger, _ := GetLogger(false)
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
}
