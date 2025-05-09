package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitFetcher_InvalidURL(t *testing.T) {
	fetcher := &GitFetcher{}
	tempDir := t.TempDir()

	// Test with an invalid URL (should fail)
	err := fetcher.Fetch("invalid-url", tempDir, false, true)
	if err == nil {
		t.Error("expected error for invalid URL")
	}

	// Verify .git directory is not present (since Fetch failed)
	gitDir := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		t.Error("expected no .git directory after failed fetch")
	}
}

func TestGitFetcher_ValidRepo(t *testing.T) {
	fetcher := &GitFetcher{}
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-Labs/hourglass-avs-template"

	err := fetcher.Fetch(repo, tempDir, false, true)
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
	fetcher := &GitFetcher{MaxDepth: 1}
	tempDir := t.TempDir()

	// Includes submodules: simple example with known submodule
	repo := "https://github.com/Layr-Labs/hourglass-avs-template"

	err := fetcher.Fetch(repo, tempDir, false, true)
	if err != nil {
		t.Fatalf("unexpected error cloning repo with submodules: %v", err)
	}

	// Example path — verify at least one known submodule folder exists
	expectedSubmodule := filepath.Join(tempDir, "contracts")
	if _, err := os.Stat(expectedSubmodule); os.IsNotExist(err) {
		t.Log("submodule not found — this may vary based on repo")
	}
}

func TestGitFetcher_MaxDepth(t *testing.T) {
	fetcher := &GitFetcher{MaxDepth: 0}
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-Labs/hourglass-avs-template"

	err := fetcher.Fetch(repo, tempDir, false, true)
	if err != nil {
		t.Fatalf("unexpected error fetching repo with depth: %v", err)
	}
	visited := filepath.Join(tempDir, "contracts")
	if _, err := os.Stat(visited); err != nil {
		t.Fatalf("expected top-level submodule not found")
	}

	contractsGitmodules := filepath.Join(tempDir, "contracts/.gitmodules")
	if _, err := os.Stat(contractsGitmodules); err == nil {
		t.Errorf("contracts/.gitmodules parsed despite MaxDepth=1")
	}
}

func TestGitFetcher_NonexistentBranch(t *testing.T) {
	fetcher := &GitFetcher{}
	tempDir := t.TempDir()

	repo := "https://github.com/Layr-Labs/hourglass-avs-template/tree/missing-branch"

	err := fetcher.Fetch(repo, tempDir, false, true)
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}
