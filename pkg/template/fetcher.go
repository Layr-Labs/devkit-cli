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

func (g *GitFetcher) Fetch(templateURL, targetDir string, verbose bool) error {
	repoURL, branch := parseGitHubURL(templateURL)

	args := []string{"clone", "--recurse-submodules", "-j8", "--depth=1"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repoURL, targetDir)
	cmd := exec.Command("git", args...)
	cmd.Dir = targetDir

	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to clone template: %w", err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to clone template: %w\nOutput: %s", err, string(output))
		}
	}

	// Cleanup .git directory
	// TODO: In future, we might want to do git init after this step based on a flag?
	gitDir := filepath.Join(targetDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove .git directory: %w", err)
	}

	return nil
}
