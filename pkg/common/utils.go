package common

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
	"os"

	"github.com/urfave/cli/v2"
)

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

// Get logger for the env we're in
func GetLogger() (iface.Logger, iface.ProgressTracker) {
	var log iface.Logger
	var tracker iface.ProgressTracker
	if progress.IsTTY() {
		log = logger.NewLogger()
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		log = logger.NewZapLogger()
		tracker = progress.NewLogProgressTracker(10, log)
	}

	return log, tracker
}

// isCI checks if the code is running in a CI environment like GitHub Actions.
func isCI() bool {
	return os.Getenv("CI") == "true"
}
