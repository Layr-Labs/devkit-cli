package common

import (
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

const DefaultConfigWithContextConfigPath = "config"

type ConfigBlock struct {
	Project ProjectConfig `json:"project" yaml:"project"`
}

type ProjectConfig struct {
	Name    string `json:"name" yaml:"name"`
	Version string `json:"version" yaml:"version"`
	Context string `json:"context" yaml:"context"`
}

type ForkConfig struct {
	Block int    `json:"block" yaml:"block"`
	Url   string `json:"url" yaml:"url"`
}

type OperatorSpec struct {
	ECDSAKey            string `json:"ecdsa_key" yaml:"ecdsa_key"`
	BlsKeystorePath     string `json:"bls_keystore_path" yaml:"bls_keystore_path"`
	BlsKeystorePassword string `json:"bls_keystore_password" yaml:"bls_keystore_password"`
	Stake               string `json:"stake" yaml:"stake"`
}

type ChainContextConfig struct {
	Name                  string         `json:"name" yaml:"name"`
	Chains                []ChainConfig  `json:"chains" yaml:"chains"`
	DeployerPrivateKey    string         `json:"deployer_private_key" yaml:"deployer_private_key"`
	AppDeployerPrivateKey string         `json:"app_private_key" yaml:"app_private_key"`
	Operators             []OperatorSpec `json:operators" yaml:"operators"`
	Avs                   AvsConfig      `json:"avs" yaml:"avs"`
}

type AvsConfig struct {
	Address          string `json:"address" yaml:"address"`
	MetadataUri      string `json:"metadata_url" yaml:"metadata_url"`
	RegistrarAddress string `json:"registrar_address" yaml:"registrar_address"`
}

type ChainConfig struct {
	Name    string      `json:"name" yaml:"name"`
	ChainID int         `json:"chain_id" yaml:"chain_id"`
	RPCURL  string      `json:"rpc_url" yaml:"rpc_url"`
	Fork    *ForkConfig `json:"fork" yaml:"fork"`
}

type ConfigWithContextConfig struct {
	Config  ConfigBlock                   `json:"config" yaml:"config"`
	Context map[string]ChainContextConfig `json:"context" yaml:"context"`
}

func LoadConfigWithContextConfig(contextName string) (*ConfigWithContextConfig, error) {
	// Load base config
	configPath := filepath.Join(DefaultConfigWithContextConfigPath, BaseConfig)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}

	var cfg ConfigWithContextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load requested context file
	contextFile := filepath.Join(DefaultConfigWithContextConfigPath, "contexts", contextName+".yaml")
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

func LoadConfigWithContextConfigWithoutContext() (*ConfigWithContextConfig, error) {
	configPath := filepath.Join(DefaultConfigWithContextConfigPath, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}
	var cfg ConfigWithContextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}
	return &cfg, nil
}
