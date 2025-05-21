package template

import (
	"github.com/Layr-Labs/devkit-cli/config"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Architectures map[string]Architecture `yaml:"architectures"`
}

type Architecture struct {
	Languages map[string]Language `yaml:"languages"`
	Contracts *ContractConfig     `yaml:"contracts,omitempty"`
}

type ContractConfig struct {
	Languages map[string]Language `yaml:"languages"`
}

type Language struct {
	BaseUrl string `yaml:"baseUrl"`
	Version string `yaml:"version"`
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

// GetTemplateURLs retrieves both main and contracts template URLs for the given architecture
// Returns main base URL, main version, contracts base URL, contracts version, and error
func GetTemplateURLs(config *Config, arch, lang string) (string, string, error) {
	archConfig, exists := config.Architectures[arch]
	if !exists {
		return "", "", nil
	}

	// Get main template URL and version
	langConfig, exists := archConfig.Languages[lang]
	if !exists {
		return "", "", nil
	}

	mainBaseURL := langConfig.BaseUrl
	mainVersion := langConfig.Version
	if mainBaseURL == "" {
		return "", "", nil
	}

	return mainBaseURL, mainVersion, nil
}
