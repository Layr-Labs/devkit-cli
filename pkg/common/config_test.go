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
	tmpYamlPath := filepath.Join(tmpDir, common.BaseConfig)

	// Copy config/config.yaml to tempDir
	assert.NoError(t, os.WriteFile(tmpYamlPath, []byte(configs.ConfigYamls[configs.LatestVersion]), 0644))

	// Copy config/contexts/devnet.yaml to tempDir/config/contexts
	tmpContextDir := filepath.Join(tmpDir, "config", "contexts")
	assert.NoError(t, os.MkdirAll(tmpContextDir, 0755))

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

	// In v0.0.6, operators use allocations instead of stake
	assert.NotEmpty(t, cfg.Context["devnet"].Operators[0].Allocations)
	assert.Equal(t, "0x7D704507b76571a51d9caE8AdDAbBFd0ba0e63d3", cfg.Context["devnet"].Operators[0].Allocations[0].StrategyAddress)
	assert.Equal(t, "stETH_Strategy", cfg.Context["devnet"].Operators[0].Allocations[0].Name)

	// Test stakers parsing - verify that stakers configuration is loaded correctly
	assert.NotEmpty(t, cfg.Context["devnet"].Stakers, "Stakers should be loaded from context")
	assert.Len(t, cfg.Context["devnet"].Stakers, 2, "Should have two stakers configured")

	staker := cfg.Context["devnet"].Stakers[0]
	assert.Equal(t, "0x23618e81E3f5cdF7f54C3d65f7FBc0aBf5B21E8f", staker.StakerAddress)
	assert.Equal(t, "0xdbda1821b80551c9d65939329250298aa3472ba22feea921c0cf5d620ea67b97", staker.StakerECDSAKey)

	// Test deposits structure
	assert.Len(t, staker.Deposits, 1, "First staker should have one deposit")

	// Test first deposit
	deposit1 := staker.Deposits[0]
	assert.Equal(t, "0x7D704507b76571a51d9caE8AdDAbBFd0ba0e63d3", deposit1.StrategyAddress)
	assert.Equal(t, "stETH_Strategy", deposit1.Name)
	assert.Equal(t, "5ETH", deposit1.DepositAmount)

	// Test operator delegation
	assert.Equal(t, "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65", staker.OperatorAddress)

	// Test second staker
	staker2 := cfg.Context["devnet"].Stakers[1]
	assert.Equal(t, "0xa0Ee7A142d267C1f36714E4a8F75612F20a79720", staker2.StakerAddress)
	assert.Equal(t, "0x2a871d0798f97d79848a013d4936a73bf4cc922c825d33c1cf7073dff6d409c6", staker2.StakerECDSAKey)
	assert.Equal(t, "0x90F79bf6EB2c4f870365E785982E1f101E93b906", staker2.OperatorAddress)
	assert.Len(t, staker2.Deposits, 1, "Second staker should have one deposit")

	assert.Equal(t, "devnet", cfg.Context["devnet"].Name)
	assert.Equal(t, "http://localhost:8545", cfg.Context["devnet"].Chains["l1"].RPCURL)
	assert.Equal(t, "http://localhost:8545", cfg.Context["devnet"].Chains["l2"].RPCURL)

	assert.Equal(t, 4056218, cfg.Context["devnet"].Chains["l1"].Fork.Block)
	assert.Equal(t, 4056218, cfg.Context["devnet"].Chains["l2"].Fork.Block)

	assert.Equal(t, "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", cfg.Context["devnet"].Avs.Address)
	assert.Equal(t, "0x0123456789abcdef0123456789ABCDEF01234567", cfg.Context["devnet"].Avs.RegistrarAddress)
	assert.Equal(t, "https://my-org.com/avs/metadata.json", cfg.Context["devnet"].Avs.MetadataUri)

	assert.Equal(t, "0x323A9FcB2De80d04B5C4B0F72ee7799100D32F0F", cfg.Context["devnet"].EigenLayer.L1.ReleaseManager)
}

func LoadConfigWithContextConfigFromPath(contextName string, config_directory_path string) (*common.ConfigWithContextConfig, error) {
	// Load base config
	data, err := os.ReadFile(filepath.Join(config_directory_path, common.BaseConfig))
	if err != nil {
		return nil, fmt.Errorf("failed to read base config: %w", err)
	}
	var cfg common.ConfigWithContextConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse base config: %w", err)
	}

	// Load requested context file
	contextFile := filepath.Join(config_directory_path, "config", "contexts", contextName+".yaml")
	ctxData, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read context %q file: %w", contextName, err)
	}

	// We expect the context file to have a top-level `context:` block
	var wrapper struct {
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
