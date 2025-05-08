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
	Contracts *ContractConfig     `yaml:"contracts,omitempty"`
}

type ContractConfig struct {
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

// GetTemplateURLs retrieves both main and contracts template URLs for the given architecture
// Returns main template URL, contracts template URL (may be empty), and error
func GetTemplateURLs(config *Config, arch, lang string) (string, string, error) {
	archConfig, exists := config.Architectures[arch]
	if !exists {
		return "", "", nil
	}

	// Get main template URL
	langConfig, exists := archConfig.Languages[lang]
	if !exists {
		return "", "", nil
	}

	mainURL := langConfig.Template
	if mainURL == "" {
		return "", "", nil
	}

	// Get contracts template URL (default to solidity, no error if missing)
	contractsURL := ""
	if archConfig.Contracts != nil {
		if contractsLang, exists := archConfig.Contracts.Languages["solidity"]; exists {
			contractsURL = contractsLang.Template
		}
	}

	return mainURL, contractsURL, nil
}
