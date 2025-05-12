package common

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const DefaultBaseConfigPath = "config/config.yaml"

type ConfigBlock struct {
	Project ProjectConfig `yaml:"project"`
}

type ProjectConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Context string `yaml:"context"`
}

type ChainContextConfig struct {
	Name    string `yaml:"name"`
	ChainID int    `yaml:"chain_id"`
	RPCURL  string `yaml:"rpc_url"`
	// Fork      *ForkConfig      `yaml:"fork,omitempty"`
	// Operators []OperatorSpec   `yaml:"operators"`
	// AVS       AVSConfig        `yaml:"avs"`
}

type BaseConfig struct {
	Config  ConfigBlock                   `yaml:"config"`
	Context map[string]ChainContextConfig `yaml:"contexts"`
}

func LoadBaseConfig(contextName string) (*BaseConfig, error) {
	// Load base config
	data, err := os.ReadFile(DefaultBaseConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}
	var cfg BaseConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load requested context file
	contextFile := filepath.Join("config", "contexts", contextName+".yaml")
	ctxData, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context %q file: %w", contextName, err)
	}

	// We expect the context file to have a top-level `context:` block
	var wrapper struct {
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
