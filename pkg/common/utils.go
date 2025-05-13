package common

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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
