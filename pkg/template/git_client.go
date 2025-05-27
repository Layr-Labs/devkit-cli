package template

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CloneEventType enumerates the kinds of things git clone can tell us
type CloneEventType int

const (
	EventSubmoduleDiscovered CloneEventType = iota
	EventSubmoduleCloneStart
	EventProgress
	EventCloneComplete
	EventCloneFailed
)

// CloneEvent is a single “thing that happened” during clone
type CloneEvent struct {
	Type     CloneEventType
	Parent   string // for submodule events
	Module   string // current module path
	Name     string // submodule name or module path
	URL      string // for discovery
	Ref      string // current ref we are cloning from
	Progress int    // 0–100
}

// Reporter consumes CloneEvents
type Reporter interface {
	Report(CloneEvent)
}

// Runner lets us inject/mock command execution in tests
type Runner interface {
	// CommandContext mirrors exec.CommandContext
	CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd
}

// execRunner is the real-world Runner
type execRunner struct{}

func (execRunner) CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

// GitClient does our actual clone + parsing
type GitClient struct {
	Runner         Runner
	ReceivingRegex *regexp.Regexp
	CloningRegex   *regexp.Regexp
	SubmoduleRegex *regexp.Regexp
}

// NewGitClient builds a GitClient using the real exec
func NewGitClient() *GitClient {
	return &GitClient{
		Runner:         execRunner{},
		ReceivingRegex: regexp.MustCompile(`Receiving objects:\s+(\d+)%`),
		CloningRegex:   regexp.MustCompile(`Cloning into ['"]?(.+?)['"]?\.{3}`),
		SubmoduleRegex: regexp.MustCompile(
			`^Submodule ['"]?([^'"]+)['"]? \(([^)]+)\) registered for path ['"]?(.+?)['"]?$`,
		),
	}
}

// NewGitClientWithRunner enables injecting a custom Runner (e.g. in tests)
func NewGitClientWithRunner(r Runner) *GitClient {
	g := NewGitClient()
	g.Runner = r
	return g
}

// Clone runs the following to enable clones from SHAs, tags and branches:
//   - git clone --no-checkout --depth 1 <repoURL> <dest>
//   - git -C <dest> checkout --quiet <ref>
//   - git -C <dest> submodule update --init --recursive --progress
func (g *GitClient) Clone(
	ctx context.Context,
	repoURL, ref, dest string,
	config GitFetcherConfig,
	reporter Reporter,
) error {
	// Derive a short name for the top-level module
	repoName := filepath.Base(strings.TrimSuffix(repoURL, ".git"))

	// Plain clone (no --depth, no parsing)
	cloneArgs := []string{"clone", "--no-checkout", "--progress", repoURL, dest}
	cloneCmd := g.Runner.CommandContext(ctx, "git", cloneArgs...)

	// In verbose mode print git logs directly
	if config.Verbose {
		cloneCmd.Stdout, cloneCmd.Stderr = os.Stdout, os.Stderr
		if err := cloneCmd.Run(); err != nil {
			if reporter != nil {
				reporter.Report(CloneEvent{Type: EventCloneFailed})
			}
			return fmt.Errorf("git clone: %w", err)
		}
	} else {
		// Report progress to reporter to handle structured logging
		stderr, err := cloneCmd.StderrPipe()
		if err != nil {
			reporter.Report(CloneEvent{Type: EventCloneFailed})
			return fmt.Errorf("stderr pipe: %w", err)
		}
		if err := cloneCmd.Start(); err != nil {
			reporter.Report(CloneEvent{Type: EventCloneFailed})
			return fmt.Errorf("start clone: %w", err)
		}
		// parse the initial clone progress into events
		if err := g.ParseCloneOutput(stderr, reporter, dest, ref); err != nil {
			return fmt.Errorf("parsing clone output: %w", err)
		}
		if err := cloneCmd.Wait(); err != nil {
			reporter.Report(CloneEvent{Type: EventCloneFailed})
			return fmt.Errorf("git clone: %w", err)
		}
	}

	// Checkout the desired ref after cloning to pull submodules from the correct refs
	coCmd := g.Runner.CommandContext(ctx,
		"git", "-C", dest, "checkout", "--quiet", ref,
	)
	if config.Verbose {
		coCmd.Stdout, coCmd.Stderr = os.Stdout, os.Stderr
	}
	if err := coCmd.Run(); err != nil {
		// Report failure
		if reporter != nil {
			reporter.Report(CloneEvent{Type: EventCloneFailed})
		}
		// If checkout fails, remove the .git folder so user isn't left in a broken state
		err = os.RemoveAll(filepath.Join(dest, ".git"))
		if err != nil {
			return fmt.Errorf("removing .git dir after git checkout %q failed: %w", ref, err)
		}
		// return err
		return fmt.Errorf("git checkout %q failed: %w", ref, err)
	}

	// Force a 100% report for the top-level repo (in case parse missed it)
	if reporter != nil {
		reporter.Report(CloneEvent{Type: EventProgress, Module: repoName, Ref: ref, Progress: 100})
	}

	// Recursive submodule update with progress
	smArgs := []string{"-C", dest, "submodule", "update", "--init", "--recursive", "--depth=1", "--progress"}
	smCmd := g.Runner.CommandContext(ctx, "git", smArgs...)
	// If verbose, print git logs directly
	if config.Verbose {
		smCmd.Stdout, smCmd.Stderr = os.Stdout, os.Stderr
		if err := smCmd.Run(); err != nil {
			return fmt.Errorf("git submodule update: %w", err)
		}
	} else {
		// Report progress to reporter to handle structured logging
		stderr, err := smCmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("stderr pipe: %w", err)
		}
		if err := smCmd.Start(); err != nil {
			return fmt.Errorf("start submodule update: %w", err)
		}
		// this will emit EventProgress etc. as submodules download
		if err := g.ParseCloneOutput(stderr, reporter, dest, ref); err != nil {
			return fmt.Errorf("parsing clone output: %w", err)
		}
		if err := smCmd.Wait(); err != nil {
			return fmt.Errorf("git submodule update: %w", err)
		}
	}

	// Notify done
	if reporter != nil {
		reporter.Report(CloneEvent{Type: EventCloneComplete})
	}
	return nil
}

// ParseCloneOutput scans git’s progress output and emits events
func (g *GitClient) ParseCloneOutput(r io.Reader, rep Reporter, dest string, ref string) error {
	scanner := bufio.NewScanner(r)
	parent, module := ".", ""
	for scanner.Scan() {
		line := scanner.Text()

		// Submodule discovery
		if m := g.SubmoduleRegex.FindStringSubmatch(line); len(m) == 4 {
			name, url, full := m[1], m[2], m[3]
			parent = strings.TrimSuffix(full, name)
			rep.Report(CloneEvent{
				Type:   EventSubmoduleDiscovered,
				Parent: parent,
				Name:   name,
				URL:    url,
				Ref:    ref,
			})
			continue
		}

		// New clone path
		if m := g.CloningRegex.FindStringSubmatch(line); len(m) == 2 {
			// Finish previous module
			if module != "" {
				rep.Report(CloneEvent{Type: EventProgress, Module: module, Ref: ref, Progress: 100})
			}
			raw := m[1]
			parts := strings.Split(raw, filepath.Join(dest, parent))
			if len(parts) > 1 {
				raw = strings.TrimPrefix(parts[len(parts)-1], string(os.PathSeparator))
			}
			module = raw
			rep.Report(CloneEvent{Type: EventSubmoduleCloneStart, Parent: parent, Module: module, Ref: ref})
			rep.Report(CloneEvent{Type: EventProgress, Module: module, Ref: ref, Progress: 0})
			continue
		}

		// Percent progress
		if m := g.ReceivingRegex.FindStringSubmatch(line); len(m) == 2 {
			var pct int
			n, err := fmt.Sscanf(m[1], "%d", &pct)
			if err != nil || n != 1 {
				return fmt.Errorf("failed to parse integer from %q: %w", m[1], err)
			}
			rep.Report(CloneEvent{Type: EventProgress, Module: module, Progress: pct, Ref: ref})
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan stderr: %w", err)
	}
	return nil
}
