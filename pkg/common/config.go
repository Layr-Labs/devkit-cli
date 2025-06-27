package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultConfigWithContextConfigPath = "config"

type ConfigBlock struct {
	Project ProjectConfig `json:"project" yaml:"project"`
}

type ProjectConfig struct {
	Name             string `json:"name" yaml:"name"`
	Version          string `json:"version" yaml:"version"`
	Context          string `json:"context" yaml:"context"`
	ProjectUUID      string `json:"project_uuid,omitempty" yaml:"project_uuid,omitempty"`
	TelemetryEnabled bool   `json:"telemetry_enabled" yaml:"telemetry_enabled"`
	TemplateBaseURL  string `json:"templateBaseUrl,omitempty" yaml:"templateBaseUrl,omitempty"`
	TemplateVersion  string `json:"templateVersion,omitempty" yaml:"templateVersion,omitempty"`
}

type ForkConfig struct {
	Url       string `json:"url" yaml:"url"`
	Block     int    `json:"block" yaml:"block"`
	BlockTime int    `json:"block_time" yaml:"block_time"`
}

type OperatorSpec struct {
	Address             string               `json:"address" yaml:"address"`
	ECDSAKey            string               `json:"ecdsa_key" yaml:"ecdsa_key"`
	BlsKeystorePath     string               `json:"bls_keystore_path" yaml:"bls_keystore_path"`
	BlsKeystorePassword string               `json:"bls_keystore_password" yaml:"bls_keystore_password"`
	Stake               string               `json:"stake,omitempty" yaml:"stake,omitempty"`
	Allocations         []OperatorAllocation `json:"allocations,omitempty" yaml:"allocations,omitempty"`
}

// OperatorAllocation defines strategy allocation for an operator
type OperatorAllocation struct {
	StrategyAddress        string                  `json:"strategy_address" yaml:"strategy_address"`
	Name                   string                  `json:"name" yaml:"name"`
	OperatorSetAllocations []OperatorSetAllocation `json:"operator_set_allocations" yaml:"operator_set_allocations"`
}

// OperatorSetAllocation defines allocation for a specific operator set
type OperatorSetAllocation struct {
	OperatorSet      string `json:"operator_set" yaml:"operator_set"`
	AllocationInWads string `json:"allocation_in_wads" yaml:"allocation_in_wads"`
}

// StakerSpec defines a staker configuration with address, key, and deposits
type StakerSpec struct {
	StakerAddress   string           `json:"address" yaml:"address"`
	StakerECDSAKey  string           `json:"ecdsa_key" yaml:"ecdsa_key"`
	Deposits        []StakerDeposits `json:"deposits" yaml:"deposits"`
	OperatorAddress string           `json:"operator" yaml:"operator"`
}

// StakerDeposits defines a deposit to a strategy
type StakerDeposits struct {
	StrategyAddress string `json:"strategy_address" yaml:"strategy_address"`
	Name            string `json:"name" yaml:"name"`
	DepositAmount   string `json:"deposit_amount" yaml:"deposit_amount"`
}

type AvsConfig struct {
	Address          string `json:"address" yaml:"address"`
	MetadataUri      string `json:"metadata_url" yaml:"metadata_url"`
	AVSPrivateKey    string `json:"avs_private_key" yaml:"avs_private_key"`
	RegistrarAddress string `json:"registrar_address" yaml:"registrar_address"`
}

type EigenLayerConfig struct {
	L1 EigenLayerL1Config `json:"l1" yaml:"l1"`
	L2 EigenLayerL2Config `json:"l2" yaml:"l2"`
}

type EigenLayerL1Config struct {
	AllocationManager    string `json:"allocation_manager" yaml:"allocation_manager"`
	DelegationManager    string `json:"delegation_manager" yaml:"delegation_manager"`
	StrategyManager      string `json:"strategy_manager" yaml:"strategy_manager"`
	BN254TableCalculator string `json:"bn254_table_calculator" yaml:"bn254_table_calculator"`
	CrossChainRegistry   string `json:"cross_chain_registry" yaml:"cross_chain_registry"`
	KeyRegistrar         string `json:"key_registrar" yaml:"key_registrar"`
	ReleaseManager       string `json:"release_manager" yaml:"release_manager"`
}

type EigenLayerL2Config struct {
	BN254CertificateVerifier string `json:"bn254_certificate_verifier" yaml:"bn254_certificate_verifier"`
	OperatorTableUpdater     string `json:"operator_table_updater" yaml:"operator_table_updater"`
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

type Config struct {
	Version string      `json:"version" yaml:"version"`
	Config  ConfigBlock `json:"config" yaml:"config"`
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

type StakeRootEntry struct {
	ChainID   uint64 `yaml:"chain_id" json:"chain_id"`
	StakeRoot string `yaml:"stake_root" json:"stake_root"`
}

type Transporter struct {
	Schedule         string           `json:"schedule" yaml:"schedule"`
	PrivateKey       string           `json:"private_key" yaml:"private_key"`
	BlsPrivateKey    string           `json:"bls_private_key" yaml:"bls_private_key"`
	ActiveStakeRoots []StakeRootEntry `json:"active_stake_roots,omitempty" yaml:"active_stake_roots,omitempty"`
}

// ArtifactConfig defines the structure for release artifacts
type ArtifactConfig struct {
	ArtifactId string `json:"artifactId" yaml:"artifactId"`
	Component  string `json:"component" yaml:"component"`
	Digest     string `json:"digest" yaml:"digest"`
	Registry   string `json:"registry" yaml:"registry"`
	Version    string `json:"version" yaml:"version"`
}

type ChainContextConfig struct {
	Name                  string                 `json:"name" yaml:"name"`
	Chains                map[string]ChainConfig `json:"chains" yaml:"chains"`
	Transporter           Transporter            `json:"transporter" yaml:"transporter"`
	DeployerPrivateKey    string                 `json:"deployer_private_key" yaml:"deployer_private_key"`
	AppDeployerPrivateKey string                 `json:"app_private_key" yaml:"app_private_key"`
	Operators             []OperatorSpec         `json:"operators" yaml:"operators"`
	Avs                   AvsConfig              `json:"avs" yaml:"avs"`
	EigenLayer            *EigenLayerConfig      `json:"eigenlayer" yaml:"eigenlayer"`
	DeployedContracts     []DeployedContract     `json:"deployed_contracts,omitempty" yaml:"deployed_contracts,omitempty"`
	OperatorSets          []OperatorSet          `json:"operator_sets" yaml:"operator_sets"`
	OperatorRegistrations []OperatorRegistration `json:"operator_registrations" yaml:"operator_registrations"`
	Stakers               []StakerSpec           `json:"stakers" yaml:"stakers"`
	Artifact              *ArtifactConfig        `json:"artifact" yaml:"artifact"`
}

func LoadBaseConfig() (map[string]interface{}, error) {
	path := filepath.Join(DefaultConfigWithContextConfigPath, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse base config: %w", err)
	}
	return cfg, nil
}

func LoadContextConfig(ctxName string) (map[string]interface{}, error) {
	// Default to devnet
	if ctxName == "" {
		ctxName = "devnet"
	}
	path := filepath.Join(DefaultConfigWithContextConfigPath, "contexts", ctxName+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read context %q: %w", ctxName, err)
	}
	var ctx map[string]interface{}
	if err := yaml.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("parse context %q: %w", ctxName, err)
	}
	return ctx, nil
}

func LoadBaseConfigYaml() (*Config, error) {
	path := filepath.Join(DefaultConfigWithContextConfigPath, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg *Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

func LoadConfigWithContextConfig(ctxName string) (*ConfigWithContextConfig, error) {
	// Default to devnet
	if ctxName == "" {
		ctxName = "devnet"
	}

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
	contextFile := filepath.Join(DefaultConfigWithContextConfigPath, "contexts", ctxName+".yaml")
	ctxData, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context %q file: %w", ctxName, err)
	}

	var wrapper struct {
		Version string             `yaml:"version"`
		Context ChainContextConfig `yaml:"context"`
	}

	if err := yaml.Unmarshal(ctxData, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse context file %q: %w", contextFile, err)
	}

	cfg.Context = map[string]ChainContextConfig{
		ctxName: wrapper.Context,
	}

	return &cfg, nil
}

func LoadContext(context string) (string, *yaml.Node, *yaml.Node, error) {
	// Set path for context yaml
	contextDir := filepath.Join("config", "contexts")
	yamlPath := path.Join(contextDir, fmt.Sprintf("%s.%s", context, "yaml"))

	// Load YAML as *yaml.Node
	rootNode, err := LoadYAML(yamlPath)
	if err != nil {
		return yamlPath, nil, nil, err
	}

	// YAML is parsed into a DocumentNode:
	//   - rootNode.Content[0] is the top-level MappingNode
	//   - It contains the 'context' mapping we're interested in
	if len(rootNode.Content) == 0 {
		return yamlPath, rootNode, nil, fmt.Errorf("empty YAML root node")
	}

	// Navigate context to arrive at context.transporter.active_stake_roots
	contextNode := GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return yamlPath, rootNode, nil, fmt.Errorf("missing 'context' key in ./config/contexts/%s.yaml", context)
	}

	return yamlPath, rootNode, contextNode, nil
}

func LoadRawContext(context string) ([]byte, error) {
	_, _, contextNode, err := LoadContext(context)
	if err != nil {
		return nil, err
	}

	var ctxMap map[string]interface{}
	if err := contextNode.Decode(&ctxMap); err != nil {
		return nil, fmt.Errorf("decode context node: %w", err)
	}

	contextBytes, err := json.Marshal(map[string]interface{}{"context": ctxMap})
	if err != nil {
		return nil, fmt.Errorf("marshal context: %w", err)
	}

	return contextBytes, nil
}

func RequireNonZero(s interface{}) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return fmt.Errorf("must be non-nil")
		}
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		// skip private or omitempty-tagged fields
		if f.PkgPath != "" || strings.Contains(f.Tag.Get("yaml"), "omitempty") {
			continue
		}
		fv := v.Field(i)
		if reflect.DeepEqual(fv.Interface(), reflect.Zero(f.Type).Interface()) {
			return fmt.Errorf("missing required field: %s", f.Name)
		}
		// if nested struct, recurse
		if fv.Kind() == reflect.Struct || (fv.Kind() == reflect.Ptr && fv.Elem().Kind() == reflect.Struct) {
			if err := RequireNonZero(fv.Interface()); err != nil {
				return fmt.Errorf("%s.%w", f.Name, err)
			}
		}
	}
	return nil
}
