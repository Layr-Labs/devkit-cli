package template_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
)

// mockRunnerSuccess always returns a Cmd that exits 0
type mockRunnerSuccess struct{}

func (mockRunnerSuccess) CommandContext(_ context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.Command("true")
}

// mockRunnerFail always returns a Cmd that exits 1
type mockRunnerFail struct{}

func (mockRunnerFail) CommandContext(_ context.Context, _ string, _ ...string) *exec.Cmd {
	return exec.Command("false")
}

// mockRunnerProgress emits "git clone"-style progress on stderr
type mockRunnerProgress struct{}

func (mockRunnerProgress) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	// This shell script writes two progress lines then exits 0
	script := `
      >&2 echo "Cloning into 'dest'..."
      >&2 echo "Receiving objects:  50%"
      sleep 0.01
      >&2 echo "Receiving objects: 100%"
      exit 0
    `
	return exec.CommandContext(ctx, "bash", "-c", script)
}

// spyTrackerDedup records only the latest Set() per module.
type spyTrackerDedup struct {
	mu    sync.Mutex
	order []string
	byID  map[string]struct {
		Pct   int
		Label string
	}
}

func newSpyTrackerDedup() *spyTrackerDedup {
	return &spyTrackerDedup{
		order: make([]string, 0),
		byID: make(map[string]struct {
			Pct   int
			Label string
		}),
	}
}

func (s *spyTrackerDedup) Set(id string, pct int, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, seen := s.byID[id]; !seen {
		s.order = append(s.order, id)
	}
	s.byID[id] = struct {
		Pct   int
		Label string
	}{pct, label}
}

func (s *spyTrackerDedup) Render() {}

func (s *spyTrackerDedup) Clear() {}

func (s *spyTrackerDedup) ProgressRows() []iface.ProgressRow { return make([]iface.ProgressRow, 0) }

// getFetcherWithRunner returns a GitFetcher and its underlying LogProgressTracker.
func getFetcherWithRunner(r template.Runner) (*template.GitFetcher, *spyTrackerDedup) {
	client := template.NewGitClientWithRunner(r)
	log := logger.NewNoopLogger()

	// Inject our spyTracker instead of the real one:
	spy := newSpyTrackerDedup()
	progressLogger := logger.NewProgressLogger(log, spy)

	return &template.GitFetcher{
		Client: client,
		Logger: *progressLogger,
		Config: template.GitFetcherConfig{Verbose: false},
	}, spy
}

func TestFetchSucceedsWithMockRunner(t *testing.T) {
	f, _ := getFetcherWithRunner(mockRunnerSuccess{})
	dir := t.TempDir()
	if err := f.Fetch(context.Background(), "any-url", "any-ref", dir); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestFetchFailsWhenCloneFails(t *testing.T) {
	f, _ := getFetcherWithRunner(mockRunnerFail{})
	dir := t.TempDir()
	err := f.Fetch(context.Background(), "any-url", "any-ref", dir)
	if err == nil {
		t.Fatal("expected error when git clone fails")
	}
}

func TestReporterGetsProgressEvents(t *testing.T) {
	// Build a client that emits 50% then 100% on stderr
	f, tracker := getFetcherWithRunner(mockRunnerProgress{})
	dir := t.TempDir()
	if err := f.Fetch(context.Background(), "irrelevant", "irrelevant", dir); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	// Inspect the tracker: it only logs on 100%
	rows := tracker.order

	// Expectations after successful run
	if len(rows) != 2 {
		t.Fatalf("expected 2 progress row, got %d", len(rows))
	}
	if tracker.byID[rows[0]].Pct != 100 {
		t.Errorf("expected the 100%% event, got %+v", rows[0])
	}
	if tracker.byID[rows[1]].Pct != 100 {
		t.Errorf("expected the 100%% event, got %+v", rows[0])
	}
}

// TestCloneRealRepo integration test
func TestCloneRealRepo(t *testing.T) {
	dir := t.TempDir()
	cfg, err := template.LoadConfig()
	if err != nil {
		t.Fatalf("failed to load templates cfg: %v", err)
	}

	// Use the default task.go template
	mainBaseURL, mainVersion, err := template.GetTemplateURLs(cfg, "task", "go")
	if err != nil {
		t.Fatalf("GetTemplateURLs failed: %v", err)
	}

	// Use real runner
	client := template.NewGitClient()
	log := logger.NewNoopLogger()
	tracker := progress.NewLogProgressTracker(100, log)
	prog := logger.NewProgressLogger(log, tracker)

	// Set-up gitFetcher with real client
	f := &template.GitFetcher{
		Client: client,
		Config: template.GitFetcherConfig{Verbose: false},
		Logger: *prog,
	}

	// Attempt a real clone
	if err := f.Fetch(context.Background(), mainBaseURL, mainVersion, dir); err != nil {
		t.Fatalf("real git clone failed: %v", err)
	}

	// Verify .git exists
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Errorf(".git directory not found after clone: %v", err)
	}

	// Verify at least one known submodule folder exists
	expectedSubmodule := filepath.Join(dir, ".devkit", "contracts")
	if _, err := os.Stat(expectedSubmodule); os.IsNotExist(err) {
		t.Log("submodule not found - has .devkit/contracts moved?")
	}

	// Verify we saw at least one 100% record
	rows := tracker.ProgressRows()
	if len(rows) > 0 {
		t.Error("expected at least one completed progress row, got none")
	}

}
