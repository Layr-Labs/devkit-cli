package telemetry

import (
	"context"
)

// Embedded devkit telemetry api key from release
var embeddedTelemetryApiKey string

// Client defines the interface for telemetry operations
type Client interface {
	// AddMetric emits a single metric
	AddMetric(ctx context.Context, metric Metric) error
	// Close cleans up any resources
	Close() error
}

type clientContextKey struct{}

// ContextWithClient returns a new context with the telemetry client
func ContextWithClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, clientContextKey{}, client)
}

// ClientFromContext retrieves the telemetry client from context
func ClientFromContext(ctx context.Context) (Client, bool) {
	client, ok := ctx.Value(clientContextKey{}).(Client)
	return client, ok
}
