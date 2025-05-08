package hooks

import (
	"context"
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

// CommandMetrics holds timing and metadata for command execution
type CommandMetrics struct {
	StartTime time.Time
	Command   string
	Flags     map[string]interface{}
}

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

func MetricsFromContext(ctx context.Context) (CommandMetrics, bool) {
	metrics, ok := ctx.Value(contextKey{}).(CommandMetrics)
	if metrics.Command == "" {
		return CommandMetrics{}, false
	}
	return metrics, ok
}

func WithCommandMetrics(ctx context.Context, metrics CommandMetrics) context.Context {
	return context.WithValue(ctx, contextKey{}, metrics)
}

func FormatEventName(command, action string) string {
	if strings.Contains(action, ".") {
		return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
	}
	return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
}

func Track(ctx context.Context, name string, props map[string]interface{}) error {
	client, ok := telemetry.FromContext(ctx)
	if !ok {
		return nil
	}
	return client.Track(ctx, name, props)
}

func FormatCustomMetric(ctx context.Context, metricPath string) string {
	metrics, ok := MetricsFromContext(ctx)
	if !ok {
		return fmt.Sprintf("cli.%sunknown.%s", CommandPrefix, metricPath)
	}
	return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, metrics.Command, metricPath)
}

func trackCommandResult(ctx *cli.Context, result string, err error) {
	metrics, ok := MetricsFromContext(ctx.Context)
	if !ok {
		return
	}

	command := metrics.Command

	// Copy the flags map to avoid modifying the original
	props := make(map[string]interface{}, len(metrics.Flags))
	for k, v := range metrics.Flags {
		props[k] = v
	}
	duration := time.Since(metrics.StartTime)
	props["duration_ms"] = duration.Milliseconds()

	// Add error message if present
	if err != nil {
		props["error"] = err.Error()
	}

	// Track the event
	_ = Track(ctx.Context, FormatEventName(command, result), props)
}

func WithTelemetry(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Get telemetry client
		client := setupTelemetry(ctx, command)
		ctx.Context = telemetry.WithContext(ctx.Context, client)

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

		// If this is the create command with --no-telemetry, switch to NoopClient after tracking "invoked"
		if command == "create" && ctx.Bool("no-telemetry") {
			log.Printf("DEBUG: Detected --no-telemetry flag in create command, switching to NoopClient after invoked event")
			ctx.Context = telemetry.WithContext(ctx.Context, telemetry.NewNoopClient())
		}

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

// ApplyMiddleware applies a list of middleware functions to commands
func ApplyMiddleware(commands []*cli.Command, middlewares ...func(cli.ActionFunc) cli.ActionFunc) {
	for _, cmd := range commands {
		// Apply middleware to this command's action if it exists
		if cmd.Action != nil {
			// Store original action
			originalAction := cmd.Action

			// Apply all middlewares in order
			wrappedAction := originalAction
			for _, middleware := range middlewares {
				wrappedAction = middleware(wrappedAction)
			}

			// Set the final wrapped action
			cmd.Action = wrappedAction
		}

		// Recursively apply to subcommands
		if len(cmd.Subcommands) > 0 {
			ApplyMiddleware(cmd.Subcommands, middlewares...)
		}
	}
}

// ApplyTelemetryToCommands applies the telemetry middleware to all commands
func ApplyTelemetryToCommands(commands []*cli.Command) {
	ApplyMiddleware(commands, WithTelemetry)
}

// ApplyEnvLoaderToCommands applies the env loader middleware to all commands
func ApplyEnvLoaderToCommands(commands []*cli.Command) {
	ApplyMiddleware(commands, WithEnvLoader)
}

// WithEnvLoader wraps a command action to load .env file before execution
// except for the create command
func WithEnvLoader(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Skip loading .env for the create command
		if command != "create" {
			if err := loadEnvFile(); err != nil {
				return err
			}
		}

		return action(ctx)
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
