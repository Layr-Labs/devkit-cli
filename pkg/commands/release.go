package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// ReleaseCommand defines the "release" command
var ReleaseCommand = &cli.Command{
	Name:  "release",
	Usage: "Manage AVS releases and artifacts",
	Subcommands: []*cli.Command{
		{
			Name:      "publish",
			Usage:     "Publish a new AVS release",
			ArgsUsage: "<avs> <operatorSetId> <registryUrl> <version> <deadlineTimestamp>",
			Flags: append([]cli.Flag{
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Show what would be published without executing the operation",
				},
				&cli.BoolFlag{
					Name:  "multi-arch",
					Usage: "Deploy as multi-architecture (invokes release.sh script)",
				},
			}, common.GlobalFlags...),
			Action: publishReleaseAction,
		},
	},
}

func publishReleaseAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Validate argument count
	if cCtx.NArg() != 5 {
		return fmt.Errorf("expected 5 arguments: <avs> <operatorSetId> <registryUrl> <version> <deadlineTimestamp>")
	}

	// Parse arguments
	avs := cCtx.Args().Get(0)
	operatorSetIdStr := cCtx.Args().Get(1)
	registryUrl := cCtx.Args().Get(2)
	version := cCtx.Args().Get(3)
	deadlineTimestampStr := cCtx.Args().Get(4)

	// Validate and parse operatorSetId
	operatorSetId, err := strconv.ParseUint(operatorSetIdStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid operatorSetId: %w", err)
	}

	// Validate and parse deadline timestamp
	deadlineTimestamp, err := strconv.ParseInt(deadlineTimestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid deadlineTimestamp: %w", err)
	}

	// Validate deadline is in the future
	if deadlineTimestamp <= time.Now().Unix() {
		return fmt.Errorf("deadline timestamp must be in the future")
	}

	logger.Info("Publishing AVS release...")
	logger.Info("  AVS address: %s", avs)
	logger.Info("  Operator Set ID: %d", operatorSetId)
	logger.Info("  Registry URL: %s", registryUrl)
	logger.Info("  Version: %s", version)
	logger.Info("  Deadline: %s", time.Unix(deadlineTimestamp, 0).Format(time.RFC3339))

	if cCtx.Bool("dry-run") {
		logger.Info("Dry run mode - would publish release with above parameters")
		return nil
	}

	// Get build artifacts from context
	cfg, err := common.LoadConfigWithContextConfig("devnet") // TODO: make context configurable
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	if cfg.Context["devnet"].Artifacts == nil {
		return fmt.Errorf("no artifacts found in context. Please run 'devkit avs build' first")
	}

	artifacts := cfg.Context["devnet"].Artifacts
	localImageId := artifacts.ArtifactId

	logger.Info("Found build artifacts:")
	logger.Info("  Local Image ID: %s", localImageId)

	if localImageId == "" {
		return fmt.Errorf("no image ID found in artifacts. Please run 'devkit avs build' first")
	}

	// Check if multi-arch flag is specified
	isMultiArch := cCtx.Bool("multi-arch")

	var imageDigest string

	if isMultiArch {
		logger.Info("Multi-architecture deployment requested")
		logger.Info("Invoking release.sh script for multi-arch deployment...")

		// Invoke release.sh script from AVS template
		// The script will handle building other architectures and creating Image Index
		imageDigest, err = invokeReleaseScript(cCtx.Context, logger, avs, registryUrl, version, localImageId)
		if err != nil {
			return fmt.Errorf("multi-arch release failed: %w", err)
		}
	} else {
		logger.Info("Single-architecture deployment")
		logger.Info("Pushing local container directly...")

		// Push the single local container directly
		imageDigest, err = pushSingleContainer(cCtx.Context, logger, avs, registryUrl, version, localImageId)
		if err != nil {
			return fmt.Errorf("single-arch release failed: %w", err)
		}
	}

	logger.Info("Container deployment completed!")
	logger.Info("Image digest: %s", imageDigest)

	// TODO: Bundle with Aggregator and Executor releases and push to ReleaseManager
	logger.Info("Bundling with Aggregator and Executor releases...")
	logger.Info("Publishing to ReleaseManager contract...")
	logger.Info("ReleaseManager integration not yet implemented")

	return nil
}

func invokeReleaseScript(ctx context.Context, logger iface.Logger, avs, registryUrl, version, localImageId string) (string, error) {
	// Sanitize AVS address for Docker repository naming
	sanitizedAvs := strings.ToLower(avs)
	sanitizedAvs = strings.TrimPrefix(sanitizedAvs, "0x")

	imageName := fmt.Sprintf("hourglass-performer-%s", sanitizedAvs)

	// Look for release.sh script in AVS template
	releaseScript := filepath.Join(".hourglass", "scripts", "release.sh")
	if _, err := os.Stat(releaseScript); err != nil {
		return "", fmt.Errorf("release.sh script not found at %s. Please ensure it exists in your AVS template", releaseScript)
	}

	logger.Info("Executing release script: %s", releaseScript)
	logger.Info("   Registry: %s", registryUrl)
	logger.Info("   Image: %s", imageName)
	logger.Info("   Version: %s", version)

	// Execute release.sh script
	// The release script will handle multi-arch building and create the Image Index
	cmd := exec.CommandContext(ctx, "bash", releaseScript, registryUrl, imageName, version, localImageId)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("Release script failed: %s", string(output))
		return "", fmt.Errorf("release script execution failed: %w", err)
	}

	logger.Info("Release script output: %s", string(output))

	// Read the digest from release script output
	digestFile := "/tmp/release_digest"
	if _, err := os.Stat(digestFile); err != nil {
		return "", fmt.Errorf("release script did not create digest file at %s", digestFile)
	}

	digestData, err := os.ReadFile(digestFile)
	if err != nil {
		return "", fmt.Errorf("failed to read digest file: %w", err)
	}

	// Parse digest from file (format: IMAGE_INDEX_DIGEST=sha256:...)
	lines := strings.Split(string(digestData), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "IMAGE_INDEX_DIGEST=") {
			digest := strings.TrimPrefix(line, "IMAGE_INDEX_DIGEST=")
			return strings.TrimSpace(digest), nil
		}
	}

	return "", fmt.Errorf("IMAGE_INDEX_DIGEST not found in release script output")
}

func pushSingleContainer(ctx context.Context, logger iface.Logger, avs, registryUrl, version, imageId string) (string, error) {
	// Sanitize AVS address for Docker repository naming
	sanitizedAvs := strings.ToLower(avs)
	sanitizedAvs = strings.TrimPrefix(sanitizedAvs, "0x")

	imageName := fmt.Sprintf("hourglass-performer-%s", sanitizedAvs)
	fullImageName := fmt.Sprintf("%s/%s:%s", registryUrl, imageName, version)

	logger.Info("Tagging image for registry...")
	logger.Info("   Source: %s", imageId)
	logger.Info("   Target: %s", fullImageName)

	// Tag the image
	tagCmd := exec.CommandContext(ctx, "docker", "tag", imageId, fullImageName)
	if err := tagCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to tag image: %w", err)
	}

	// Push the image
	logger.Info("Pushing to registry...")
	pushCmd := exec.CommandContext(ctx, "docker", "push", fullImageName)
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		logger.Error("Push failed: %s", string(pushOutput))
		return "", fmt.Errorf("failed to push image: %w", err)
	}

	logger.Info("Push output: %s", string(pushOutput))

	// Get the image digest
	logger.Info("Getting image digest...")
	inspectCmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{index .RepoDigests 0}}", fullImageName)
	digestOutput, err := inspectCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get image digest: %w", err)
	}

	// Parse digest from output (format: registry/image@sha256:...)
	repoDigest := strings.TrimSpace(string(digestOutput))
	if strings.Contains(repoDigest, "@") {
		parts := strings.Split(repoDigest, "@")
		if len(parts) == 2 {
			return parts[1], nil
		}
	}

	return "", fmt.Errorf("could not parse digest from: %s", repoDigest)
}
