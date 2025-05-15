package common

import (
	"devkit-cli/pkg/common/iface"
	"devkit-cli/pkg/common/logger"
	"devkit-cli/pkg/common/progress"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
)

// IsVerboseEnabled checks if either the CLI --verbose flag is set,
// or eigen.toml has [log] level = "debug"
func IsVerboseEnabled(cCtx *cli.Context, cfg *ConfigWithContextConfig) bool {
	// Check CLI flag
	if cCtx.Bool("verbose") {
		return true
	}

	// Check eigen.toml config
	// level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))  // TODO(nova): Get log level debug from config.yaml also . For now only using the cli flag
	// return level == "debug"
	return true
}

func CopyFileTesting(t *testing.T, src, dst string) {
	srcFile, err := os.Open(src)
	assert.NoError(t, err)
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	assert.NoError(t, err)
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	assert.NoError(t, err)
	assert.NoError(t, dstFile.Sync())
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}

	return dstFile.Sync()
}

// GetChainByName returns the pointer to a ChainConfig with the specified name,
// or nil and false if not found.
func GetChainByName(ctx ChainContextConfig, name string) (*ChainConfig, bool) {
	for i, chain := range ctx.Chains {
		if chain.Name == name {
			return &ctx.Chains[i], true
		}
	}
	return nil, false
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
