package hooks

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/telemetry"

	"github.com/urfave/cli/v2"
)

// mockTelemetryClient is a test implementation of the telemetry.Client interface
type mockTelemetryClient struct {
	metrics []telemetry.Metric
}

func (m *mockTelemetryClient) AddMetric(_ context.Context, metric telemetry.Metric) error {
	m.metrics = append(m.metrics, metric)
	return nil
}

func (m *mockTelemetryClient) Close() error {
	return nil
}

// MockWithTelemetry is a test version of WithMetricEmission that uses a provided client
func MockWithTelemetry(action cli.ActionFunc, mockClient telemetry.Client) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Use the mock client directly instead of setupTelemetry
		ctx.Context = telemetry.ContextWithClient(ctx.Context, mockClient)

		// Create metrics context
		metrics := telemetry.NewMetricsContext()
		ctx.Context = telemetry.WithMetricsContext(ctx.Context, metrics)

		// Add base properties
		metrics.Properties["cli_version"] = ctx.App.Version
		metrics.Properties["os"] = runtime.GOOS
		metrics.Properties["arch"] = runtime.GOARCH
		metrics.Properties["project_uuid"] = "test-uuid"
		metrics.Properties["user_uuid"] = "user-uuid"

		// Add command flags as properties
		flags := collectFlagValues(ctx)
		for k, v := range flags {
			metrics.Properties[k] = fmt.Sprintf("%v", v)
		}

		// Track command invocation
		metrics.AddMetric("Count", 1)

		// Execute the wrapped action and capture result
		err := action(ctx)

		// emit metrics
		emitTelemetryMetrics(ctx, err)
		return err
	}
}

func TestAddMetric(t *testing.T) {
	// Create a mock telemetry client
	mockClient := &mockTelemetryClient{}

	// Create context with telemetry client
	ctx := context.Background()
	ctx = telemetry.ContextWithClient(ctx, mockClient)

	// Add a custom metric
	props := map[string]string{
		"direct_prop": "direct_value",
	}

	err := mockClient.AddMetric(ctx, telemetry.Metric{Name: "metricName", Value: 42, Dimensions: props})
	if err != nil {
		t.Fatalf("AddMetric returned error: %v", err)
	}

	// Verify metrics were tracked
	if len(mockClient.metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(mockClient.metrics))
	}

	metric := mockClient.metrics[0]
	if metric.Name != "metricName" {
		t.Errorf("Expected metric 'custom.event', got '%s'", metric.Name)
	}

	if metric.Value != 42 {
		t.Errorf("Expected value 42, got %f", metric.Value)
	}

	// Check dimensions
	if val, ok := metric.Dimensions["direct_prop"]; !ok || val != "direct_value" {
		t.Errorf("Direct property not correctly captured: %v", metric.Dimensions)
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

	// Use our mock version instead of the real WithMetricEmission
	wrappedAction := MockWithTelemetry(originalAction, mockClient)

	// Run the wrapped action
	err := wrappedAction(cliCtx)
	if err != nil {
		t.Fatalf("Wrapped action returned error: %v", err)
	}

	// Verify events were tracked (invoked and success)
	if len(mockClient.metrics) != 3 {
		t.Fatalf("Expected 3 metrics, got %d", len(mockClient.metrics))
	}

	// Check invoked event
	if mockClient.metrics[0].Name != "Count" {
		t.Errorf("Expected Count metric, got '%s'", mockClient.metrics[0].Name)
	}

	// Check success event
	if mockClient.metrics[1].Name != "Success" {
		t.Errorf("Expected success metric, got '%s'", mockClient.metrics[1].Name)
	}

	// Check duration event
	if mockClient.metrics[2].Name != "DurationMilliseconds" {
		t.Errorf("Expected duration metric, got '%s'", mockClient.metrics[2].Name)
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
	if len(mockClient.metrics) != 3 {
		t.Fatalf("Expected 2 metrics, got %d", len(mockClient.metrics))
	}

	// Check invoked event
	if mockClient.metrics[0].Name != "Count" {
		t.Errorf("Expected Count metric, got '%s'", mockClient.metrics[0].Name)
	}

	// Check fail event
	if mockClient.metrics[1].Name != "Failure" {
		t.Errorf("Expected fail metric, got '%s'", mockClient.metrics[1].Name)
	}

	// Check error message in event properties
	if val, ok := mockClient.metrics[1].Dimensions["error"]; !ok || val != "test error message" {
		t.Errorf("Error message not correctly captured: %v", mockClient.metrics[1].Dimensions)
	}

	// Check duration event
	if mockClient.metrics[2].Name != "DurationMilliseconds" {
		t.Errorf("Expected duration metric, got '%s'", mockClient.metrics[2].Name)
	}
}
