package common

import (
	"devkit-cli/pkg/common/iface"
	"devkit-cli/pkg/common/logger"
	"devkit-cli/pkg/common/progress"
	"fmt"
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

func CleanYAML(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for k, v := range x {
			m[fmt.Sprint(k)] = CleanYAML(v)
		}
		return m
	case []interface{}:
		for i, v := range x {
			x[i] = CleanYAML(v)
		}
	}
	return v
}

func DeepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		if dv, ok := dst[k]; ok {
			dMap, ok1 := dv.(map[string]interface{})
			sMap, ok2 := v.(map[string]interface{})
			if ok1 && ok2 {
				dst[k] = DeepMerge(dMap, sMap)
				continue
			}
		}
		dst[k] = v
	}
	return dst
}
