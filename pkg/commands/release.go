package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
			Usage:     "Publish a new AVS release to Docker Hub",
			ArgsUsage: "<avs> <operatorSetId> <digest> <version> <deadlineTimestamp>",
			Flags: append([]cli.Flag{
				&cli.BoolFlag{
					Name:  "dry-run",
					Usage: "Show what would be published without executing the operation",
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
		return fmt.Errorf("expected 5 arguments: <avs> <operatorSetId> <digest> <version> <deadlineTimestamp>")
	}

	// Parse arguments
	avs := cCtx.Args().Get(0)
	operatorSetIdStr := cCtx.Args().Get(1)
	digest := cCtx.Args().Get(2)
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

	// Validate digest format (should be sha256 hash)
	if len(digest) != 64 {
		return fmt.Errorf("digest must be a 64-character SHA256 hash")
	}

	// Validate deadline is in the future
	if deadlineTimestamp <= time.Now().Unix() {
		return fmt.Errorf("deadline timestamp must be in the future")
	}

	logger.Info("Publishing AVS release to Docker Hub...")
	logger.Info("  AVS address: %s", avs)
	logger.Info("  Operator Set ID: %d", operatorSetId)
	logger.Info("  Digest: %s", digest)
	logger.Info("  Version: %s", version)
	logger.Info("  Deadline: %s", time.Unix(deadlineTimestamp, 0).Format(time.RFC3339))

	if cCtx.Bool("dry-run") {
		logger.Info("🔍 Dry run mode - would publish release with above parameters")
		return nil
	}

	// Publish to Docker Hub
	return publishToDockerHub(cCtx.Context, logger, avs, digest)
}

func publishToDockerHub(ctx context.Context, logger iface.Logger, avsId string, digest string) error {
	// Check if crane is available
	if _, err := exec.LookPath("crane"); err != nil {
		return fmt.Errorf("crane CLI tool not found. Please install it from: https://github.com/google/go-containerregistry/blob/main/cmd/crane/README.md")
	}

	// Get registry URL from context configuration
	cfg, err := common.LoadConfigWithContextConfig("devnet") // TODO: make context configurable
	if err != nil {
		return fmt.Errorf("failed to load context config: %w", err)
	}

	var registryURL string
	if cfg.Context["devnet"].Artifacts != nil {
		registryURL = cfg.Context["devnet"].Artifacts.RegistryUrl
	}

	if registryURL == "" {
		return fmt.Errorf("registry_url is not set in context configuration. Please set it in your context artifacts section")
	}

	// Sanitize AVS ID for Docker repository naming (lowercase, remove 0x prefix)
	sanitizedAvsId := strings.ToLower(avsId)
	sanitizedAvsId = strings.TrimPrefix(sanitizedAvsId, "0x")

	// Create repository name using registry URL
	repoName := fmt.Sprintf("%s/hourglass-performer-%s", registryURL, sanitizedAvsId)
	// Use the digest as the tag instead of version
	tag := digest
	fullRef := fmt.Sprintf("%s:%s", repoName, tag)

	logger.Info("🚀 Publishing to registry: %s", fullRef)

	// Check if the release manifest OCI layout exists
	releaseManifestDir := "./release-manifest"
	if _, err := os.Stat(releaseManifestDir); err != nil {
		logger.Error("Release Manifest OCI layout not found at: %s", releaseManifestDir)
		logger.Error("Run 'devkit avs build' first to create the Release Manifest")
		return fmt.Errorf("release Manifest OCI layout not found - run 'devkit avs build' first")
	}

	logger.Info("📦 Found Release Manifest OCI layout")
	logger.Info("📤 Pushing to registry url: %s", registryURL)

	// Push the Release Manifest OCI layout to Docker Hub
	pushCmd := exec.CommandContext(ctx, "crane", "push", releaseManifestDir, fullRef)
	pushOutput, err := pushCmd.CombinedOutput()
	if err != nil {
		logger.Error("Failed to push: %s", string(pushOutput))
		return fmt.Errorf("failed to push to Docker Hub: %w", err)
	}

	logger.Info("Successfully published to Docker Hub!")
	logger.Info("Published to: %s", fullRef)
	logger.Info("Push output: %s", string(pushOutput))

	// Verify the push
	logger.Info("Verifying published manifest...")
	verifyCmd := exec.CommandContext(ctx, "crane", "manifest", fullRef)
	if verifyOutput, verifyErr := verifyCmd.Output(); verifyErr == nil {
		logger.Info("✅ Verification successful - manifest is accessible")
		logger.Debug("Published manifest: %s", string(verifyOutput))
	} else {
		logger.Warn(" Could not verify published manifest: %v", verifyErr)
	}

	return nil
}
