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
	// Create temporary shell script
	script := `#!/bin/bash
input=$1
echo '{"status": "ok", "received": '"$input"'}'`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "echo.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	// Parse the provided params
	inputJSON, err := json.Marshal(map[string]interface{}{"context": map[string]interface{}{"foo": "bar"}})
	if err != nil {
		t.Fatalf("marshal context: %v", err)
	}

	// Run the template script
	const expectJSONResponse = true
	out, err := CallTemplateScript(context.Background(), scriptPath, expectJSONResponse, inputJSON)
	if err != nil {
		t.Fatalf("CallTemplateScript failed: %v", err)
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
}
