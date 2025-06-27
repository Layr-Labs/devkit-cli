package telemetry

import (
	"context"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
)

func TestNoopClient(t *testing.T) {
	client := NewNoopClient()
	if !IsNoopClient(client) {
		t.Error("Expected IsNoopClient to return true for NoopClient")
	}

	// Test AddMetric doesn't panic
	err := client.AddMetric(context.Background(), Metric{
		Name:       "test.metric",
		Value:      42,
		Dimensions: map[string]string{"test": "value"},
	})
	if err != nil {
		t.Errorf("AddMetric returned error: %v", err)
	}

	// Test Close doesn't panic
	err = client.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestContext(t *testing.T) {
	client := NewNoopClient()
	ctx := ContextWithClient(context.Background(), client)

	retrieved, ok := ClientFromContext(ctx)
	if !ok {
		t.Error("Failed to retrieve client from context")
	}
	if retrieved != client {
		t.Error("Retrieved client does not match original")
	}

	_, ok = ClientFromContext(context.Background())
	if ok {
		t.Error("Should not find client in empty context")
	}
}

func TestProperties(t *testing.T) {
	props := common.NewAppEnvironment("darwin", "amd64", "test-uuid", "user-uuid")
	if props.CLIVersion == "" {
		t.Error("Version not using default")
	}
	if props.OS != "darwin" {
		t.Error("OS mismatch")
	}
	if props.Arch != "amd64" {
		t.Error("Arch mismatch")
	}
	if props.ProjectUUID != "test-uuid" {
		t.Error("ProjectUUID mismatch")
	}
}
