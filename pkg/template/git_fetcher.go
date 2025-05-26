package template

import (
	"context"
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
)

// GitFetcherConfig holds options; we only care about Verbose here
type GitFetcherConfig struct {
	Verbose bool
}

// TODO: implement metric transport
type GitMetrics interface {
	CloneStarted(repo string)
	CloneFinished(repo string, err error)
}

// GitFetcher wraps clone with metrics and reporting
type GitFetcher struct {
	Client  *GitClient
	Metrics GitMetrics
	Config  GitFetcherConfig
	Logger  logger.ProgressLogger
}

func (f *GitFetcher) Fetch(ctx context.Context, repoURL, ref, targetDir string) error {
	if repoURL == "" {
		return fmt.Errorf("repoURL is required")
	}

	// Print job initiation
	f.Logger.Info("\nCloning repo: %s → %s\n\n", repoURL, targetDir)

	// Report to metrics
	if f.Metrics != nil {
		f.Metrics.CloneStarted(repoURL)
	}

	// Build a reporter that knows how to drive our ProgressLogger
	var reporter Reporter
	if !f.Config.Verbose {
		reporter = NewCloneReporter(repoURL, f.Logger, f.Metrics)
	}

	// Initiate clone
	err := f.Client.Clone(ctx, repoURL, ref, targetDir, f.Config, reporter)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// Print job completion
	f.Logger.Info("Clone repo complete: %s\n\n", repoURL)
	return nil
}
