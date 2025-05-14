package common_test

import (
	"devkit-cli/pkg/common"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestLoadConfigWithContextConfig_FromCopiedTempFile(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()
	tmpYamlPath := filepath.Join(tmpDir, "config.yaml")

	// Copy config/config.yaml to tempDir
	srcConfigPath := filepath.Join("..", "..", "config", "config.yaml")
	common.CopyFileTesting(t, srcConfigPath, tmpYamlPath)

	// Copy config/contexts/devnet.yaml to tempDir/config/contexts
	tmpContextDir := filepath.Join(tmpDir, "config", "contexts")
	assert.NoError(t, os.MkdirAll(tmpContextDir, 0755))

	srcDevnetPath := filepath.Join("..", "..", "config", "contexts", "devnet.yaml")
	tmpDevnetPath := filepath.Join(tmpContextDir, "devnet.yaml")
	common.CopyFileTesting(t, srcDevnetPath, tmpDevnetPath)

	// Run loader with the new base path
	cfg, err := LoadConfigWithContextConfigFromPath("devnet", tmpDir)
	assert.NoError(t, err)

	assert.Equal(t, "my-avs", cfg.Config.Project.Name)
	assert.Equal(t, "0.1.0", cfg.Config.Project.Version)
	assert.Equal(t, "devnet", cfg.Config.Project.Context)
}

func LoadConfigWithContextConfigFromPath(contextName string, config_directory_path string) (*common.ConfigWithContextConfig, error) {
	// Load base config
	data, err := os.ReadFile(filepath.Join(config_directory_path, "config.yaml"))
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
