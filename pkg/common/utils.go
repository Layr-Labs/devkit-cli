package common

import (
	"context"
	"os"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"

	"github.com/urfave/cli/v2"
)

// loggerContextKey is used to store the logger in the context
type loggerContextKey struct{}

// progressTrackerContextKey is used to store the progress tracker in the context
type progressTrackerContextKey struct{}

// IsVerboseEnabled checks if either the CLI --verbose flag is set,
// or config.yaml has [log] level = "debug"
func IsVerboseEnabled(cCtx *cli.Context, cfg *ConfigWithContextConfig) bool {
	// Check CLI flag
	if cCtx.Bool("verbose") {
		return true
	}

	// Check config.yaml config
	// level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))  // TODO(nova): Get log level debug from config.yaml also . For now only using the cli flag
	// return level == "debug"
	return true
}

// GetLoggerFromCLIContext creates a logger based on the CLI context
// It checks the verbose flag and returns the appropriate logger
func GetLoggerFromCLIContext(cCtx *cli.Context) (iface.Logger, iface.ProgressTracker) {
	verbose := cCtx.Bool("verbose")
	return GetLogger(verbose)
}

// Get logger for the env we're in
func GetLogger(verbose bool) (iface.Logger, iface.ProgressTracker) {

	var log iface.Logger
	var tracker iface.ProgressTracker

	if progress.IsTTY() {
		baseLogger := logger.NewLogger(verbose)
		// Wrap with ColoredLogger for TTY environments to enable color-coded logging
		log = logger.NewColoredLogger(baseLogger)
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		baseLogger := logger.NewZapLogger(verbose)
		// Wrap with ColoredLogger for non-TTY environments too (colors will be stripped by most log processors)
		log = logger.NewColoredLogger(baseLogger)
		tracker = progress.NewLogProgressTracker(10, log)
	}

	return log, tracker
}

// isCI checks if the code is running in a CI environment like GitHub Actions.
func isCI() bool {
	return os.Getenv("CI") == "true"
}

// WithLogger stores the logger in the context
func WithLogger(ctx context.Context, logger iface.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// WithProgressTracker stores the progress tracker in the context
func WithProgressTracker(ctx context.Context, tracker iface.ProgressTracker) context.Context {
	return context.WithValue(ctx, progressTrackerContextKey{}, tracker)
}

// LoggerFromContext retrieves the logger from the context
// If no logger is found, it returns a non-verbose logger as fallback
func LoggerFromContext(ctx context.Context) iface.Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(iface.Logger); ok {
		return logger
	}
	// Fallback to non-verbose logger if not found in context
	log, _ := GetLogger(false)
	return log
}

// ProgressTrackerFromContext retrieves the progress tracker from the context
// If no tracker is found, it returns a non-verbose tracker as fallback
func ProgressTrackerFromContext(ctx context.Context) iface.ProgressTracker {
	if tracker, ok := ctx.Value(progressTrackerContextKey{}).(iface.ProgressTracker); ok {
		return tracker
	}
	// Fallback to non-verbose tracker if not found in context
	_, tracker := GetLogger(false)
	return tracker
}
