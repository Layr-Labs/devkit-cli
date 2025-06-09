package hooks

import (
	"fmt"
	"os"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/telemetry"

	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

// EnvFile is the name of the environment file
const EnvFile = ".env"
const namespace = "DevKit"

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

func setupTelemetry(ctx *cli.Context) telemetry.Client {
	logger := common.LoggerFromContext(ctx.Context)

	// Get effective telemetry preference (project takes precedence over global)
	telemetryEnabled, err := common.GetEffectiveTelemetryPreference()
	if err != nil {
		logger.Debug("Failed to get telemetry preference: %v", err)
		return telemetry.NewNoopClient()
	}

	// If telemetry is disabled, return noop client
	if !telemetryEnabled {
		return telemetry.NewNoopClient()
	}

	appEnv, ok := common.AppEnvironmentFromContext(ctx.Context)
	if !ok {
		return telemetry.NewNoopClient()
	}

	phClient, err := telemetry.NewPostHogClient(appEnv, namespace)
	if err != nil {
		return telemetry.NewNoopClient()
	}

	return phClient
}

// WithFirstRunTelemetryPrompt handles first-run telemetry setup
func WithFirstRunTelemetryPrompt(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Check if this is the first run
	isFirstRun, err := common.IsFirstRun()
	if err != nil {
		logger.Debug("Failed to check first run status: %v", err)
		return nil // Don't fail the command, just skip the prompt
	}

	if !isFirstRun {
		return nil // Not first run, continue normally
	}

	// Check for global flags that control telemetry prompt behavior
	opts := common.TelemetryPromptOptions{
		EnableTelemetry:  cCtx.Bool("enable-telemetry"),
		DisableTelemetry: cCtx.Bool("disable-telemetry"),
		SkipPromptInCI:   true, // Always skip in CI environments
	}

	// Show telemetry prompt with options and get user choice
	choice, err := common.TelemetryPromptWithOptions(logger, opts)
	if err != nil {
		logger.Debug("Failed to show telemetry prompt: %v", err)
		// If prompt fails, mark first run complete but don't set telemetry preference
		err = common.MarkFirstRunComplete()
		if err != nil {
			logger.Debug("Failed to mark first run complete: %v", err)
		}
		return nil
	}

	// Save the user's choice globally
	if err := common.SetGlobalTelemetryPreference(choice); err != nil {
		logger.Debug("Failed to save telemetry preference: %v", err)
		// Still mark first run complete even if save fails
		err = common.MarkFirstRunComplete()
		if err != nil {
			logger.Debug("Failed to mark first run complete: %v", err)
		}
		return nil
	}

	// First run handling complete
	logger.Debug("First run telemetry setup completed")
	return nil
}

func WithMetricEmission(action cli.ActionFunc) cli.ActionFunc {
	return func(ctx *cli.Context) error {
		// Run command action
		err := action(ctx)

		client := setupTelemetry(ctx)
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

	l := logger.NewZapLogger(false)
	for _, metric := range metrics.Metrics {
		mDimensions := metric.Dimensions
		for k, v := range metrics.Properties {
			mDimensions[k] = v
		}
		err = client.AddMetric(ctx.Context, metric)
		if err != nil {
			l.Error("failed to add metric", "error", err.Error())
		}
	}
}

func LoadEnvFile(ctx *cli.Context) error {
	// Skip loading .env for the create command
	if ctx.Command.Name != "create" {
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

	if appEnv, ok := common.AppEnvironmentFromContext(ctx.Context); ok {
		metrics.Properties["cli_version"] = appEnv.CLIVersion
		metrics.Properties["os"] = appEnv.OS
		metrics.Properties["arch"] = appEnv.Arch
		metrics.Properties["project_uuid"] = appEnv.ProjectUUID
		metrics.Properties["user_uuid"] = appEnv.UserUUID
	}

	for k, v := range collectFlagValues(ctx) {
		metrics.Properties[k] = fmt.Sprintf("%v", v)
	}

	metrics.AddMetric("Count", 1)
	return nil
}
