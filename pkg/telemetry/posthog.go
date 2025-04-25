package telemetry

import (
	"context"
	"os"
	"time"

	"github.com/posthog/posthog-go"
	"gopkg.in/yaml.v3"
)

// PostHogClient implements the Client interface using PostHog
type PostHogClient struct {
	client posthog.Client
	props  Properties
}

// NewPostHogClient creates a new PostHog client
func NewPostHogClient(props Properties) (*PostHogClient, error) {
	apiKey := getPostHogAPIKey()
	if apiKey == "" {
		// No API key available, return noop client without error
		return nil, nil
	}

	client, err := posthog.NewWithConfig(apiKey, posthog.Config{
		Endpoint: getPostHogEndpoint(),
		Interval: 30 * time.Second,
	})
	if err != nil {
		// Error creating client - return nil without error to allow fallback
		return nil, nil
	}

	return &PostHogClient{
		client: client,
		props:  props,
	}, nil
}

// Track implements the Client interface
func (c *PostHogClient) Track(ctx context.Context, event string, props map[string]interface{}) error {
	if c == nil || c.client == nil {
		return nil
	}

	mergedProps := make(map[string]interface{})
	mergedProps["cli_version"] = c.props.CLIVersion
	mergedProps["os"] = c.props.OS
	mergedProps["arch"] = c.props.Arch
	mergedProps["project_uuid"] = c.props.ProjectUUID

	for k, v := range props {
		mergedProps[k] = v
	}

	// Never return errors from telemetry operations
	_ = c.client.Enqueue(posthog.Capture{
		DistinctId: c.props.ProjectUUID,
		Event:      event,
		Properties: mergedProps,
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
	return "https://app.posthog.com"
}
