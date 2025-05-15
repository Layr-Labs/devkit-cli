package telemetry

import (
	"context"
)

// Client defines the interface for telemetry operations
type Client interface {
	// AddMetric emits a single metric
	AddMetric(ctx context.Context, metric Metric) error
	// Close cleans up any resources
	Close() error
}

// ContextWithClient returns a new context with the telemetry client
func ContextWithClient(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}

// ClientFromContext retrieves the telemetry client from context
func ClientFromContext(ctx context.Context) (Client, bool) {
	client, ok := ctx.Value(contextKey{}).(Client)
	return client, ok
}

type contextKey struct{}
