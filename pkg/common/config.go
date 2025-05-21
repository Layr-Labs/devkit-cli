package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigWithContextConfigPath = "config"

type ConfigBlock struct {
	Project ProjectConfig `json:"project" yaml:"project"`
}

type ProjectConfig struct {
	Name            string `json:"name" yaml:"name"`
	Version         string `json:"version" yaml:"version"`
	Context         string `json:"context" yaml:"context"`
	TemplateBaseURL string `json:"templateBaseUrl,omitempty" yaml:"templateBaseUrl,omitempty"`
	TemplateVersion string `json:"templateVersion,omitempty" yaml:"templateVersion,omitempty"`
}

type ForkConfig struct {
	Url       string `json:"url" yaml:"url"`
	Block     int    `json:"block" yaml:"block"`
	BlockTime int    `json:"block_time" yaml:"block_time"`
}

type OperatorSpec struct {
	Address             string `json:"address" yaml:"address"`
	ECDSAKey            string `json:"ecdsa_key" yaml:"ecdsa_key"`
	BlsKeystorePath     string `json:"bls_keystore_path" yaml:"bls_keystore_path"`
	BlsKeystorePassword string `json:"bls_keystore_password" yaml:"bls_keystore_password"`
	Stake               string `json:"stake" yaml:"stake"`
}

type AvsConfig struct {
	Address          string `json:"address" yaml:"address"`
	MetadataUri      string `json:"metadata_url" yaml:"metadata_url"`
	AVSPrivateKey    string `json:"avs_private_key" yaml:"avs_private_key"`
	RegistrarAddress string `json:"registrar_address" yaml:"registrar_address"`
}

type EigenLayerConfig struct {
	AllocationManager string `json:"allocation_manager" yaml:"allocation_manager"`
	DelegationManager string `json:"delegation_manager" yaml:"delegation_manager"`
}

type ChainConfig struct {
	ChainID int         `json:"chain_id" yaml:"chain_id"`
	RPCURL  string      `json:"rpc_url" yaml:"rpc_url"`
	Fork    *ForkConfig `json:"fork" yaml:"fork"`
}

type DeployedContract struct {
	Name    string `json:"name" yaml:"name"`
	Address string `json:"address" yaml:"address"`
	Abi     string `json:"abi" yaml:"abi"`
}

type ConfigWithContextConfig struct {
	Config  ConfigBlock                   `json:"config" yaml:"config"`
	Context map[string]ChainContextConfig `json:"context" yaml:"context"`
}

type ContextConfig struct {
	Version string             `json:"version" yaml:"version"`
	Context ChainContextConfig `json:"context" yaml:"context"`
}

type OperatorSet struct {
	OperatorSetID uint64     `json:"operator_set_id" yaml:"operator_set_id"`
	Strategies    []Strategy `json:"strategies" yaml:"strategies"`
}

type Strategy struct {
	StrategyAddress string `json:"strategy" yaml:"strategy"`
}

type OperatorRegistration struct {
	Address       string `json:"address" yaml:"address"`
	OperatorSetID uint64 `json:"operator_set_id" yaml:"operator_set_id"`
	Payload       string `json:"payload" yaml:"payload"`
}

type ChainContextConfig struct {
	Name                  string                 `json:"name" yaml:"name"`
	Chains                map[string]ChainConfig `json:"chains" yaml:"chains"`
	DeployerPrivateKey    string                 `json:"deployer_private_key" yaml:"deployer_private_key"`
	AppDeployerPrivateKey string                 `json:"app_private_key" yaml:"app_private_key"`
	Operators             []OperatorSpec         `json:"operators" yaml:"operators"`
	Avs                   AvsConfig              `json:"avs" yaml:"avs"`
	EigenLayer            *EigenLayerConfig      `json:"eigenlayer" yaml:"eigenlayer"`
	DeployedContracts     []DeployedContract     `json:"deployed_contracts,omitempty" yaml:"deployed_contracts,omitempty"`
	OperatorSets          []OperatorSet          `json:"operator_sets" yaml:"operator_sets"`
	OperatorRegistrations []OperatorRegistration `json:"operator_registrations" yaml:"operator_registrations"`
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

func LoadContext(yamlPath string) ([]byte, error) {
	rootNode, err := LoadYAML(yamlPath)
	if err != nil {
		return nil, err
	}
	if len(rootNode.Content) == 0 {
		return nil, fmt.Errorf("empty YAML root node")
	}

	contextNode := GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return nil, fmt.Errorf("missing 'context' key in %s", yamlPath)
	}

	var ctxMap map[string]interface{}
	if err := contextNode.Decode(&ctxMap); err != nil {
		return nil, fmt.Errorf("decode context node: %w", err)
	}

	context, err := json.Marshal(map[string]interface{}{"context": ctxMap})
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	return context, nil
}
