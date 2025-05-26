package template_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/template"
)

// RunnerFunc is an adapter to let us define Runner inline
type RunnerFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

func (f RunnerFunc) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return f(ctx, name, args...)
}

// errReader always returns an error on Read
type errReader struct{}

func (errReader) Read(_ []byte) (int, error) { return 0, errors.New("read failed") }

// In-memory reporter for assertions
type recordingReporter struct {
	events []template.CloneEvent
}

func (r *recordingReporter) Report(ev template.CloneEvent) {
	r.events = append(r.events, ev)
}

// makeClientWithOutput gives us a real GitClient with default regexps and a mock Runner
func makeClientWithOutput(output string) *template.GitClient {
	cli := &template.GitClient{
		// stub Runner: any CommandContext returns a Cmd that just writes `output` to stderr
		Runner: RunnerFunc(func(_ context.Context, name string, args ...string) *exec.Cmd {
			script := fmt.Sprintf(">&2 printf %q; exit 0\n", output)
			return exec.Command("bash", "-c", script)
		}),
		ReceivingRegex: regexp.MustCompile(`Receiving objects:\s+(\d+)%`),
		CloningRegex:   regexp.MustCompile(`Cloning into ['"]?(.+?)['"]?\.{3}`),
		SubmoduleRegex: regexp.MustCompile(
			`^Submodule ['"]?([^'"]+)['"]? \(([^)]+)\) registered for path ['"]?(.+?)['"]?$`,
		),
	}
	return cli
}

func TestParseCloneOutput_AllEvents(t *testing.T) {
	stub := strings.Join([]string{
		// top-level progress
		`Receiving objects:  10% (5/50)`,
		// submodule discovery
		`Submodule 'bar' (https://example.com/bar.git) registered for path 'lib/bar'`,
		// enter submodule
		`Cloning into '/tmp/foo/lib/bar'...`,
		`Receiving objects: 100% (25/25)`,
	}, "\n")

	rep := &recordingReporter{}
	client := makeClientWithOutput(stub)
	err := client.ParseCloneOutput(bytes.NewBufferString(stub), rep, "/tmp/foo", "myref")
	if err != nil {
		t.Fatal(err)
	}

	// We expect exactly 5 events in sequence:
	// - top-level progress 10%
	// - submodule discovered
	// - submodule clone start
	// - submodule progress 0%
	// - submodule progress 100%
	if len(rep.events) != 5 {
		t.Fatalf("got %d events; want 5", len(rep.events))
	}

	var (
		foundTop10      bool
		foundDiscovery  bool
		foundCloneStart bool
		foundSubm0      bool
		foundSubm100    bool
	)
	for _, ev := range rep.events {
		switch {
		case ev.Type == template.EventProgress && ev.Progress == 10 && !foundTop10:
			foundTop10 = true

		case ev.Type == template.EventSubmoduleDiscovered && foundTop10 && !foundDiscovery:
			foundDiscovery = true

		case ev.Type == template.EventSubmoduleCloneStart && foundDiscovery && !foundCloneStart:
			wantMod := "bar"
			if ev.Module != wantMod {
				t.Errorf("clone start module = %q; want %q", ev.Module, wantMod)
			}
			foundCloneStart = true

		case ev.Type == template.EventProgress && ev.Progress == 0 && foundCloneStart && !foundSubm0:
			foundSubm0 = true

		case ev.Type == template.EventProgress && ev.Progress == 100 && foundSubm0 && !foundSubm100:
			foundSubm100 = true
		}
	}

	if !foundTop10 {
		t.Error("did not see top-level 10% progress")
	}
	if !foundDiscovery {
		t.Error("did not see submodule discovery")
	}
	if !foundCloneStart {
		t.Error("did not see submodule clone start")
	}
	if !foundSubm0 {
		t.Error("did not see submodule 0% progress")
	}
	if !foundSubm100 {
		t.Error("did not see submodule 100% progress")
	}
}

func TestParseCloneOutput_TrimPath(t *testing.T) {
	stub := `Cloning into '/home/user/proj/lib/submod'...`
	rep := &recordingReporter{}
	client := makeClientWithOutput(stub)
	err := client.ParseCloneOutput(strings.NewReader(stub+"\n"), rep, "/home/user/proj", "r")
	if err != nil {
		t.Fatal(err)
	}
	// Should have one SubmoduleCloneStart event and progress set to 0
	if len(rep.events) != 2 {
		t.Fatalf("got %d events; want 2", len(rep.events))
	}
	if ev := rep.events[0]; ev.Type != template.EventSubmoduleCloneStart || ev.Module != filepath.Join("lib", "submod") {
		t.Errorf("got module %q; want %q", ev.Module, filepath.Join("lib", "submod"))
	}
	if ev := rep.events[1]; ev.Type != template.EventProgress && ev.Progress == 0 {
		t.Errorf("progress report was not initiated")
	}
}

func TestParseCloneOutput_ScanError(t *testing.T) {
	// Simulate a reader that fails mid-scan
	reader := &errReader{}
	rep := &recordingReporter{}
	client := makeClientWithOutput("")
	err := client.ParseCloneOutput(reader, rep, "/tmp/foo", "r")
	if err == nil || !strings.Contains(err.Error(), "scan stderr") {
		t.Fatalf("expected scan stderr error, got %v", err)
	}
	if len(rep.events) != 0 {
		t.Error("expected no events on scan error")
	}
}

func TestClone_WithMockProgress(t *testing.T) {
	// Make a Runner that prints two progress lines then succeeds
	mock := RunnerFunc(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		script := `
		  >&2 echo "Cloning into 'dest'..."
		  >&2 echo "Receiving objects:  42%"
		  >&2 echo "Receiving objects: 100%"
		  exit 0
		`
		return exec.CommandContext(ctx, "bash", "-c", script)
	})
	client := template.NewGitClientWithRunner(mock)
	rep := &recordingReporter{}
	err := client.Clone(context.Background(),
		"https://example.com/foo.git", "main", "/tmp/foo",
		template.GitFetcherConfig{Verbose: false}, rep,
	)
	if err != nil {
		t.Fatalf("unexpected clone error: %v", err)
	}
	// Ensure we saw progress event at 42% and final 100% for top-level
	found42, found100 := false, false
	for _, e := range rep.events {
		if e.Type == template.EventProgress && e.Progress == 42 {
			found42 = true
		}
		if e.Type == template.EventProgress && e.Progress == 100 && e.Module == "foo" {
			found100 = true
		}
	}
	if !found42 || !found100 {
		t.Errorf("progress events missing; saw %+v", rep.events)
	}
}
