package commands

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
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

// BuildConfig represents the structure of .hourglass/build.yaml
type BuildConfig struct {
	Container struct {
		Registry string `yaml:"registry"`
		Image    string `yaml:"image"`
		Version  string `yaml:"version"`
	} `yaml:"container"`
}

// OperatorSetData represents the data for each operator set
type OperatorSetData struct {
	Digest      string `json:"digest"`
	RegistryUrl string `json:"registry_url"`
}

// parseOperatorSetMapping parses the JSON output from the release script
func parseOperatorSetMapping(jsonOutput string) (map[string][]OperatorSetData, error) {
	// Parse the JSON structure: {"0": [{"digest": "...", "registry_url": "..."}], "1": [...]}
	var rawMapping map[string][]OperatorSetData
	if err := json.Unmarshal([]byte(jsonOutput), &rawMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operator set mapping: %w", err)
	}

	// Validate that each operator set has at least one artifact
	for opSetId, dataArray := range rawMapping {
		if len(dataArray) == 0 {
			return nil, fmt.Errorf("operator set %s has empty data array", opSetId)
		}
	}

	return rawMapping, nil
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
	artifactsSection := common.GetChildByKey(contextSection, "artifacts")
	if artifactsSection == nil {
		return fmt.Errorf("artifacts section not found in context")
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

// validateAndFormatVersion validates and formats a literal version string
func validateAndFormatVersion(version string) (string, error) {
	// Remove 'v' prefix if present for processing
	cleanVersion := strings.TrimPrefix(version, "v")

	// Basic semantic version validation
	if !strings.Contains(cleanVersion, ".") {
		return "", fmt.Errorf("version should follow semantic versioning format (e.g., 1.0.0)")
	}

	// Return without 'v' prefix
	return cleanVersion, nil
}

// performMultiArchBuildAndPush performs multi-architecture build and push using buildx
func performMultiArchBuildAndPush(ctx context.Context, logger iface.Logger, registryUrl, version string, operatorSetId uint64) (string, error) {
	// Get project name from current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(cwd)

	// Image name as {project-name}-performer-op-set-{operator-set-id}
	imageName := fmt.Sprintf("%s-performer-op-set-%d", projectName, operatorSetId)

	fullImageName := fmt.Sprintf("%s/%s:%s", registryUrl, imageName, version)

	logger.Info("Building multi-architecture image: %s", fullImageName)
	logger.Info("Platforms: linux/amd64,linux/arm64")

	// Setup buildx for multi-platform
	logger.Info("Setting up multi-platform builder...")
	if err := setupBuildx(ctx); err != nil {
		return "", fmt.Errorf("failed to setup buildx: %w", err)
	}

	// Build and push multi-arch
	logger.Info("Building and pushing multi-architecture image...")
	buildCmd := exec.CommandContext(ctx, "docker", "buildx", "build",
		"--platform", "linux/amd64,linux/arm64",
		"--tag", fullImageName,
		"--push",
		".")

	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		logger.Error("Multi-arch build failed: %s", string(buildOutput))
		return "", fmt.Errorf("failed to build and push multi-arch image: %w", err)
	}

	logger.Info("Built and pushed multi-architecture image: %s", fullImageName)

	// Get the Image Index digest (first Digest line is the manifest list digest)
	logger.Info("Getting Image Index digest...")
	inspectCmd := exec.CommandContext(ctx, "docker", "buildx", "imagetools", "inspect", fullImageName)
	inspectOutput, err := inspectCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect image: %w", err)
	}

	// Parse the digest from the output - get the FIRST "Digest:" line (Image Index)
	inspectStr := string(inspectOutput)
	lines := strings.Split(inspectStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Digest:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				digest := parts[1]
				logger.Info("📋 Image Index Digest: %s", digest)
				return digest, nil
			}
		}
	}

	return "", fmt.Errorf("could not find digest in inspect output")
}

// setupBuildx sets up the buildx builder for multi-platform builds
func setupBuildx(ctx context.Context) error {
	// Check if multiarch builder exists
	inspectCmd := exec.CommandContext(ctx, "docker", "buildx", "inspect", "multiarch")
	if inspectCmd.Run() != nil {
		// Create multi-platform builder
		createCmd := exec.CommandContext(ctx, "docker", "buildx", "create", "--name", "multiarch", "--driver", "docker-container", "--use")
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create buildx builder: %w", err)
		}

		// Bootstrap the builder
		bootstrapCmd := exec.CommandContext(ctx, "docker", "buildx", "inspect", "--bootstrap")
		if err := bootstrapCmd.Run(); err != nil {
			return fmt.Errorf("failed to bootstrap buildx builder: %w", err)
		}
	} else {
		// Use existing builder
		useCmd := exec.CommandContext(ctx, "docker", "buildx", "use", "multiarch")
		if err := useCmd.Run(); err != nil {
			return fmt.Errorf("failed to use buildx builder: %w", err)
		}
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
				&cli.StringFlag{
					Name:     "avs",
					Usage:    "AVS contract address",
					Required: true,
				},
				&cli.Uint64Flag{
					Name:     "operator-set-id",
					Usage:    "Operator set ID",
					Required: true,
				},
				&cli.Int64Flag{
					Name:     "upgrade-by-time",
					Usage:    "Unix timestamp by which the upgrade must be completed",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "version",
					Usage: "Version to release (e.g., 1.0.0). If not provided, will use version from context",
				},
			}...),
			Action: publishReleaseAction,
		},
	},
}

func publishReleaseAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Get values from flags
	avs := cCtx.String("avs")
	operatorSetId := cCtx.Uint64("operator-set-id")
	upgradeByTime := cCtx.Int64("upgrade-by-time")
	version := cCtx.String("version")

	// Validate AVS address
	if avs == "" {
		return fmt.Errorf("AVS address cannot be empty")
	}

	// Get build artifact from context first to read registry URL and version
	cfg, err := common.LoadConfigWithContextConfig("devnet") // TODO: make context configurable
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	if cfg.Context["devnet"].Artifact == nil {
		return fmt.Errorf("no artifact found in context. Please run 'devkit avs build' first")
	}

	artifact := cfg.Context["devnet"].Artifact

	// Handle version - if not provided, read from context
	if version == "" {
		version = artifact.Version
		if version == "" {
			return fmt.Errorf("no version specified and no version found in context")
		}
		// Validate provided version
		version, err = validateAndFormatVersion(version)
		if err != nil {
			return fmt.Errorf("invalid version format in context: %w", err)
		}
		logger.Info("No version specified, using version from context: %s", version)
	} else {
		// Validate provided version
		version, err = validateAndFormatVersion(version)
		if err != nil {
			return fmt.Errorf("invalid version format: %w", err)
		}
		logger.Info("Using provided version: %s", version)
	}

	// Validate operator set ID fits in uint32 range (since it gets cast to uint32 later)
	if operatorSetId > math.MaxUint32 {
		return fmt.Errorf("operator set ID %d exceeds maximum value for uint32 (%d)", operatorSetId, math.MaxUint32)
	}

	// Validate upgradeByTime is in the future
	if upgradeByTime <= time.Now().Unix() {
		return fmt.Errorf("upgrade-by-time timestamp %d must be in the future (current time: %d)", upgradeByTime, time.Now().Unix())
	}

	logger.Info("Publishing AVS release...")
	logger.Info("  AVS address: %s", avs)
	logger.Info("  Version: %s", version)
	logger.Info("  Operator Set ID: %d", operatorSetId)
	logger.Info("  Registry URL: %s", artifact.RegistryUrl)
	logger.Info("  UpgradeByTime: %s", time.Unix(upgradeByTime, 0).Format(time.RFC3339))

	// Check if component is present (from local build)
	if artifact.Component == "" {
		logger.Info("No artifact found in context. Please run 'devkit avs build' first to create a local build.")
		return nil
	}

	// Call release.sh script to check if image has changed
	logger.Info("Checking if image has changed since last build...")
	scriptsDir := filepath.Join(".hourglass", "scripts")
	releaseScriptPath := filepath.Join(scriptsDir, "release.sh")

	// Execute release script with version
	releaseCmd := exec.CommandContext(cCtx.Context, "bash", releaseScriptPath, "--version", version)
	releaseCmd.Stderr = os.Stderr // Show stderr in terminal

	// Capture stdout to get the operator set mapping JSON
	output, err := releaseCmd.Output()
	if err != nil {
		// Script returned non-zero exit code, meaning image has changed
		logger.Info("Image has changed since last build. Please ensure your build is stable before releasing.")
		logger.Info("Run 'devkit avs build' again and verify no code changes were made.")
		return nil
	}

	logger.Info("Image unchanged - proceeding with release...")

	// In the new flow, digest is expected to be empty since build doesn't push
	// We need to handle multi-arch build and push during release process
	logger.Info("Starting multi-architecture build and push process...")

	// Implement multi-arch build, push to registry, and get digest
	finalRegistryUrl := artifact.RegistryUrl

	if finalRegistryUrl == "" {
		return fmt.Errorf("registry URL not found in context. Please ensure it's set in your context configuration")
	}

	// Perform multi-arch build and push to get Image Index digest
	imageDigest, err := performMultiArchBuildAndPush(cCtx.Context, logger, finalRegistryUrl, version, operatorSetId)
	if err != nil {
		return fmt.Errorf("failed to perform multi-arch build and push: %w", err)
	}

	logger.Info("Multi-architecture build completed")
	logger.Info("Registry URL: %s", finalRegistryUrl)
	logger.Info("Image digest: %s", imageDigest)

	// Create artifact array with the Image Index digest
	logger.Info("Creating artifact for ReleaseManager...")
	digestBytes, err := hexStringToBytes32(imageDigest)
	if err != nil {
		return fmt.Errorf("failed to convert digest to bytes32: %w", err)
	}

	artifactArray := []releasemanager.IReleaseManagerTypesArtifact{
		{
			Digest:      digestBytes,
			RegistryUrl: finalRegistryUrl,
		},
	}

	logger.Info("Artifact created:")
	logger.Info("  Digest: %s", imageDigest)
	logger.Info("  Registry URL: %s", finalRegistryUrl)
	logger.Info("Publishing to ReleaseManager contract...")

	if err := PublishReleaseToReleaseManagerAction(cCtx.Context, logger, avs, uint32(operatorSetId), upgradeByTime, artifactArray); err != nil {
		logger.Error("Failed to publish release to ReleaseManager: %s", err)
		return fmt.Errorf("failed to publish release to ReleaseManager: %w", err)
	}

	// Update context with the digest after successful release
	logger.Info("Updating context with release digest...")
	if err := updateContextWithDigest(imageDigest); err != nil {
		logger.Warn("Failed to update context with digest: %v", err)
		// Don't fail the release if context update fails
	} else {
		logger.Info("Successfully updated context with digest: %s", imageDigest)
	}

	// Parse the operator set mapping JSON from script output
	logger.Info("Processing operator set mapping from script output...")
	operatorSetMapping, err := parseOperatorSetMapping(string(output))
	if err != nil {
		logger.Warn("Failed to parse operator set mapping in hourglass release script: %v", err)
		return nil
	}

	logger.Info("Retrieved operator set mapping with %d operator sets", len(operatorSetMapping))

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
			logger.Info("  Artifact %d:", i+1)
			logger.Info("    Digest: %s", opSetData.Digest)
			logger.Info("    Registry URL: %s", opSetData.RegistryUrl)

			// Convert digest to bytes32
			digestBytes, err := hexStringToBytes32(opSetData.Digest)
			if err != nil {
				logger.Warn("Failed to convert digest to bytes32 for operator set %s artifact %d: %v", opSetId, i+1, err)
				continue
			}

			artifact := releasemanager.IReleaseManagerTypesArtifact{
				Digest:      digestBytes,
				RegistryUrl: opSetData.RegistryUrl,
			}
			artifacts = append(artifacts, artifact)
		}

		if len(artifacts) == 0 {
			logger.Warn("No valid artifacts for operator set %s, skipping", opSetId)
			continue
		}

		logger.Info("Publishing release for operator set %s with %d artifacts...", opSetId, len(artifacts))
		if err := PublishReleaseToReleaseManagerAction(cCtx.Context, logger, avs, uint32(opSetIdInt), upgradeByTime, artifacts); err != nil {
			logger.Warn("Failed to publish release for operator set %s: %v", opSetId, err)
			continue
		}
		logger.Info("Successfully published release for operator set %s", opSetId)
	}

	return nil
}

func PublishReleaseToReleaseManagerAction(ctx context.Context, logger iface.Logger, avs string, operatorSetId uint32, upgradeByTime int64, artifacts []releasemanager.IReleaseManagerTypesArtifact) error {

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
