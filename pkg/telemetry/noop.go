package telemetry

import "context"

// NoopClient implements the Client interface with no-op methods
type NoopClient struct{}

// NewNoopClient creates a new no-op client
func NewNoopClient() *NoopClient {
	return &NoopClient{}
}

// Track implements the Client interface
func (c *NoopClient) Track(_ context.Context, _ string, _ map[string]interface{}) error {
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
