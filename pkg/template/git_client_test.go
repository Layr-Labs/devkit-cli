package template

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testReporter collects events for inspection.
type testReporter struct {
	events []CloneEvent
}

func (r *testReporter) Report(ev CloneEvent) {
	r.events = append(r.events, ev)
}

func makeClient() *GitClient {
	return &GitClient{
		receivingRegex: regexp.MustCompile(`Receiving objects:\s+(\d+)%`),
		cloningRegex:   regexp.MustCompile(`Cloning into ['"]?(.+?)['"]?\.{3}`),
		submoduleRegex: regexp.MustCompile(
			`^Submodule ['"]?([^'"]+)['"]? \(([^)]+)\) registered for path ['"]?(.+?)['"]?$`,
		),
	}
}

func TestParseCloneOutput(t *testing.T) {
	stub := strings.Join([]string{
		// Initial top-level clone
		`Cloning into '/tmp/foo'...`,
		`Receiving objects:   10% (5/50)`,
		// Submodule discovery
		`Submodule 'bar' (https://example.com/bar.git) registered for path 'lib/bar'`,
		// Entering submodule
		`Cloning into '/tmp/foo/lib/bar'...`,
		`Receiving objects:   50% (25/50)`,
	}, "\n")

	var rep testReporter
	client := makeClient()
	err := client.parseCloneOutput(bytes.NewBufferString(stub), &rep, "/tmp/foo", "myref")
	assert.NoError(t, err)

	// We expect at least these three types of events in order:
	// - EventProgress for top-level (10%)
	// - EventSubmoduleDiscovered
	// - EventSubmoduleCloneStart for 'bar'
	// - EventProgress for 'bar' at 50%
	found := map[CloneEventType]bool{}
	for _, ev := range rep.events {
		found[ev.Type] = true
	}

	for _, want := range []CloneEventType{
		EventProgress,
		EventSubmoduleDiscovered,
		EventSubmoduleCloneStart,
	} {
		if !found[want] {
			t.Errorf("missing event type %v", want)
		}
	}

	// Check that the discovered URL and parent are correct
	var foundDiscovery bool
	for _, ev := range rep.events {
		if ev.Type == EventSubmoduleDiscovered {
			if ev.URL != "https://example.com/bar.git" {
				t.Errorf("got URL %q; want %q", ev.URL, "https://example.com/bar.git")
			}
			if ev.Parent != "lib/" {
				t.Errorf("got Parent %q; want %q", ev.Parent, "lib/")
			}
			foundDiscovery = true
		}
	}
	if !foundDiscovery {
		t.Fatal("did not find submodule discovery event")
	}
}

func TestParseCloneOutput_TrimPath(t *testing.T) {
	// ensure that filepath.Rel works as intended
	stub := `Cloning into '/home/user/proj/lib/submod'...`
	var rep testReporter
	client := makeClient()
	err := client.parseCloneOutput(strings.NewReader(stub+"\n"), &rep, "/home/user/proj", "r")
	assert.NoError(t, err)

	var start CloneEvent
	for _, ev := range rep.events {
		if ev.Type == EventSubmoduleCloneStart {
			start = ev
			break
		}
	}
	if start.Module != "lib/submod" {
		t.Errorf("got Module %q; want %q", start.Module, filepath.Join("lib", "submod"))
	}
}
