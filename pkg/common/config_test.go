package common_test

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"

	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestLoadConfigWithContextConfig_FromCopiedTempFile(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Copy config/config.yaml to tempDir
	assert.NoError(t, os.WriteFile(tmpYamlPath, []byte(configs.ConfigYamls[configs.LatestVersion]), 0644))

	// Write config/config.yaml
	configYamlPath := filepath.Join(configDir, "config.yaml")
	assert.NoError(t, os.WriteFile(configYamlPath, []byte(config.DefaultConfigYaml), 0644))
	// Write config/contexts/devnet.yaml
	devnetYamlPath := filepath.Join(configDir, "contexts", "devnet.yaml")
	assert.NoError(t, os.WriteFile(devnetYamlPath, []byte(config.ContextYamls["devnet"]), 0644))

	tmpDevnetPath := filepath.Join(tmpContextDir, "devnet.yaml")
	assert.NoError(t, os.WriteFile(tmpDevnetPath, []byte(contexts.ContextYamls[contexts.LatestVersion]), 0644))

	// Run loader with the new base path
	cfg, err := LoadConfigWithContextConfigFromPath("devnet", tmpDir)
	assert.NoError(t, err)

	assert.Equal(t, "my-avs", cfg.Config.Project.Name)
	assert.Equal(t, "0.1.0", cfg.Config.Project.Version)
	assert.Equal(t, "devnet", cfg.Config.Project.Context)

	assert.Equal(t, "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", cfg.Context["devnet"].DeployerPrivateKey)
	assert.Equal(t, "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a", cfg.Context["devnet"].AppDeployerPrivateKey)

	assert.Equal(t, "keystores/operator1.keystore.json", cfg.Context["devnet"].Operators[0].BlsKeystorePath)
	assert.Equal(t, "keystores/operator2.keystore.json", cfg.Context["devnet"].Operators[1].BlsKeystorePath)
	assert.Equal(t, "testpass", cfg.Context["devnet"].Operators[0].BlsKeystorePassword)
	assert.Equal(t, "testpass", cfg.Context["devnet"].Operators[0].BlsKeystorePassword)
	assert.Equal(t, "1000ETH", cfg.Context["devnet"].Operators[0].Stake)
	assert.Equal(t, "1000ETH", cfg.Context["devnet"].Operators[1].Stake)

	assert.Equal(t, "devnet", cfg.Context["devnet"].Name)
	assert.Equal(t, "http://localhost:8545", cfg.Context["devnet"].Chains["l1"].RPCURL)
	assert.Equal(t, "http://localhost:8545", cfg.Context["devnet"].Chains["l2"].RPCURL)
	assert.Equal(t, 22475020, cfg.Context["devnet"].Chains["l1"].Fork.Block)
	assert.Equal(t, 22475020, cfg.Context["devnet"].Chains["l1"].Fork.Block)

	assert.Equal(t, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", cfg.Context["devnet"].Avs.Address)
	assert.Equal(t, "0x0123456789abcdef0123456789ABCDEF01234567", cfg.Context["devnet"].Avs.RegistrarAddress)
	assert.Equal(t, "https://my-org.com/avs/metadata.json", cfg.Context["devnet"].Avs.MetadataUri)

}

func LoadConfigWithContextConfigWithPath(contextName string, configDir string) (*common.ConfigWithContextConfig, error) {
	// Load base config
	configPath := filepath.Join(configDir, common.BaseConfig)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}

	var cfg common.ConfigWithContextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load requested context file
	contextFile := filepath.Join(configDir, "contexts", contextName+".yaml")
	ctxData, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context %q file: %w", contextName, err)
	}

	var wrapper struct {
		Version string                    `yaml:"version"`
		Context common.ChainContextConfig `yaml:"context"`
	}

	if err := yaml.Unmarshal(ctxData, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse context file %q: %w", contextFile, err)
	}

	cfg.Context = map[string]common.ChainContextConfig{
		contextName: wrapper.Context,
	}

	return &cfg, nil
}
