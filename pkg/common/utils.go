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

// GetVerboseLogger creates a verbose logger with progress tracker
func GetVerboseLogger() (iface.Logger, iface.ProgressTracker) {
	return GetLogger(true)
}

// Get logger for the env we're in
func GetLogger(verbose bool) (iface.Logger, iface.ProgressTracker) {

	log := logger.NewZapLogger(verbose)
	var tracker iface.ProgressTracker

	if progress.IsTTY() {
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		tracker = progress.NewLogProgressTracker(10, log)
	}

	return log, tracker
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
