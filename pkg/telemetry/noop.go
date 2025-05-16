package telemetry

import "context"

// NoopClient implements the Client interface with no-op methods
type NoopClient struct{}

// NewNoopClient creates a new no-op client
func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

// AddMetric implements the Client interface
func (c *NoopClient) AddMetric(_ context.Context, _ Metric) error {
	return nil
}

// Close implements the Client interface
func (c *NoopClient) Close() error {
	return nil
}

// IsNoopClient checks if the client is a NoopClient (disabled telemetry)
func IsNoopClient(client Client) bool {
	_, isNoop := client.(*NoopClient)
	return isNoop
}
