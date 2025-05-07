package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitFetcher(t *testing.T) {
	fetcher := &GitFetcher{}
	tempDir := t.TempDir()

	// Test with an invalid URL (should fail)
	err := fetcher.Fetch("invalid-url", tempDir, false)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	// Verify .git directory is not present (since Fetch failed)
	gitDir := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
		t.Error("Expected no .git directory after failed fetch")
	}
}
