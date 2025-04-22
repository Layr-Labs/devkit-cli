package template

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var configPath = filepath.Join("config", "templates.yml")

type Config struct {
	Architectures map[string]Architecture `yaml:"architectures"`
}

type Architecture struct {
	Languages map[string]Language `yaml:"languages"`
}

type Language struct {
	Template string `yaml:"template"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func GetTemplateURL(config *Config, arch, lang string) (string, error) {
	archConfig, exists := config.Architectures[arch]
	if !exists {
		return "", nil
	}

	langConfig, exists := archConfig.Languages[lang]
	if !exists {
		return "", nil
	}

	if langConfig.Template == "" {
		return "", nil
	}

	return langConfig.Template, nil
}
