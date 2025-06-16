package template_test

import (
	"strings"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
)

type mockTracker struct {
	// Percentage by module
	perc   map[string]int
	label  map[string]string
	clears int
}

func newMockTracker() *mockTracker {
	return &mockTracker{
		perc:  make(map[string]int),
		label: make(map[string]string),
	}
}

// Set is called by ProgressLogger.SetProgress
func (f *mockTracker) Set(id string, pct int, displayText string) {
	// record only the max pct seen
	if old, ok := f.perc[id]; !ok || pct > old {
		f.perc[id] = pct
		f.label[id] = displayText
	}
}

// Render is no-op here
func (f *mockTracker) Render() {}

// Clear is called when progress is cleared
func (f *mockTracker) Clear() {
	f.clears++
}

// ProgressRows is a no-op here
func (s *mockTracker) ProgressRows() []iface.ProgressRow { return make([]iface.ProgressRow, 0) }

func TestCloneReporterEndToEnd(t *testing.T) {
	log := logger.NewNoopLogger()
	mock := newMockTracker()
	progLogger := *logger.NewProgressLogger(log, mock)

	// Create the reporter for a repo named "foo"
	rep := template.NewCloneReporter("https://example.com/foo.git", progLogger, nil)

	// Simulate events
	events := []template.CloneEvent{
		{Type: template.EventProgress, Module: "foo", Progress: 100, Ref: "main"},
		{Type: template.EventSubmoduleDiscovered, Parent: ".", Name: "modA", URL: "uA", Ref: "main"},
		{Type: template.EventSubmoduleCloneStart, Parent: ".", Module: "modA", Ref: "main"},
		{Type: template.EventProgress, Parent: ".", Module: "modA", Progress: 50, Ref: "main"},
		{Type: template.EventProgress, Parent: ".", Module: "modA", Progress: 75, Ref: "main"},
		{Type: template.EventCloneComplete},
	}
	for _, ev := range events {
		rep.Report(ev)
	}

	// After completion, we expect:
	// - For repo root "foo": final percentage 100
	// - For modA: final percentage 100
	if pct, ok := mock.perc["modA"]; !ok || pct != 100 {
		t.Errorf("modA expected 100%%, got %d%%", pct)
	}
	if pct, ok := mock.perc["foo"]; !ok || pct != 100 {
		t.Errorf("foo expected 100%%, got %d%%", pct)
	}

	// We also expect that the displayText for foo contains the ref
	if lbl, ok := mock.label["foo"]; !ok || !strings.Contains(lbl, "Cloning from ref: main") {
		t.Errorf("foo label expected to mention ref, got %q", lbl)
	}

	// And that Clear was called at least once (at end)
	if mock.clears == 0 {
		t.Error("expected at least one Clear() call")
	}
}

func TestCloneReporterSubmoduleDiscoveryGrouping(t *testing.T) {
	log := logger.NewNoopLogger()
	mock := newMockTracker()
	progLogger := *logger.NewProgressLogger(log, mock)

	rep := template.NewCloneReporter("https://example.com/bar.git", progLogger, nil)

	// Two discoveries under same parent, then start
	rep.Report(template.CloneEvent{Type: template.EventSubmoduleDiscovered, Parent: "p1/", Name: "a", URL: "uA"})
	rep.Report(template.CloneEvent{Type: template.EventSubmoduleDiscovered, Parent: "p1/", Name: "b", URL: "uB"})
	// Now trigger the clone start: should flush both
	rep.Report(template.CloneEvent{Type: template.EventSubmoduleCloneStart, Parent: "p1/", Module: "a"})

	// That flush should have called Clear() once
	if mock.clears != 1 {
		t.Errorf("expected Clear after submodule flush, got %d", mock.clears)
	}
}

func TestCloneReporterFallbackRootProgress(t *testing.T) {
	log := logger.NewNoopLogger()
	mock := newMockTracker()
	progLogger := *logger.NewProgressLogger(log, mock)

	rep := template.NewCloneReporter("https://example.com/baz.git", progLogger, nil)

	// Emit a Progress event with Module=""
	rep.Report(template.CloneEvent{Type: template.EventProgress, Module: "", Progress: 33, Ref: "dev"})

	// We should have seen update to 33%
	if pct, ok := mock.perc["baz"]; !ok || pct != 33 {
		t.Errorf("baz expected 33%%, got %d%%", pct)
	}
	if lbl, ok := mock.label["baz"]; !ok || !strings.Contains(lbl, "Cloning from ref: dev") {
		t.Errorf("baz label should show ref, got %q", lbl)
	}
}
