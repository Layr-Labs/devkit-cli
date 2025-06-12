package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/Layr-Labs/devkit-cli/internal/version"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
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

// VersionCompatibilityError represents a version mismatch error in migration
type VersionCompatibilityError struct {
	ContextVersion  string
	CLIVersion      string
	LatestSupported string
	ContextFile     string
}

func (e *VersionCompatibilityError) Error() string {
	return fmt.Sprintf(`
⚠️  VERSION COMPATIBILITY WARNING ⚠️

Your context file version is newer than what this devkit 
CLI version supports:

  Current Context file:     %s
  Current Context version:  %s  
  Current CLI version:      %s
  Latest supported context version: %s

This can cause context corruption if you proceed. Please update your devkit CLI first:

  # Update devkit CLI to latest version

VERSION=%s
ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
DISTRO=$(uname -s | tr '[:upper:]' '[:lower:]')

mkdir -p $HOME/bin
curl -sL "https://s3.amazonaws.com/eigenlayer-devkit-releases/${VERSION}/devkit-${DISTRO}-${ARCH}-${VERSION}.tar.gz" | tar xv -C "$HOME/bin"
  
  # Or build from source
  git pull origin main && make install

After updating, verify the CLI version supports your context:
  devkit --version

DO NOT edit the context file until you update the CLI version.
`, e.ContextFile, e.ContextVersion, e.CLIVersion, e.LatestSupported, embeddedDevkitReleaseVersion)
}

// parseVersion converts version string like "0.0.5" to comparable integers
func parseVersion(v string) (major, minor, patch int, err error) {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", v)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}

// compareVersions returns true if v1 > v2
func compareVersions(v1, v2 string) (bool, error) {
	major1, minor1, patch1, err := parseVersion(v1)
	if err != nil {
		return false, fmt.Errorf("parse version %s: %w", v1, err)
	}

	major2, minor2, patch2, err := parseVersion(v2)
	if err != nil {
		return false, fmt.Errorf("parse version %s: %w", v2, err)
	}

	if major1 > major2 {
		return true, nil
	}
	if major1 < major2 {
		return false, nil
	}

	if minor1 > minor2 {
		return true, nil
	}
	if minor1 < minor2 {
		return false, nil
	}

	return patch1 > patch2, nil
}

// checkVersionCompatibility validates that the context version is supported by the current CLI
// Logs a warning if there's a version mismatch, but allows execution to continue
func checkVersionCompatibility(contextVersion, contextFile string, logger iface.Logger) {
	if contextVersion == "" {
		// Missing version - could be very old context, warn but allow
		if logger != nil {
			logger.Info("⚠️  Context file %s is missing version field - this may be an old context that needs migration", contextFile)
		}
		return
	}

	// Get the latest version supported by this CLI
	latestSupported := DevkitLatestContextVersion // This should match contexts.LatestVersion , but cannot check due to import cycle error

	// Compare versions
	isNewer, err := compareVersions(contextVersion, latestSupported)
	if err != nil {
		if logger != nil {
			logger.Info("⚠️  Failed to compare versions: %v", err)
		}
		return
	}

	// If context version is newer than what we support, log compatibility warning
	if isNewer {
		compatError := &VersionCompatibilityError{
			ContextVersion:  contextVersion,
			CLIVersion:      version.GetVersion(),
			LatestSupported: latestSupported,
			ContextFile:     contextFile,
		}
		if logger != nil {
			logger.Info("%s", compatError.Error())
		}
	}
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
	return LoadContextConfigWithLogger(ctxName, nil)
}

func LoadContextConfigWithLogger(ctxName string, logger iface.Logger) (map[string]interface{}, error) {
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

	// Check version compatibility
	if version, ok := ctx["version"].(string); ok {
		checkVersionCompatibility(version, path, logger)
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
	return LoadConfigWithContextConfigAndLogger(ctxName, nil)
}

func LoadConfigWithContextConfigAndLogger(ctxName string, logger iface.Logger) (*ConfigWithContextConfig, error) {
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

	// Check version compatibility before proceeding
	checkVersionCompatibility(wrapper.Version, contextFile, logger)

	cfg.Context = map[string]ChainContextConfig{
		ctxName: wrapper.Context,
	}

	return &cfg, nil
}

func LoadRawContext(yamlPath string) ([]byte, error) {
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
