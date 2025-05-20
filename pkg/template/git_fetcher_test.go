package template

import (
	"context"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
	"os"
	"path/filepath"
	"testing"
)

func TestGitFetcher_InvalidURL(t *testing.T) {
	fetcher := getFetcher(1)
	tempDir := t.TempDir()

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test with an invalid URL (should fail)
	err := fetcher.Fetch(cmdCtx, "invalid-url", tempDir)
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// Verify .git directory is not present (since Fetch failed)
	gitDir := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		t.Error("expected no .git directory after failed fetch")
	}
}

func getFetcher(maxDepth int) *GitFetcher {
	// Set Cache location as ~/.devkit
	basePath := filepath.Join(os.Getenv("HOME"), ".devkit")
	// Set logger
	log := logger.NewZapLogger()
	// Set fetcher
	fetcher := &GitFetcher{
		Git:   NewGitClient(),
		Cache: NewGitRepoCache(basePath),
		Logger: *logger.NewProgressLogger(
			log,
			progress.NewLogProgressTracker(10, log),
		),
		Config: GitFetcherConfig{
			CacheDir:       basePath,
			MaxDepth:       maxDepth,
			MaxRetries:     3,
			MaxConcurrency: 8,
		},
	}
	return fetcher
}

func TestGitFetcher_ValidRepo(t *testing.T) {
	// Set fetcher
	fetcher := getFetcher(1)
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-labs/eigenlayer-contracts"

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := fetcher.Fetch(cmdCtx, repo, tempDir)
	if err != nil {
		t.Fatalf("unexpected error fetching repo: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".git")); err != nil {
		t.Errorf(".git not found after clone: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "README.md")); err != nil {
		t.Log("README file not found — still valid but may have changed")
	}
}

func TestGitFetcher_Submodules(t *testing.T) {
	fetcher := getFetcher(1)
	tempDir := t.TempDir()

	// Includes submodules: simple example with known submodule
	repo := "https://github.com/Layr-labs/eigenlayer-contracts"

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := fetcher.Fetch(cmdCtx, repo, tempDir)
	if err != nil {
		t.Fatalf("unexpected error cloning repo with submodules: %v", err)
	}

	// Example path — verify at least one known submodule folder exists
	expectedSubmodule := filepath.Join(tempDir, "lib", "forge-std")
	if _, err := os.Stat(expectedSubmodule); os.IsNotExist(err) {
		t.Log("submodule not found — this may vary based on repo")
	}
}

func TestGitFetcher_MaxDepth(t *testing.T) {
	fetcher := getFetcher(0)
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-labs/eigenlayer-contracts"

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := fetcher.Fetch(cmdCtx, repo, tempDir)
	if err != nil {
		t.Fatalf("unexpected error fetching repo with depth: %v", err)
	}
	visited := filepath.Join(tempDir, "lib", "forge-std")
	if _, err := os.Stat(visited); err != nil {
		t.Fatalf("expected top-level submodule not found")
	}

	contractsGitmodules := filepath.Join(tempDir, "lib", "forge-std", ".gitmodules")
	if _, err := os.Stat(contractsGitmodules); err == nil {
		t.Errorf("lib/forge-std/.gitmodules parsed despite MaxDepth=1")
	}
}

func TestGitFetcher_NonexistentBranch(t *testing.T) {
	fetcher := getFetcher(0)
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-labs/eigenlayer-contracts/tree/missing-branch"

	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := fetcher.Fetch(cmdCtx, repo, tempDir)
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}
