package hooks

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"devkit-cli/pkg/common"
	"devkit-cli/pkg/telemetry"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// EnvFile is the name of the environment file
const EnvFile = ".env"

// contextKey is used to store command metrics in context
type contextKey struct{}

// CommandPrefix is the prefix to apply to all command names
const CommandPrefix = "avs_"

func getFlagValue(ctx *cli.Context, name string) interface{} {
	if !ctx.IsSet(name) {
		return nil
	}

	if ctx.Bool(name) {
		return ctx.Bool(name)
	}
	if ctx.String(name) != "" {
		return ctx.String(name)
	}
	if ctx.Int(name) != 0 {
		return ctx.Int(name)
	}
	if ctx.Float64(name) != 0 {
		return ctx.Float64(name)
	}
	return nil
}

func collectFlagValues(ctx *cli.Context) map[string]interface{} {
	flags := make(map[string]interface{})

	// App-level flags
	for _, flag := range ctx.App.Flags {
		flagName := flag.Names()[0]
		if ctx.IsSet(flagName) {
			flags[flagName] = getFlagValue(ctx, flagName)
		}
	}

	// Command-level flags
	for _, flag := range ctx.Command.Flags {
		flagName := flag.Names()[0]
		if ctx.IsSet(flagName) {
			flags[flagName] = getFlagValue(ctx, flagName)
		}
	}

	return flags
}

func setupTelemetry(ctx *cli.Context, command string) telemetry.Client {
	if command != "create" && !common.IsTelemetryEnabled() {
		return telemetry.NewNoopClient()
	}

	// Try to create active client
	props := telemetry.NewProperties(
		ctx.App.Version,
		runtime.GOOS,
		runtime.GOARCH,
		common.GetProjectUUID(),
	)

	phClient, _ := telemetry.NewPostHogClient(props)
	if phClient != nil {
		return phClient
	}

	// no client available, return noop client which means telemetry is disabled
	return telemetry.NewNoopClient()
}

func FormatMetricName(command, action string) string {
	if strings.Contains(action, ".") {
		return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
	}
	return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
}

// CommandMiddleware wraps a command with pre-processing and post-processing steps
type CommandMiddleware struct {
	PreProcessors  []cli.ActionFunc
	PostProcessors []cli.ActionFunc
}

// NewCommandMiddleware creates a new command middleware
func NewCommandMiddleware() *CommandMiddleware {
	return &CommandMiddleware{
		PreProcessors:  make([]cli.ActionFunc, 0),
		PostProcessors: make([]cli.ActionFunc, 0),
	}
}

// AddPreProcessor adds a pre-processing step
func (m *CommandMiddleware) AddPreProcessor(processor cli.ActionFunc) {
	m.PreProcessors = append(m.PreProcessors, processor)
}

// AddPostProcessor adds a post-processing step
func (m *CommandMiddleware) AddPostProcessor(processor cli.ActionFunc) {
	m.PostProcessors = append(m.PostProcessors, processor)
}

// Wrap wraps a command action with all pre-processors and post-processors
func (m *CommandMiddleware) Wrap(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Run all pre-processors in order
		for _, pre := range m.PreProcessors {
			if err := pre(ctx); err != nil {
				return err
			}
		}

		// Run the main action
		err := action(ctx)

		// Run all post-processors in order
		for _, post := range m.PostProcessors {
			// We don't want to stop post-processing if one fails
			_ = post(ctx)
		}

		return err
	}
}

// ApplyMiddleware applies a list of middleware functions to commands
func ApplyMiddleware(commands []*cli.Command, middleware *CommandMiddleware) {
	for _, cmd := range commands {
		// Apply middleware to this command's action if it exists
		if cmd.Action != nil {
			// Store original action
			originalAction := cmd.Action

			// Wrap with middleware
			cmd.Action = middleware.Wrap(originalAction)
		}

		// Recursively apply to subcommands
		if len(cmd.Subcommands) > 0 {
			ApplyMiddleware(cmd.Subcommands, middleware)
		}
	}
}

// WithTelemetryPreProcessor creates a pre-processor that sets up telemetry
func WithTelemetryPreProcessor() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Get telemetry client
		client := setupTelemetry(ctx, command)
		ctx.Context = telemetry.WithContext(ctx.Context, client)

		// Create metrics context
		metrics := telemetry.NewMetricsContext(ctx.App.Name, command)
		ctx.Context = telemetry.WithMetricsContext(ctx.Context, metrics)

		// Add base properties
		metrics.Properties["cli_version"] = ctx.App.Version
		metrics.Properties["os"] = runtime.GOOS
		metrics.Properties["arch"] = runtime.GOARCH
		metrics.Properties["project_uuid"] = common.GetProjectUUID()

		// Add command flags as properties
		flags := collectFlagValues(ctx)
		for k, v := range flags {
			// TODO: (brandon c) verify this is adequate
			metrics.Properties[k] = fmt.Sprintf("%v", v)
		}

		// Add command invocation metric
		metrics.AddMetric(FormatMetricName(command, "Count"), 1)

		// If this is the create command with --no-telemetry, switch to NoopClient after tracking "invoked"
		if command == "create" && ctx.Bool("no-telemetry") {
			log.Printf("DEBUG: Detected --no-telemetry flag in create command, switching to NoopClient after invoked event")
			ctx.Context = telemetry.WithContext(ctx.Context, telemetry.NewNoopClient())
		}

		return nil
	}
}

// WithTelemetryPostProcessor creates a post-processor that emits metrics
func WithTelemetryPostProcessor() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Get metrics context
		metrics, err := telemetry.MetricsFromContext(ctx.Context)
		if err != nil {
			log.Printf("Unable to get metrics from context: %v", err)
			return nil
		}
		command := ctx.Command.Name

		// Add command result as a metric
		result := "Success"
		if ctx.Err() != nil {
			result = "Failure"
			metrics.Properties[result] = ctx.Err().Error()
		}
		metrics.AddMetric(FormatMetricName(command, result), 1)

		// Add duration metric
		duration := time.Since(metrics.StartTime).Milliseconds()
		metrics.AddMetric(FormatMetricName(command, "DurationMilliseconds"), float64(duration))

		// Emit all collected metrics
		client, ok := telemetry.ClientFromContext(ctx.Context)
		if !ok {
			return nil
		}

		// For each metric, combine context properties with metric dimensions
		for _, metric := range metrics.Metrics {
			// Create properties map starting with context properties
			props := make(map[string]interface{})
			for k, v := range metrics.Properties {
				props[k] = v
			}

			// Add metric value
			props["metric_value"] = metric.Value

			// Add metric-specific dimensions
			for k, v := range metric.Dimensions {
				props[k] = v
			}

			// Track the metric event
			_ = client.AddMetric(ctx.Context, metric)
		}

		return nil
	}
}

// WithEnvLoader creates a pre-processor that loads environment variables
func WithEnvLoader() cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Skip loading .env for the create command
		if command != "create" {
			if err := loadEnvFile(); err != nil {
				return err
			}
		}

		return nil
	}
}

// loadEnvFile loads environment variables from .env file if it exists
// Silently succeeds if no .env file is found
func loadEnvFile() error {
	// Check if .env file exists in current directory
	if _, err := os.Stat(EnvFile); os.IsNotExist(err) {
		return nil // .env doesn't exist, just return without error
	}

	// Load .env file
	return godotenv.Load(EnvFile)
}
