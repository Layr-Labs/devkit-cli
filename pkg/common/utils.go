package common

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

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
		log = logger.NewLogger(verbose)
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		log = logger.NewZapLogger(verbose)
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

// ParseETHAmount parses ETH amount strings like "5ETH", "10.5ETH", "1000000000000000000" (wei)
// Returns the amount in wei as *big.Int
func ParseETHAmount(amountStr string) (*big.Int, error) {
	if amountStr == "" {
		return nil, fmt.Errorf("amount string is empty")
	}

	// Remove any whitespace
	amountStr = strings.TrimSpace(amountStr)

	// Check if it ends with "ETH"
	if strings.HasSuffix(strings.ToUpper(amountStr), "ETH") {
		// Remove the "ETH" suffix (case insensitive)
		ethIndex := strings.LastIndex(strings.ToUpper(amountStr), "ETH")
		numericPart := strings.TrimSpace(amountStr[:ethIndex])

		// Parse the numeric part as float64 to handle decimals like "1.5ETH"
		ethAmount, err := strconv.ParseFloat(numericPart, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ETH amount '%s': %w", numericPart, err)
		}

		// Convert ETH to wei (multiply by 10^18)
		// Use big.Float to handle the large numbers properly
		ethBig := big.NewFloat(ethAmount)
		weiPerEth := big.NewFloat(1e18)
		weiBig := new(big.Float).Mul(ethBig, weiPerEth)

		// Convert to big.Int
		weiInt, _ := weiBig.Int(nil)
		return weiInt, nil
	}

	// If no "ETH" suffix, assume it's already in wei
	weiAmount := new(big.Int)
	if _, ok := weiAmount.SetString(amountStr, 10); !ok {
		return nil, fmt.Errorf("invalid wei amount '%s'", amountStr)
	}

	return weiAmount, nil
}
