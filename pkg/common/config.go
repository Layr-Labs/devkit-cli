package common

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const DefaultBaseConfigPath = "config"

type ConfigBlock struct {
	Project ProjectConfig `yaml:"project"`
}

type ProjectConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Context string `yaml:"context"`
}

type ForkConfig struct {
	Block int    `yaml:"block"`
	Url   string `yaml:"url"`
}

type OperatorSpec struct {
	ECDSAKey string `json:"ecdsa_key"`
}

type ChainContextConfig struct {
	Name      string         `yaml:"name"`
	ChainID   int            `yaml:"chain_id"`
	RPCURL    string         `yaml:"rpc_url"`
	Fork      *ForkConfig    `yaml:"fork"`
	Operators []OperatorSpec `yaml:"operators"`
}

type BaseConfig struct {
	Config  ConfigBlock                   `yaml:"config"`
	Context map[string]ChainContextConfig `yaml:"contexts"`
}

func LoadBaseConfig(contextName string) (*BaseConfig, error) {
	// Load base config
	configPath := filepath.Join(DefaultBaseConfigPath, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}

	var cfg BaseConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load requested context file
	contextFile := filepath.Join(DefaultBaseConfigPath, "contexts", contextName+".yaml")
	ctxData, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context %q file: %w", contextName, err)
	}

	var wrapper struct {
		Version string             `yaml:"version"`
		Context ChainContextConfig `yaml:"context"`
	}

	if err := yaml.Unmarshal(ctxData, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse context file %q: %w", contextFile, err)
	}

	cfg.Context = map[string]ChainContextConfig{
		contextName: wrapper.Context,
	}

	return &cfg, nil
}

func LoadBaseConfigWithoutContext() (*BaseConfig, error) {
	configPath := filepath.Join(DefaultBaseConfigPath, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}
	var cfg BaseConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}
	return &cfg, nil
}
