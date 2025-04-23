package template

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Fetcher interface {
	Fetch(templateURL, targetDir string) error
}

type GitFetcher struct{}

func parseGitHubURL(url string) (repoURL, branch string) {
	// Handle GitHub tree URLs (e.g., https://github.com/user/repo/tree/branch)
	if strings.Contains(url, "/tree/") {
		parts := strings.Split(url, "/tree/")
		if len(parts) == 2 {
			repoURL = parts[0] + ".git"
			branch = parts[1]
			return
		}
	}
	// Handle regular Git URLs
	return url, ""
}

func (g *GitFetcher) Fetch(templateURL, targetDir string) error {
	repoURL, branch := parseGitHubURL(templateURL)

	args := []string{"clone", "--depth=1"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repoURL, targetDir)

	cmd := exec.Command("git", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone template: %w", err)
	}

	// Cleanup .git directory
	gitDir := filepath.Join(targetDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove .git directory: %w", err)
	}

	return nil
}
