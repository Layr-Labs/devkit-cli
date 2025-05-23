package template

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
)

type GitClient interface {
	Clone(ctx context.Context, repoURL, dest string, opts CloneOptions) error
	CloneFromCache(ctx context.Context, cacheDir, dest string, opts CloneOptions) error
	RetryClone(ctx context.Context, repoURL, dest string, opts CloneOptions, maxRetries int) error
	Checkout(ctx context.Context, repoDir, commit string) error
	WorktreeCheckout(ctx context.Context, mirrorPath, commit, worktreePath string) error
	ResolveRemoteCommit(ctx context.Context, repoURL, branch string) (string, error)
	SubmoduleList(ctx context.Context, repoDir string) ([]Submodule, error)
	SubmoduleCommit(ctx context.Context, repoDir, path string) (string, error)
	SubmoduleClone(
		ctx context.Context,
		submodule Submodule,
		commit string,
		repoUrl string,
		targetDir string,
		repoDir string,
		opts CloneOptions,
	) error
	CheckoutCommit(ctx context.Context, repoDir, commitHash string) error
	StageSubmodule(ctx context.Context, repoDir, path, sha string) error
	SetSubmoduleURL(ctx context.Context, repoDir, name, url string) error
	ActivateSubmodule(ctx context.Context, repoDir, name string) error
	AddSubmodule(ctx context.Context, repoDir, url, path string) error
	SubmoduleInit(ctx context.Context, repoDir string, opts CloneOptions) error
}

type CloneOptions struct {
	// Ref is the branch, commit or tag to checkout after cloning
	Ref        string
	Bare       bool
	ProgressCB func(int)
}

type Submodule struct {
	Name, Path, URL, Branch, Commit string
}

type SubmoduleFailure struct {
	mod Submodule
	err error
}

type execGitClient struct {
	repoLocksMu    sync.Mutex
	repoLocks      map[string]*sync.Mutex
	receivingRegex *regexp.Regexp
	isSHA          *regexp.Regexp
}

func NewGitClient() GitClient {
	return &execGitClient{
		repoLocks:      make(map[string]*sync.Mutex),
		receivingRegex: regexp.MustCompile(`Receiving objects:\s+(\d+)%`),
		isSHA:          regexp.MustCompile(`^[0-9a-f]{40}$`),
	}
}

func (g *execGitClient) run(ctx context.Context, dir string, opts CloneOptions, args ...string) ([]byte, error) {
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// capture stdout
	var out bytes.Buffer
	cmd.Stdout = &out

	// capture stderr
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	// start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git %v failed to start: %w", args, err)
	}

	// read stderr for progress
	scanner := bufio.NewScanner(stderr)
	var lastReportedProgress int
	for scanner.Scan() {
		line := scanner.Text()

		// look for progress line with percentage (e.g., receiving objects: 100%)
		if match := g.receivingRegex.FindStringSubmatch(line); match != nil {
			pct := percentToInt(match[1])
			// only report progress if the percentage has changed
			if pct != lastReportedProgress {
				if opts.ProgressCB != nil {
					// call ProgressCB with updated progress
					opts.ProgressCB(pct)
				}
				lastReportedProgress = pct
			}
		}
	}

	// handle any errors encountered in stderr
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("stderr scan error: %w", err)
	}

	// wait for the command to complete
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("git %v failed: %w\nOutput:\n%s", args, err, out.String())
	}

	return out.Bytes(), nil
}

func (g *execGitClient) Clone(ctx context.Context, repoURL, cacheDir string, opts CloneOptions) error {
	// make or reuse a bare repo in cacheDir
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %v", cacheDir, err)
	}
	// if not already initialized as bare, do it
	if _, err := os.Stat(filepath.Join(cacheDir, "HEAD")); os.IsNotExist(err) {
		if out, err := g.run(ctx, cacheDir, CloneOptions{}, "init", "--bare"); err != nil {
			return fmt.Errorf("git init --bare: %s", out)
		}
		if out, err := g.run(ctx, cacheDir, CloneOptions{}, "remote", "add", "origin", repoURL); err != nil {
			return fmt.Errorf("git remote add origin: %s", out)
		}
	}
	// always fetch all branches+tags into proper refs
	fetchArgs := []string{
		"fetch", "--prune", "origin",
		"+refs/heads/*:refs/heads/*",
		"+refs/tags/*:refs/tags/*",
	}
	// conditionally append --progress
	if opts.ProgressCB != nil && progress.IsTTY() {
		fetchArgs = append(fetchArgs, "--progress")
	}
	// run the fetch to pull the refs
	if out, err := g.run(ctx, cacheDir, opts, fetchArgs...); err != nil {
		return fmt.Errorf("git fetch cache update failed: %s", out)
	}
	return nil
}

func (g *execGitClient) CloneFromCache(ctx context.Context, cacheDir, dest string, opts CloneOptions) error {
	// clean and mkdir dest
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %v", dest, err)
	}

	// do a shallow, single-branch copy from the bare cache works for branches or tags; for SHA we fetch after
	cloneArgs := []string{
		"clone", "--no-checkout", "--depth=1",
		"--shared", cacheDir,
		dest,
	}
	// conditionally append --progress
	if opts.ProgressCB != nil && progress.IsTTY() {
		cloneArgs = append(cloneArgs, "--progress")
	}
	// if ref is a branch or tag name, pass --branch
	if !g.isSHA.MatchString(opts.Ref) {
		cloneArgs = append(cloneArgs, "--branch", opts.Ref)
	}
	if out, err := g.run(ctx, "", opts, cloneArgs...); err != nil {
		return fmt.Errorf("git clone from cache failed: %s", out)
	}

	// check for provided sha
	if g.isSHA.MatchString(opts.Ref) {
		// detached‐head checkout of the SHA
		if out, err := g.run(ctx, dest, opts, "fetch", "--depth=1", "origin", opts.Ref); err != nil {
			return fmt.Errorf("git fetch sha %s: %s", opts.Ref, out)
		}
		if _, err := g.run(ctx, dest, CloneOptions{}, "checkout", opts.Ref); err != nil {
			return fmt.Errorf("git checkout %s: %w", opts.Ref, err)
		}
	} else {
		// just checkout the branch/tag
		if _, err := g.run(ctx, dest, CloneOptions{}, "checkout", "-f", "HEAD"); err != nil {
			return fmt.Errorf("git checkout HEAD: %w", err)
		}
	}

	return nil
}

func (g *execGitClient) RetryClone(ctx context.Context, repoURL, dest string, opts CloneOptions, maxRetries int) error {
	var err error
	for attempt := 0; attempt+1 <= maxRetries; attempt++ {
		err = g.Clone(ctx, repoURL, dest, opts)
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(attempt+1) * 250 * time.Millisecond)
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

func (g *execGitClient) SubmoduleClone(
	ctx context.Context,
	submodule Submodule,
	commit string,
	repoUrl string,
	targetDir string,
	repoDir string,
	opts CloneOptions,
) error {
	// clean up target
	_ = os.RemoveAll(targetDir)

	// clone from provided repoUrl (cachePath or URL)
	if err := g.CloneFromCache(ctx, repoUrl, targetDir, opts); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// lock against repoDir to guard global state
	repoLock := g.lockForRepo(repoDir)
	repoLock.Lock()
	defer repoLock.Unlock()

	// stage submodule in parent
	if err := g.StageSubmodule(ctx, repoDir, submodule.Path, commit); err != nil {
		return fmt.Errorf("stage failed: %w", err)
	}

	// set submodule URL
	if err := g.SetSubmoduleURL(ctx, repoDir, submodule.Name, submodule.URL); err != nil {
		return fmt.Errorf("set-url failed: %w", err)
	}

	// activate submodule
	if err := g.ActivateSubmodule(ctx, repoDir, submodule.Name); err != nil {
		return fmt.Errorf("activate failed: %w", err)
	}

	return nil
}

func (g *execGitClient) Checkout(ctx context.Context, repoDir, commit string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "checkout", commit)
	return err
}

func (g *execGitClient) WorktreeCheckout(ctx context.Context, mirrorPath, commit, worktreePath string) error {
	_, err := g.run(ctx, mirrorPath, CloneOptions{}, "worktree", "add", "--detach", worktreePath, commit)
	return err
}

func (g *execGitClient) SubmoduleList(ctx context.Context, repoDir string) ([]Submodule, error) {
	out, err := g.run(ctx, repoDir, CloneOptions{}, "--no-pager", "config", "-f", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var subs []Submodule
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimPrefix(parts[0], "submodule.")
		name = strings.TrimSuffix(name, ".path")
		path := parts[1]
		urlOut, err := g.run(ctx, repoDir, CloneOptions{}, "config", "-f", ".gitmodules", "--get", fmt.Sprintf("submodule.%s.url", name))
		if err != nil {
			return nil, err
		}
		branchOut, err := g.run(ctx, repoDir, CloneOptions{}, "config", "-f", ".gitmodules", "--get", fmt.Sprintf("submodule.%s.branch", name))
		branch := ""
		if err == nil {
			branch = strings.TrimSpace(string(branchOut))
		}
		subs = append(subs, Submodule{
			Name:   name,
			Path:   path,
			URL:    strings.TrimSpace(string(urlOut)),
			Branch: branch,
		})
	}
	return subs, nil
}

func (g *execGitClient) SubmoduleInit(ctx context.Context, repoDir string, opts CloneOptions) error {
	_, err := g.run(ctx, repoDir, opts, "submodule", "update", "--recursive", "--depth", "2", "--init", "--progress")
	return err
}

func (g *execGitClient) SubmoduleCommit(ctx context.Context, repoDir, path string) (string, error) {
	out, err := g.run(ctx, repoDir, CloneOptions{}, "ls-tree", "HEAD", path)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) < 3 {
		return "", fmt.Errorf("unexpected ls-tree output: %s", out)
	}
	return fields[2], nil
}

func (g *execGitClient) ResolveRemoteCommit(ctx context.Context, repoURL, ref string) (string, error) {
	args := []string{"ls-remote", repoURL}
	if ref != "" {
		args = append(args, ref)
	} else {
		args = append(args, "HEAD")
	}
	out, err := g.run(ctx, "", CloneOptions{}, args...)
	if err != nil {
		return "", err
	}
	// if len is 0 with no error, we've been provided a commit hash, so let's use it
	if len(out) == 0 {
		return ref, nil
	}

	// otherwise, parse the output and take the first commit hash
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return "", fmt.Errorf("unexpected output: %s", out)
	}
	return fields[0], nil
}

func (g *execGitClient) CheckoutCommit(ctx context.Context, repoDir, commitHash string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "checkout", commitHash)
	return err
}

func (g *execGitClient) StageSubmodule(ctx context.Context, repoDir, path, sha string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "update-index", "--add", "--cacheinfo", "160000", sha, path)
	return err
}

func (g *execGitClient) SetSubmoduleURL(ctx context.Context, repoDir, name, url string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "config", "--local", fmt.Sprintf("submodule.%s.url", name), url)
	return err
}

func (g *execGitClient) ActivateSubmodule(ctx context.Context, repoDir, name string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "config", "--local", fmt.Sprintf("submodule.%s.active", name), "true")
	return err
}

func (g *execGitClient) AddSubmodule(ctx context.Context, repoDir, url, path string) error {
	_, err := g.run(ctx, repoDir, CloneOptions{}, "submodule", "add", url, path)
	return err
}

// Helper to return a per-repo mutex to synchronise operations on the same repo
func (g *execGitClient) lockForRepo(repo string) *sync.Mutex {
	g.repoLocksMu.Lock()
	defer g.repoLocksMu.Unlock()
	mu, ok := g.repoLocks[repo]
	if !ok {
		mu = &sync.Mutex{}
		g.repoLocks[repo] = mu
	}
	return mu
}

// Helper function to convert the percentage from string to int
func percentToInt(s string) int {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0
	}
	return i
}
