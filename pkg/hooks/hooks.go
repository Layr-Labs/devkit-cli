package hooks

import (
	"devkit-cli/pkg/common"
	"fmt"
	"os"
	"runtime"
	"time"

	kitcontext "devkit-cli/pkg/context"
	"devkit-cli/pkg/telemetry"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// EnvFile is the name of the environment file
const EnvFile = ".env"

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

func (ac *ActionChain) Wrap(action cli.ActionFunc) cli.ActionFunc {
	for i := len(ac.Processors) - 1; i >= 0; i-- {
		action = ac.Processors[i](action)
	}
	return action
}

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
	// TODO: future-proof for other "create" commands.
	if command == "devkit avs create" && ctx.Bool("disable-telemetry") {
		return telemetry.NewNoopClient()
	}

	if !common.IsTelemetryEnabled() {
		return telemetry.NewNoopClient()
	}

	appEnv, ok := kitcontext.AppEnvironmentFromContext(ctx.Context)
	if !ok {
		return telemetry.NewNoopClient()
	}

	phClient, err := telemetry.NewPostHogClient(appEnv, "DevKit")
	if err != nil {
		return telemetry.NewNoopClient()
	}

	return phClient
}

func WithAppEnvironment(ctx *cli.Context) {
	ctx.Context = kitcontext.WithAppEnvironment(ctx.Context, kitcontext.NewAppEnvironment(
		ctx.App.Version,
		runtime.GOOS,
		runtime.GOARCH,
		common.GetProjectUUID(),
	))
}

func WithMetricEmission(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Run command action
		err := action(ctx)

		client := setupTelemetry(ctx, ctx.Command.HelpName)
		ctx.Context = telemetry.ContextWithClient(ctx.Context, client)
		// emit result metrics
		emitTelemetryMetrics(ctx, err)

		return err
	}
}

func emitTelemetryMetrics(ctx *cli.Context, actionError error) {
	metrics, err := telemetry.MetricsFromContext(ctx.Context)
	if err != nil {
		return
	}
	metrics.Properties["command"] = ctx.Command.HelpName
	result := "Success"
	dimensions := map[string]string{}
	if actionError != nil {
		result = "Failure"
		dimensions["error"] = actionError.Error()
	}
	metrics.AddMetricWithDimensions(result, 1, dimensions)

	duration := time.Since(metrics.StartTime).Milliseconds()
	metrics.AddMetric("DurationMilliseconds", float64(duration))

	client, ok := telemetry.ClientFromContext(ctx.Context)
	if !ok {
		return
	}
	defer client.Close()

	for _, metric := range metrics.Metrics {
		mDimensions := metric.Dimensions
		for k, v := range metrics.Properties {
			mDimensions[k] = v
		}
		_ = client.AddMetric(ctx.Context, metric)
	}
}

func LoadEnvFile(ctx *cli.Context) error {
	command := ctx.Command.Name

	// Skip loading .env for the create command
	if command != "create" {
		if err := loadEnvFile(); err != nil {
			return err
		}
	}
	return nil
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

func WithCommandMetricsContext(ctx *cli.Context) error {
	metrics := telemetry.NewMetricsContext()
	ctx.Context = telemetry.WithMetricsContext(ctx.Context, metrics)

	if appEnv, ok := kitcontext.AppEnvironmentFromContext(ctx.Context); ok {
		metrics.Properties["cli_version"] = appEnv.CLIVersion
		metrics.Properties["os"] = appEnv.OS
		metrics.Properties["arch"] = appEnv.Arch
		metrics.Properties["project_uuid"] = appEnv.ProjectUUID
	}

	for k, v := range collectFlagValues(ctx) {
		metrics.Properties[k] = fmt.Sprintf("%v", v)
	}

	metrics.AddMetric("Count", 1)
	return nil
}
