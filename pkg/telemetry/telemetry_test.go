package telemetry

import (
	"context"
	"testing"
)

func TestNoopClient(t *testing.T) {
	client := NewNoopClient()
	ctx := context.Background()

	if err := client.Track(ctx, "test", nil); err != nil {
		t.Errorf("Track returned error: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Errorf("Close returned error: %v", err)
	}
}

func TestContext(t *testing.T) {
	client := NewNoopClient()
	ctx := WithContext(context.Background(), client)

	retrieved, ok := FromContext(ctx)
	if !ok {
		t.Error("Failed to retrieve client from context")
	}
	if retrieved != client {
		t.Error("Retrieved client does not match original")
	}

	_, ok = FromContext(context.Background())
	if ok {
		t.Error("Should not find client in empty context")
	}
}

func TestProperties(t *testing.T) {
	props := NewProperties("1.0.0", "darwin", "amd64", "test-uuid")
	if props.CLIVersion != "1.0.0" {
		t.Error("CLIVersion mismatch")
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
