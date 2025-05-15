package telemetry

import (
	"context"
	devcontext "devkit-cli/pkg/context"
	"github.com/posthog/posthog-go"
	"gopkg.in/yaml.v3"
	"os"
)

// PostHogClient implements the Client interface using PostHog
type PostHogClient struct {
	client         posthog.Client
	appEnvironment *devcontext.AppEnvironment
}

// NewPostHogClient creates a new PostHog client
func NewPostHogClient(environment *devcontext.AppEnvironment) (*PostHogClient, error) {
	apiKey := getPostHogAPIKey()
	if apiKey == "" {
		// No API key available, return noop client without error
		return nil, nil
	}
	client, err := posthog.NewWithConfig(apiKey, posthog.Config{Endpoint: getPostHogEndpoint()})
	if err != nil {
		return nil, err
	}

	return &PostHogClient{
		client:         client,
		appEnvironment: environment,
	}, nil
}

// AddMetric implements the Client interface
func (c *PostHogClient) AddMetric(ctx context.Context, metric Metric) error {
	if c == nil || c.client == nil {
		return nil
	}

	// Create properties map starting with base properties
	props := make(map[string]interface{})
	// Add metric value
	props["value"] = metric.Value

	// Add metric dimensions
	for k, v := range metric.Dimensions {
		props[k] = v
	}

	// Never return errors from telemetry operations
	_ = c.client.Enqueue(posthog.Capture{
		DistinctId: c.appEnvironment.ProjectUUID,
		Event:      metric.Name,
		Properties: props,
	})
	return nil
}

// Close implements the Client interface
func (c *PostHogClient) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	// Ignore any errors from Close operations
	_ = c.client.Close()
	return nil
}

// Embedded API key, can be set during build time
var embeddedPostHogAPIKey string

func getPostHogAPIKey() string {
	// Priority order:
	// 1. Environment variable
	// 2. Project config file
	// 3. Embedded key (set at build time)

	// Check environment variable first
	if key := os.Getenv("DEVKIT_POSTHOG_KEY"); key != "" {
		return key
	}

	// Check project config file next
	// Use a direct import to avoid circular dependencies
	configPath := ".config.devkit.yml"
	data, err := os.ReadFile(configPath)
	if err == nil {
		// Simple YAML parsing to extract just the key we need
		var config struct {
			PostHogAPIKey string `yaml:"posthog_api_key"`
		}
		if yaml.Unmarshal(data, &config) == nil && config.PostHogAPIKey != "" {
			return config.PostHogAPIKey
		}
	}

	// Finally, check embedded key
	return embeddedPostHogAPIKey
}

func getPostHogEndpoint() string {
	if endpoint := os.Getenv("DEVKIT_POSTHOG_ENDPOINT"); endpoint != "" {
		return endpoint
	}
	return "https://us.i.posthog.com"
}
