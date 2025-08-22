package commands

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/artifact"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	releasemanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ReleaseManager"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func publishReleaseAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Get values from flags
	upgradeByTime := cCtx.Int64("upgrade-by-time")
	registry := cCtx.String("registry")
	contextName := cCtx.String("context")

	// Get build artifact from context first to read registry URL and version
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	// Extract context details
	if cfg.Context[contextName].Artifact == nil {
		return fmt.Errorf("no artifact found in context. Please run 'devkit avs build' first")
	}

	artifact := cfg.Context[contextName].Artifact
	avs := cfg.Context[contextName].Avs.Address
	// Validate AVS address
	if avs == "" {
		return fmt.Errorf("AVS addressempty in context")
	}

	// Check if metadata URI is set for any operator set before proceeding
	logger.Info("Checking AVS metadata URI...")
	if err := checkMetadataURIExists(logger, contextName, cfg, avs); err != nil {
		return err
	}

	version := artifact.Version
	// first time publishing, version is empty
	if version == "" {
		version = "0"
	}

	// Validate upgradeByTime is in the future
	if upgradeByTime <= time.Now().Unix() {
		return fmt.Errorf("upgrade-by-time timestamp %d must be in the future (current time: %d)", upgradeByTime, time.Now().Unix())
	}

	if artifact.Component == "" {
		return fmt.Errorf("no artifact found to release. Please run 'devkit avs build' first")
	}

	logger.Info("Publishing AVS release...")
	logger.Info("AVS address: %s", avs)
	logger.Info("Version: %s", version)
	logger.Info("Registry: %s", registry)
	logger.Info("UpgradeByTime: %s", time.Unix(upgradeByTime, 0).Format(time.RFC3339))

	// Call release.sh script to check if image has changed
	scriptsDir := filepath.Join(".devkit", "scripts")
	releaseScriptPath := filepath.Join(scriptsDir, "release")

	// Get registry from flag or context
	finalRegistry := registry
	if finalRegistry == "" {
		if artifact.Registry == "" {
			return fmt.Errorf("no registry found in context")
		}
		finalRegistry = artifact.Registry
		logger.Info("Using registry from context: %s", finalRegistry)
	} else {
		logger.Info("Using provided registry: %s", finalRegistry)
	}
	component := cfg.Context[contextName].Artifact.Component
	// Execute release script with version and registry
	releaseCmd := exec.CommandContext(cCtx.Context, "bash", releaseScriptPath,
		"--version", version,
		"--registry", finalRegistry,
		"--image", component)
	releaseCmd.Stderr = os.Stderr

	// Add environment variable for context
	releaseCmd.Env = append(os.Environ(), fmt.Sprintf("CONTEXT_NAME=%s", contextName))

	// Capture stdout to get the operator set mapping JSON
	output, err := releaseCmd.Output()
	if err != nil {
		// Script returned non-zero exit code, meaning image has changed
		return fmt.Errorf("failed to release artifact: %w", err)
	}

	// Parse the operator set mapping JSON from script output
	logger.Info("Processing operator set mapping from script output...")
	operatorSetMapping, err := parseOperatorSetMapping(string(output))
	if err != nil {
		logger.Warn("Failed to parse operator set mapping in hourglass release script: %v", err)
		return err
	}

	logger.Info("Retrieved operator set mapping with %d operator sets", len(operatorSetMapping))

	// Publish releases for each operator set
	if err := processOperatorSetsAndPublishReleaseOnChain(cCtx, logger, contextName, operatorSetMapping, avs, upgradeByTime, finalRegistry, version); err != nil {
		return err
	}

	// Only increment version after successful publishing
	newVersion, err := incrementVersion(version)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}

	// Update version in context
	if err := updateContextWithVersion(contextName, newVersion); err != nil {
		return fmt.Errorf("failed to update context with version: %w", err)
	}

	logger.Info("Successfully published release and incremented version to %s", newVersion)

	return nil
}

// processOperatorSets processes each operator set and publishes releases on chain
func processOperatorSetsAndPublishReleaseOnChain(
	cCtx *cli.Context,
	logger iface.Logger,
	contextName string,
	operatorSetMapping map[string]OperatorSetRelease,
	avs string,
	upgradeByTime int64,
	registry string,
	version string,
) error {
	// Create OCI artifact builder
	ociBuilder := artifact.NewOCIArtifactBuilder(logger)

	// Get AVS name from context for artifact naming
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	// Get AVS name from project configuration
	avsName := cfg.Config.Project.Name
	if avsName == "" {
		return fmt.Errorf("project name not found in config.yaml. Please ensure config.project.name is set")
	}

	// Publish releases for each operator set
	for opSetId, opSetData := range operatorSetMapping {
		opSetIdInt, err := strconv.ParseUint(opSetId, 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse operator set ID %s: %v", opSetId, err)
		}

		logger.Info("Processing operator set %s", opSetId)
		logger.Info("Digest: %s", opSetData.Digest)
		logger.Info("Registry: %s", opSetData.Registry)

		// Create OCI artifact for runtime spec
		logger.Info("Creating OCI artifact for runtime spec...")
		artifactTag := fmt.Sprintf("opset-%s-v%s", opSetId, version)

		// Create and push OCI artifact
		ociDigest, err := ociBuilder.CreateEigenRuntimeArtifact(
			[]byte(opSetData.RuntimeSpec),
			registry,
			avsName,
			artifactTag,
		)
		if err != nil {
			logger.Error("Failed to create OCI artifact for operator set %s: %v", opSetId, err)
			return fmt.Errorf("failed to create OCI artifact: %w", err)
		}

		finalDigest := ociDigest
		finalRegistry := registry
		logger.Info("Successfully created OCI artifact with digest: %s", finalDigest)

		// Update context with digest
		err = updateContextWithDigest(contextName, finalDigest)
		if err != nil {
			return fmt.Errorf("failed to update context with digest for operator set %s: %v", opSetId, err)
		}
		logger.Info("Successfully updated context with digest for operator set %s", opSetId)

		// Convert digest to bytes32
		digestBytes, err := hexStringToBytes32(finalDigest)
		if err != nil {
			logger.Warn("Failed to convert digest to bytes32 for operator set %s: %v", opSetId, err)
			continue
		}

		// Create artifact for this operator set
		artifact := releasemanager.IReleaseManagerTypesArtifact{
			Digest:   digestBytes,
			Registry: finalRegistry,
		}
		artifacts := []releasemanager.IReleaseManagerTypesArtifact{artifact}

		logger.Info("Publishing release for operator set %s...", opSetId)
		if err := publishReleaseToReleaseManagerAction(cCtx.Context, logger, contextName, avs, uint32(opSetIdInt), upgradeByTime, artifacts); err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				logger.Warn("Failed to publish release for operator set %s: %v", opSetId, err)
				logger.Info("Check if devnet is running and try again")
			}
			return err
		}
		logger.Info("Successfully published release for operator set %s", opSetId)
	}

	return nil
}

func incrementVersion(version string) (string, error) {
	// version is a int
	versionInt, err := strconv.Atoi(version)
	if err != nil {
		return "", fmt.Errorf("failed to convert version to int: %w", err)
	}
	versionInt++
	return strconv.Itoa(versionInt), nil
}

// checkMetadataURIExists checks if metadata URI is set for at least one operator set
func checkMetadataURIExists(logger iface.Logger, contextName string, cfg *common.ConfigWithContextConfig, avsAddress string) error {
	// Get L1 chain config
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	// Connect to L1
	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	// Get AVS private key
	avsPrivateKey := envCtx.Avs.AVSPrivateKey
	if avsPrivateKey == "" {
		return fmt.Errorf("AVS private key not found in context")
	}
	avsPrivateKey = strings.TrimPrefix(avsPrivateKey, "0x")

	// Get contract addresses
	_, _, _, _, _, _, _, releaseManagerAddress := common.GetEigenLayerAddresses(contextName, cfg)

	// Create contract caller
	contractCaller, err := common.NewContractCaller(
		avsPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(releaseManagerAddress),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	// Check metadata URI for common operator sets (0 and 1)
	metadataFound := false
	operatorSetsToCheck := []uint32{0, 1}

	for _, opSetId := range operatorSetsToCheck {
		uri, err := contractCaller.GetReleaseMetadataUri(ethcommon.HexToAddress(avsAddress), opSetId)
		if err != nil {
			logger.Debug("Error checking metadata URI for operator set %d: %v", opSetId, err)
			continue
		}
		if uri != "" {
			logger.Info("Found metadata URI for operator set %d: %s", opSetId, uri)
			metadataFound = true
		}
	}

	if !metadataFound {
		return fmt.Errorf("no release metadata URI found for AVS %s. Please set metadata URI using:\n  devkit avs release uri --metadata-uri <uri> --operator-set-id <id>", avsAddress)
	}

	return nil
}

func publishReleaseToReleaseManagerAction(
	ctx context.Context,
	logger iface.Logger,
	contextName string,
	avs string,
	operatorSetId uint32,
	upgradeByTime int64,
	artifacts []releasemanager.IReleaseManagerTypesArtifact,
) error {
	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	avsPrivateKey := envCtx.Avs.AVSPrivateKey
	if avsPrivateKey == "" {
		return fmt.Errorf("AVS private key not found in context")
	}
	// Trim 0x
	avsPrivateKey = strings.TrimPrefix(avsPrivateKey, "0x")
	_, _, _, _, _, _, _, releaseManagerAddress := common.GetEigenLayerAddresses(contextName, cfg)

	contractCaller, err := common.NewContractCaller(
		avsPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(releaseManagerAddress),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}
	logger.Info("Publishing operator set mapping from script output...")
	err = contractCaller.PublishRelease(ctx, ethcommon.HexToAddress(avs), artifacts, operatorSetId, uint32(upgradeByTime))
	if err != nil {
		return fmt.Errorf("failed to publish release: %w", err)
	}

	logger.Info("Successfully published release to ReleaseManager contract")
	return nil
}

// setReleaseMetadataURIAction handles the "release uri" subcommand
func setReleaseMetadataURIAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Get values from flags
	metadataURI := cCtx.String("metadata-uri")
	operatorSetID := cCtx.Uint("operator-set-id")
	avsAddressStr := cCtx.String("avs-address")
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	// Get AVS address from flag or context
	var avsAddress string
	if avsAddressStr != "" {
		avsAddress = avsAddressStr
	} else {
		avsAddress = cfg.Context[contextName].Avs.Address
		if avsAddress == "" {
			return fmt.Errorf("AVS address not provided and not found in context")
		}
	}

	logger.Info("Setting release metadata URI...")
	logger.Info("AVS address: %s", avsAddress)
	logger.Info("Operator Set ID: %d", operatorSetID)
	logger.Info("Metadata URI: %s", metadataURI)

	// Get L1 chain config
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	l1Cfg, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", contextName)
	}

	// Connect to L1
	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	// Get AVS private key
	avsPrivateKey := envCtx.Avs.AVSPrivateKey
	if avsPrivateKey == "" {
		return fmt.Errorf("AVS private key not found in context")
	}
	avsPrivateKey = strings.TrimPrefix(avsPrivateKey, "0x")

	// Get contract addresses
	_, _, _, _, _, _, _, releaseManagerAddress := common.GetEigenLayerAddresses(contextName, cfg)

	// Create contract caller
	contractCaller, err := common.NewContractCaller(
		avsPrivateKey,
		big.NewInt(int64(l1Cfg.ChainID)),
		client,
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(""),
		ethcommon.HexToAddress(releaseManagerAddress),
		ethcommon.HexToAddress(""),
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	// Set release metadata URI
	err = contractCaller.SetReleaseMetadata(
		cCtx.Context,
		metadataURI,
		ethcommon.HexToAddress(avsAddress),
		uint32(operatorSetID),
	)
	if err != nil {
		return fmt.Errorf("failed to set release metadata URI: %w", err)
	}

	logger.Info("Successfully set release metadata URI for operator set %d", operatorSetID)
	return nil
}

// hexStringToBytes32 converts a hex string (like "sha256:abc123...") to [32]byte
func hexStringToBytes32(hexStr string) ([32]byte, error) {
	var result [32]byte

	// Remove "sha256:" prefix if present
	hexStr = strings.TrimPrefix(hexStr, "sha256:")

	// Decode hex string
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return result, fmt.Errorf("failed to decode hex string: %w", err)
	}

	// Ensure we have exactly 32 bytes
	if len(bytes) != 32 {
		return result, fmt.Errorf("digest must be exactly 32 bytes, got %d", len(bytes))
	}

	copy(result[:], bytes)
	return result, nil
}

// parseOperatorSetMapping parses the JSON output from the release script
func parseOperatorSetMapping(jsonOutput string) (map[string]OperatorSetRelease, error) {
	// Parse the JSON structure: {"0": {"digest": "...", "registry": "...", "runtimeSpec": "..."}, "1": {...}}
	var releases map[string]OperatorSetRelease
	if err := json.Unmarshal([]byte(jsonOutput), &releases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operator set mapping: %w", err)
	}

	// Validate that we have at least one operator set
	if len(releases) == 0 {
		return nil, fmt.Errorf("no operator sets found in release output")
	}

	return releases, nil
}

// updateContextWithDigest updates the context YAML file with the digest after successful release
func updateContextWithDigest(contextName, digest string) error {
	// Load the context yaml file
	contextPath := filepath.Join("config", "contexts", fmt.Sprintf("%s.yaml", contextName))
	contextNode, err := common.LoadYAML(contextPath)
	if err != nil {
		return fmt.Errorf("failed to load context yaml: %w", err)
	}

	// Get the root node (first content node)
	rootNode := contextNode.Content[0]

	// Get the context section
	contextSection := common.GetChildByKey(rootNode, "context")
	if contextSection == nil {
		return fmt.Errorf("context section not found in yaml")
	}

	// Get or create artifacts section
	artifactsSection := common.GetChildByKey(contextSection, "artifact")
	if artifactsSection == nil {
		return fmt.Errorf("artifact section not found in context")
	}

	// Update digest field
	common.SetMappingValue(artifactsSection,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "digest"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: digest})

	// Write the updated yaml back to file
	if err := common.WriteYAML(contextPath, contextNode); err != nil {
		return fmt.Errorf("failed to write updated yaml: %w", err)
	}

	return nil
}

// updateContextWithVersion updates the context YAML file with the new version
func updateContextWithVersion(contextName, version string) error {
	// Load the context yaml file
	var yamlPath string
	var rootNode, contextNode *yaml.Node
	var err error
	if contextName == "" {
		yamlPath, rootNode, contextNode, _, err = common.LoadDefaultContext()
	} else {
		yamlPath, rootNode, contextNode, _, err = common.LoadContext(contextName)
	}
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Get or create artifact section
	artifactSection := common.GetChildByKey(contextNode, "artifact")
	if artifactSection == nil {
		artifactSection = &yaml.Node{Kind: yaml.MappingNode}
		common.SetMappingValue(contextNode,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "artifact"},
			artifactSection)
	}

	// Update version field
	common.SetMappingValue(artifactSection,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "version"},
		&yaml.Node{Kind: yaml.ScalarNode, Value: version})

	// Write the updated yaml back to file
	if err := common.WriteYAML(yamlPath, rootNode); err != nil {
		return fmt.Errorf("failed to write updated yaml: %w", err)
	}

	return nil
}
