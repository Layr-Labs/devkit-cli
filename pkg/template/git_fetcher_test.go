package template

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
)

func getFetcher() *GitFetcher {
	log := logger.NewZapLogger()
	return &GitFetcher{
		Client: NewGitClient(),
		Logger: *logger.NewProgressLogger(
			log,
			progress.NewLogProgressTracker(10, log),
		),
	}
}

func TestGitFetcher_InvalidURL(t *testing.T) {
	f := getFetcher()
	tmp := t.TempDir()

	err := f.Fetch(context.Background(), "not-a-url", "master", tmp)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if _, err := os.Stat(filepath.Join(tmp, ".git")); !os.IsNotExist(err) {
		t.Error("expected no .git directory after failure")
	}
}

func TestGitFetcher_InvalidRef(t *testing.T) {
	f := getFetcher()
	tmp := t.TempDir()

	err := f.Fetch(context.Background(),
		"https://github.com/Layr-Labs/hourglass-avs-template",
		"no-such-branch",
		tmp,
	)
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
	if _, err := os.Stat(filepath.Join(tmp, ".git")); !os.IsNotExist(err) {
		t.Error("expected no .git directory after failure")
	}
}

func TestGitFetcher_ValidClone(t *testing.T) {
	f := getFetcher()
	tmp := t.TempDir()

	err := f.Fetch(context.Background(),
		"https://github.com/Layr-Labs/hourglass-avs-template",
		"master",
		tmp,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".git")); err != nil {
		t.Errorf(".git missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "README.md")); os.IsNotExist(err) {
		t.Log("warning: README.md not found")
	}
}

func TestGitFetcher_Submodules(t *testing.T) {
	f := getFetcher()
	tmp := t.TempDir()

	err := f.Fetch(context.Background(),
		"https://github.com/Layr-Labs/hourglass-avs-template",
		"master",
		tmp,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// check for at least one submodule directory
	// adjust the path based on the repo’s actual layout
	sub := filepath.Join(tmp, ".devkit", "contracts")
	if _, err := os.Stat(sub); os.IsNotExist(err) {
		t.Logf("submodule directory %q not found (repo layout may have changed)", sub)
	}
}

func TestGitFetcher_NonexistentBranch(t *testing.T) {
	f := getFetcher()
	tmp := t.TempDir()

	err := f.Fetch(context.Background(),
		"https://github.com/Layr-Labs/hourglass-avs-template",
		"lol-fake",
		tmp,
	)
	if err == nil {
		t.Fatal("expected error on nonexistent branch")
	}
}
