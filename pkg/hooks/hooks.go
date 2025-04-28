package hooks

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"devkit-cli/pkg/common"
	"devkit-cli/pkg/telemetry"

	"github.com/urfave/cli/v2"
)

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

// ApplyTelemetryToCommands applies the WithTelemetry middleware to all commands in a list
func ApplyTelemetryToCommands(commands []*cli.Command) {
	for _, cmd := range commands {
		// Apply to this command's action if it exists
		if cmd.Action != nil {
			// Store original action before replacing it
			originalAction := cmd.Action
			cmd.Action = WithTelemetry(func(ctx *cli.Context) error {
				return originalAction(ctx)
			})
		}

		// Recursively apply to subcommands
		if len(cmd.Subcommands) > 0 {
			ApplyTelemetryToCommands(cmd.Subcommands)
		}
	}
}
