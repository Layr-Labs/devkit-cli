package hooks

import (
	"context"
	"errors"
	"testing"
	"time"

	"devkit-cli/pkg/telemetry"

	"github.com/urfave/cli/v2"
)

// mockTelemetryClient is a test implementation of the telemetry.Client interface
type mockTelemetryClient struct {
	events []mockEvent
}

type mockEvent struct {
	name  string
	props map[string]interface{}
}

func (m *mockTelemetryClient) Track(_ context.Context, event string, props map[string]interface{}) error {
	m.events = append(m.events, mockEvent{
		name:  event,
		props: props,
	})
	return nil
}

func (m *mockTelemetryClient) Close() error {
	return nil
}

// MockWithTelemetry is a test version of WithTelemetry that uses a provided client
func MockWithTelemetry(action cli.ActionFunc, mockClient telemetry.Client) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Use the mock client directly instead of setupTelemetry
		ctx.Context = telemetry.WithContext(ctx.Context, mockClient)

		// Collect flags and store metrics
		flags := collectFlagValues(ctx)
		startTime := time.Now()

		// Store command metrics in context
		metrics := CommandMetrics{
			StartTime: startTime,
			Command:   command,
			Flags:     flags,
		}
		ctx.Context = WithCommandMetrics(ctx.Context, metrics)

		// Track command invocation
		_ = Track(ctx.Context, FormatEventName(command, "invoked"), flags)

		// Execute the wrapped action and capture result
		err := action(ctx)

		// Track result based on error
		if err != nil {
			trackCommandResult(ctx, "fail", err)
		} else {
			trackCommandResult(ctx, "success", nil)
		}

		return err
	}
}

func TestWithTelemetry(t *testing.T) {
	// Create a mock telemetry client
	mockClient := &mockTelemetryClient{}

	// Create a CLI context
	app := &cli.App{Name: "testapp"}
	cliCtx := cli.NewContext(app, nil, nil)

	// Properly set up the Command
	command := &cli.Command{Name: "test-command"}
	cliCtx.Command = command

	// Create context
	ctx := context.Background()
	cliCtx.Context = ctx

	// Create a wrapped action
	originalAction := func(ctx *cli.Context) error {
		return nil
	}

	// Use our mock version instead of the real WithTelemetry
	wrappedAction := MockWithTelemetry(originalAction, mockClient)

	// Run the wrapped action
	err := wrappedAction(cliCtx)
	if err != nil {
		t.Fatalf("Wrapped action returned error: %v", err)
	}

	// Verify events were tracked (invoked and success)
	if len(mockClient.events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(mockClient.events))
	}

	// Check invoked event
	if mockClient.events[0].name != FormatEventName("test-command", "invoked") {
		t.Errorf("Expected invoked event, got '%s'", mockClient.events[0].name)
	}

	// Check success event
	if mockClient.events[1].name != FormatEventName("test-command", "success") {
		t.Errorf("Expected success event, got '%s'", mockClient.events[1].name)
	}
}

func TestWithTelemetryError(t *testing.T) {
	// Create a mock telemetry client
	mockClient := &mockTelemetryClient{}

	// Create a CLI context
	app := &cli.App{Name: "testapp"}
	cliCtx := cli.NewContext(app, nil, nil)

	// Set up the command
	command := &cli.Command{Name: "test-command"}
	cliCtx.Command = command

	// Create context
	ctx := context.Background()
	cliCtx.Context = ctx

	// Create a wrapped action that returns an error
	testErr := errors.New("test error message")
	originalAction := func(ctx *cli.Context) error {
		return testErr
	}

	// Use our mock version
	wrappedAction := MockWithTelemetry(originalAction, mockClient)

	// Run the wrapped action
	err := wrappedAction(cliCtx)
	if err != testErr {
		t.Fatalf("Expected wrapped action to return the original error")
	}

	// Verify events were tracked (invoked and fail)
	if len(mockClient.events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(mockClient.events))
	}

	// Check invoked event
	if mockClient.events[0].name != FormatEventName("test-command", "invoked") {
		t.Errorf("Expected invoked event, got '%s'", mockClient.events[0].name)
	}

	// Check fail event
	if mockClient.events[1].name != FormatEventName("test-command", "fail") {
		t.Errorf("Expected fail event, got '%s'", mockClient.events[1].name)
	}

	// Check error message in event properties
	if val, ok := mockClient.events[1].props["error"]; !ok || val != "test error message" {
		t.Errorf("Error message not correctly captured: %v", mockClient.events[1].props)
	}
}

func TestTrack(t *testing.T) {
	// Create a mock telemetry client
	mockClient := &mockTelemetryClient{}

	// Create context with telemetry client
	ctx := context.Background()
	ctx = telemetry.WithContext(ctx, mockClient)

	// Track a custom metric
	props := map[string]interface{}{
		"direct_prop": "direct_value",
	}

	err := Track(ctx, "custom.event", props)
	if err != nil {
		t.Fatalf("Track returned error: %v", err)
	}

	// Verify events were tracked
	if len(mockClient.events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(mockClient.events))
	}

	event := mockClient.events[0]
	if event.name != "custom.event" {
		t.Errorf("Expected event 'custom.event', got '%s'", event.name)
	}

	// Check properties
	if val, ok := event.props["direct_prop"]; !ok || val != "direct_value" {
		t.Errorf("Direct property not correctly captured: %v", event.props)
	}
}

func TestFormatEventName(t *testing.T) {
	tests := []struct {
		command  string
		action   string
		expected string
	}{
		{"avs_create", "invoked", "cli.avs_avs_create.invoked"},
		{"avs_run", "task_completed", "cli.avs_avs_run.task_completed"},
		{"build", "failed_step", "cli.avs_build.failed_step"},
	}

	for _, test := range tests {
		result := FormatEventName(test.command, test.action)
		if result != test.expected {
			t.Errorf("FormatEventName(%s, %s) = %s, want %s",
				test.command, test.action, result, test.expected)
		}
	}
}
