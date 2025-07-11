package template

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Framework map[string]FrameworkSpec `yaml:"framework"`
}

type FrameworkSpec struct {
	Template  string   `yaml:"template"`
	Version   string   `yaml:"version"`
	Languages []string `yaml:"languages"`
}

func LoadConfig() (*Config, error) {
	// pull from embedded string
	data := []byte(config.TemplatesYaml)

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetTemplateURLs returns template URL & version for the requested framework + language.
// Fails fast if the framework does not exist, the template URL is blank, or the
// language is not declared in the framework's Languages slice.
func GetTemplateURLs(config *Config, framework, lang string) (string, string, error) {
	fw, ok := config.Framework[framework]
	if !ok {
		return "", "", fmt.Errorf("unknown framework %q", framework)
	}
	if fw.Template == "" {
		return "", "", fmt.Errorf("template URL missing for framework %q", framework)
	}

	// Language gate – only enforce if Languages slice is populated
	if len(fw.Languages) != 0 {
		for _, l := range fw.Languages {
			if l == lang {
				return fw.Template, fw.Version, nil
			}
		}
		return "", "", fmt.Errorf("language %q not available for framework %q", lang, framework)
	}

	return fw.Template, fw.Version, nil
}
