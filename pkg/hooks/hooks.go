package hooks

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/logger"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	devcontext "devkit-cli/pkg/context"
	"devkit-cli/pkg/telemetry"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// EnvFile is the name of the environment file
const EnvFile = ".env"

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
	if command == "create" && ctx.Bool("disable-telemetry") {
		return telemetry.NewNoopClient()
	}

	appEnv, ok := devcontext.AppEnvironmentFromContext(ctx.Context)
	if !ok {
		return telemetry.NewNoopClient()
	}

	logger.NewLogger().Info("Creating posthog client.")
	phClient, err := telemetry.NewPostHogClient(appEnv)
	if err != nil {
		return telemetry.NewNoopClient()
	}

	return phClient
}

func FormatMetricName(command, action string) string {
	if strings.Contains(action, ".") {
		return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
	}
	return fmt.Sprintf("cli.%s%s.%s", CommandPrefix, command, action)
}

// CommandMiddleware wraps a command with pre-processing and post-processing steps
type CommandMiddleware struct {
	PreProcessors  []func(action cli.ActionFunc) cli.ActionFunc
	PostProcessors []func(action cli.ActionFunc) cli.ActionFunc
}

type ActionChain struct {
	Processors []func(action cli.ActionFunc) cli.ActionFunc
}

// NewActionChain creates a new action chain
func NewActionChain() *ActionChain {
	return &ActionChain{
		Processors: make([]func(action cli.ActionFunc) cli.ActionFunc, 0),
	}
}

// Use appends a new processor to the chain
func (ac *ActionChain) Use(processor func(action cli.ActionFunc) cli.ActionFunc) {
	ac.Processors = append(ac.Processors, processor)
}

// Wrap applies all processors in the correct order
func (ac *ActionChain) Wrap(action cli.ActionFunc) cli.ActionFunc {
	for i := len(ac.Processors) - 1; i >= 0; i-- {
		action = ac.Processors[i](action)
	}
	return action
}

// ApplyMiddleware applies a list of middleware functions to commands
func ApplyMiddleware(commands []*cli.Command, chain *ActionChain) {
	for _, cmd := range commands {
		if cmd.Action != nil {
			cmd.Action = chain.Wrap(cmd.Action)
		}
		if len(cmd.Subcommands) > 0 {
			ApplyMiddleware(cmd.Subcommands, chain)
		}
	}
}

func WithTelemetry(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		// Pre-processing to set up telemetry context and metrics
		setupTelemetryContext(ctx, command)

		// Run requested cli action
		err := action(ctx)

		// Post-processing to emit result metrics
		emitTelemetryMetrics(ctx, command, err)

		return err
	}
}

func setupTelemetryContext(ctx *cli.Context, command string) {
	client := setupTelemetry(ctx, command)
	ctx.Context = telemetry.WithContext(ctx.Context, client)

	metrics := telemetry.NewMetricsContext(ctx.App.Name, command)
	ctx.Context = telemetry.WithMetricsContext(ctx.Context, metrics)

	if appEnv, ok := devcontext.AppEnvironmentFromContext(ctx.Context); ok {
		metrics.Properties["cli_version"] = appEnv.CLIVersion
		metrics.Properties["os"] = appEnv.OS
		metrics.Properties["arch"] = appEnv.Arch
		metrics.Properties["project_uuid"] = appEnv.ProjectUUID
	}

	for k, v := range collectFlagValues(ctx) {
		metrics.Properties[k] = fmt.Sprintf("%v", v)
	}

	metrics.AddMetric(FormatMetricName(command, "Count"), 1)

	// Handle no-telemetry override
	if command == "create" && ctx.Bool("no-telemetry") {
		log.Printf("DEBUG: Detected --no-telemetry flag in create command, switching to NoopClient after invoked event")
		ctx.Context = telemetry.WithContext(ctx.Context, telemetry.NewNoopClient())
	}
}

func emitTelemetryMetrics(ctx *cli.Context, command string, actionError error) {
	metrics, mErr := telemetry.MetricsFromContext(ctx.Context)
	if mErr != nil {
		return
	}

	result := "Success"
	if actionError != nil {
		result = "Failure"
		metrics.Properties["error"] = actionError.Error()
	}

	metrics.AddMetric(FormatMetricName(command, result), 1)
	duration := time.Since(metrics.StartTime).Milliseconds()
	metrics.AddMetric(FormatMetricName(command, "DurationMilliseconds"), float64(duration))

	client, ok := telemetry.ClientFromContext(ctx.Context)
	if !ok {
		return
	}
	defer client.Close()

	for _, metric := range metrics.Metrics {
		props := make(map[string]interface{})
		for k, v := range metrics.Properties {
			props[k] = v
		}
		for k, v := range metric.Dimensions {
			props[k] = v
		}
		props["metric_value"] = metric.Value

		_ = client.AddMetric(ctx.Context, metric)
	}
}

// WithEnvLoader creates a pre-processor that loads environment variables
func WithEnvLoader(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		command := ctx.Command.Name

		ctx.Context = devcontext.WithAppEnvironment(ctx.Context, devcontext.NewAppEnvironment(
			ctx.App.Version,
			runtime.GOOS,
			runtime.GOARCH,
			common.GetProjectUUID(),
		))

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
