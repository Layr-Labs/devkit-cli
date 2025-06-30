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
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	releasemanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ReleaseManager"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// OperatorSetRelease represents the data for each operator set
type OperatorSetRelease struct {
	Digest   string `json:"digest"`
	Registry string `json:"registry"`
}

// parseOperatorSetMapping parses the JSON output from the release script
func parseOperatorSetMapping(jsonOutput string) (map[string][]OperatorSetRelease, error) {
	// Parse the JSON structure: {"0": [{"digest": "...", "registry": "..."}], "1": [...]}
	var releases map[string][]OperatorSetRelease
	if err := json.Unmarshal([]byte(jsonOutput), &releases); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operator set mapping: %w", err)
	}

	// Validate that each operator set has at least one artifact
	for opSetId, dataArray := range releases {
		if len(dataArray) == 0 {
			return nil, fmt.Errorf("operator set %s has empty data array", opSetId)
		}
	}

	return releases, nil
}

// updateContextWithDigest updates the context YAML file with the digest after successful release
func updateContextWithDigest(digest string) error {
	// Load the context yaml file
	contextPath := filepath.Join("config", "contexts", "devnet.yaml") // TODO: make context configurable
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
func updateContextWithVersion(version string) error {
	// Load the context yaml file
	yamlPath, rootNode, contextNode, err := common.LoadContext(devnet.DEVNET_CONTEXT)
	if err != nil {
		return err
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

// ReleaseCommand defines the "release" command
var ReleaseCommand = &cli.Command{
	Name:  "release",
	Usage: "Manage AVS releases and artifacts",
	Subcommands: []*cli.Command{
		{
			Name:  "publish",
			Usage: "Publish a new AVS release",
			Flags: append(common.GlobalFlags, []cli.Flag{
				&cli.Int64Flag{
					Name:     "upgrade-by-time",
					Usage:    "Unix timestamp by which the upgrade must be completed",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "registry",
					Usage: "Registry to use for the release. If not provided, will use registry from context",
				},
			}...),
			Action: publishReleaseAction,
		},
	},
}

// processOperatorSets processes each operator set and publishes releases on chain
func processOperatorSetsAndPublishReleaseOnChain(cCtx *cli.Context, logger iface.Logger, operatorSetMapping map[string][]OperatorSetRelease, avs string, upgradeByTime int64, registry string) error {
	// Publish releases for each operator set
	for opSetId, opSetDataArray := range operatorSetMapping {
		opSetIdInt, err := strconv.ParseUint(opSetId, 10, 32)
		if err != nil {
			logger.Warn("Failed to parse operator set ID %s: %v", opSetId, err)
			continue
		}

		logger.Info("Processing operator set %s with %d artifacts:", opSetId, len(opSetDataArray))

		// Create artifacts array for this operator set
		var artifacts []releasemanager.IReleaseManagerTypesArtifact
		for i, opSetData := range opSetDataArray {
			logger.Info("Artifact %d:", i+1)
			logger.Info("Digest: %s", opSetData.Digest)
			logger.Info("Registry: %s", opSetData.Registry)

			// this means this is the component
			if opSetData.Registry == registry {
				err := updateContextWithDigest(opSetData.Digest)
				if err != nil {
					logger.Warn("Failed to update context with digest for operator set %s artifact %d: %v", opSetId, i+1, err)
					continue
				}
				logger.Info("Successfully updated context with digest for operator set %s artifact %d", opSetId, i+1)
			}

			// Convert digest to bytes32
			digestBytes, err := hexStringToBytes32(opSetData.Digest)
			if err != nil {
				logger.Warn("Failed to convert digest to bytes32 for operator set %s artifact %d: %v", opSetId, i+1, err)
				continue
			}

			artifact := releasemanager.IReleaseManagerTypesArtifact{
				Digest:      digestBytes,
				RegistryUrl: opSetData.Registry,
			}
			artifacts = append(artifacts, artifact)
		}

		if len(artifacts) == 0 {
			logger.Warn("No valid artifacts for operator set %s, skipping", opSetId)
			continue
		}

		logger.Info("Publishing release for operator set %s with %d artifacts...", opSetId, len(artifacts))
		if err := publishReleaseToReleaseManagerAction(cCtx.Context, logger, avs, uint32(opSetIdInt), upgradeByTime, artifacts); err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				logger.Warn("Failed to publish release for operator set %s: %v", opSetId, err)
				logger.Info("Check if devnet is running and try again")
				return err
			}
		}
		logger.Info("Successfully published release for operator set %s", opSetId)
	}

	return nil
}

func publishReleaseAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Get values from flags
	upgradeByTime := cCtx.Int64("upgrade-by-time")
	registry := cCtx.String("registry")

	// Get build artifact from context first to read registry URL and version
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT) // TODO: make context configurable
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	if cfg.Context[devnet.DEVNET_CONTEXT].Artifact == nil {
		return fmt.Errorf("no artifact found in context. Please run 'devkit avs build' first")
	}

	artifact := cfg.Context[devnet.DEVNET_CONTEXT].Artifact
	avs := cfg.Context[devnet.DEVNET_CONTEXT].Avs.Address
	// Validate AVS address
	if avs == "" {
		return fmt.Errorf("AVS addressempty in context")
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
		return fmt.Errorf("no component found in context. Please run 'devkit avs build' first")
	}

	logger.Info("Publishing AVS release...")
	logger.Info("AVS address: %s", avs)
	logger.Info("Version: %s", version)
	logger.Info("Registry: %s", registry)
	logger.Info("UpgradeByTime: %s", time.Unix(upgradeByTime, 0).Format(time.RFC3339))

	// Call release.sh script to check if image has changed
	scriptsDir := filepath.Join(".hourglass", "scripts")
	releaseScriptPath := filepath.Join(scriptsDir, "release.sh")

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
	component := cfg.Context[devnet.DEVNET_CONTEXT].Artifact.Component
	// Execute release script with version and registry
	releaseCmd := exec.CommandContext(cCtx.Context, "bash", releaseScriptPath,
		"--version", version,
		"--registry", finalRegistry,
		"--image", component,
		"--original-image-id", artifact.ArtifactId)
	releaseCmd.Stderr = os.Stderr // Show stderr in terminal

	// Capture stdout to get the operator set mapping JSON
	output, err := releaseCmd.Output()
	if err != nil {
		// Script returned non-zero exit code, meaning image has changed
		logger.Info("Image has changed since last build. Please ensure your build is stable before releasing.")
		logger.Info("Run 'devkit avs build' again and verify no code changes were made.")
		return err
	}

	// update version in context, by incrementing it
	version, err = incrementVersion(version)
	if err != nil {
		return fmt.Errorf("failed to increment version: %w", err)
	}

	// Update version in context
	if err := updateContextWithVersion(version); err != nil {
		return fmt.Errorf("failed to update context with version: %w", err)
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
	if err := processOperatorSetsAndPublishReleaseOnChain(cCtx, logger, operatorSetMapping, avs, upgradeByTime, finalRegistry); err != nil {
		return err
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

func publishReleaseToReleaseManagerAction(ctx context.Context, logger iface.Logger, avs string, operatorSetId uint32, upgradeByTime int64, artifacts []releasemanager.IReleaseManagerTypesArtifact) error {

	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for operator registration: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}

	l1Cfg, ok := envCtx.Chains[devnet.L1]
	if !ok {
		return fmt.Errorf("failed to get l1 chain config for context '%s'", devnet.DEVNET_CONTEXT)
	}

	client, err := ethclient.Dial(l1Cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("failed to connect to L1 RPC: %w", err)
	}
	defer client.Close()

	operatorSetId = uint32(operatorSetId)
	upgradeByTime = int64(upgradeByTime)

	avsPrivateKey := envCtx.Avs.AVSPrivateKey
	if avsPrivateKey == "" {
		return fmt.Errorf("AVS private key not found in context")
	}
	// Trim 0x
	avsPrivateKey = strings.TrimPrefix(avsPrivateKey, "0x")
	_, _, _, _, _, _, releaseManagerAddress := devnet.GetEigenLayerAddresses(cfg)

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
		logger,
	)
	if err != nil {
		return fmt.Errorf("failed to create contract caller: %w", err)
	}

	// Use the artifacts array passed in
	err = contractCaller.PublishRelease(ctx, ethcommon.HexToAddress(avs), artifacts, operatorSetId, upgradeByTime)
	if err != nil {
		return fmt.Errorf("failed to publish release: %w", err)
	}

	logger.Info("Successfully published release to ReleaseManager contract")
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
