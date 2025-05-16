package template

import (
	"devkit-cli/config"

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
	Template string `yaml:"template"`
	Commit   string `yaml:"commit"`
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
// Returns main template URL, contracts template URL (may be empty), and error
func GetTemplateURLs(config *Config, arch, lang string) (string, string, string, string) {
	archConfig, exists := config.Architectures[arch]
	if !exists {
		return "", "", "", ""
	}

	// Get main template URL
	langConfig, exists := archConfig.Languages[lang]
	if !exists {
		return "", "", "", ""
	}

	mainURL := langConfig.Template
	if mainURL == "" {
		return "", "", "", ""
	}
	mainCommit := langConfig.Commit

	// Get contracts template URL (default to solidity, no error if missing)
	contractsURL := ""
	contractsCommit := ""
	if archConfig.Contracts != nil {
		if contractsLang, exists := archConfig.Contracts.Languages["solidity"]; exists {
			contractsURL = contractsLang.Template
			contractsCommit = contractsLang.Commit
		}
	}

	return mainURL, mainCommit, contractsURL, contractsCommit
}
