package plugin

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"path/filepath"
	goplugin "plugin"
	"strings"
)

type IPlugin interface {
	GetCommands() []*cli.Command
	Name() string
	Version() string
}

func resolvePluginPath(pluginPath string) (string, error) {
	if pluginPath == "~" || strings.HasPrefix(pluginPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %v", err)
		}
		if pluginPath == "~" {
			pluginPath = homeDir
		} else {

		}
		pluginPath = filepath.Join(homeDir, pluginPath[2:])
	}
	return filepath.Abs(pluginPath)
}

func LoadPlugins(pluginPath string) ([]IPlugin, error) {
	absPluginPath, err := resolvePluginPath(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve symlinks: %v", err)
	}
	fmt.Printf("Resolved path: %s\n", absPluginPath)

	pathStat, err := os.Stat(absPluginPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("warning: plugin path does not exist: %s", absPluginPath)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat plugin path: %v", err)
	}
	if !pathStat.IsDir() {
		return nil, fmt.Errorf("plugin path is not a directory: %s", absPluginPath)
	}

	var plugins []IPlugin
	files, err := filepath.Glob(filepath.Join(absPluginPath, "*.so"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob plugin files: %v", err)
	}

	if len(files) == 0 {
		return nil, nil
	}

	for _, file := range files {
		p, err := goplugin.Open(file)
		if err != nil {
			return nil, fmt.Errorf("failed to open plugin file %s: %v", file, err)
		}

		sym, err := p.Lookup("GetPlugin")
		if err != nil {
			log.Printf("failed to lookup GetPlugin in %s: %v", file, err)
			continue
		}

		getPlugin, ok := sym.(func() IPlugin)
		if !ok {
			log.Printf("Plugin %s has invalid 'GetPlugin' symbol", file)
			continue
		}

		plugin := getPlugin()
		plugins = append(plugins, plugin)
		log.Printf("Loaded plugin: %s, version: %s", plugin.Name(), plugin.Version())
	}
	return plugins, nil
}
