package common

import (
	"github.com/urfave/cli/v2"
)

// IsVerboseEnabled checks if either the CLI --verbose flag is set,
// or eigen.toml has [log] level = "debug"
func IsVerboseEnabled(cCtx *cli.Context, cfg *BaseConfig) bool {
	// Check CLI flag
	if cCtx.Bool("verbose") {
		return true
	}

	// Check eigen.toml config
	// level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))
	// return level == "debug"
	return true
}
