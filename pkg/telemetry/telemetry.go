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

// Properties represents the base properties for all events
type Properties struct {
	CLIVersion  string
	OS          string
	Arch        string
	ProjectUUID string
}

// NewProperties creates a new Properties instance
func NewProperties(cliVersion, os, arch, projectUUID string) Properties {
	return Properties{
		CLIVersion:  cliVersion,
		OS:          os,
		Arch:        arch,
		ProjectUUID: projectUUID,
	}
}

// WithContext returns a new context with the telemetry client
func WithContext(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}

// ClientFromContext retrieves the telemetry client from context
func ClientFromContext(ctx context.Context) (Client, bool) {
	client, ok := ctx.Value(contextKey{}).(Client)
	return client, ok
}

type contextKey struct{}
