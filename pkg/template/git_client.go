package template

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GitClient interface {
	Clone(ctx context.Context, repoURL, dest string, opts CloneOptions) error
	Checkout(ctx context.Context, repoDir, commit string) error
	WorktreeCheckout(ctx context.Context, mirrorPath, commit, worktreePath string) error
	SubmoduleList(ctx context.Context, repoDir string) ([]Submodule, error)
	SubmoduleCommit(ctx context.Context, repoDir, path string) (string, error)
	ResolveRemoteCommit(ctx context.Context, repoURL, branch string) (string, error)
	RetryClone(ctx context.Context, repoURL, dest string, opts CloneOptions, maxRetries int) error
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
	Ref         string
	Depth       int
	Bare        bool
	Dissociate  bool
	NoHardlinks bool
	ProgressCB  func(int)
}

type Submodule struct {
	Name, Path, URL, Branch string
}

type SubmoduleFailure struct {
	mod Submodule
	err error
}

type execGitClient struct {
	repoLocksMu    sync.Mutex
	repoLocks      map[string]*sync.Mutex
	receivingRegex *regexp.Regexp
}

func NewGitClient() GitClient {
	return &execGitClient{
		repoLocks:      make(map[string]*sync.Mutex),
		receivingRegex: regexp.MustCompile(`Receiving objects:\s+(\d+)%`),
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

func (g *execGitClient) Clone(ctx context.Context, repoURL, dest string, opts CloneOptions) error {
	args := []string{"clone"}

	// TODO(seanmcgary): commented this out since this breaks when passing a commit hash
	// since a commit hash is not a branch.
	//
	// the other flags also break the case of passing a commit hash

	// handle flags for bare, depth, branch, dissociate, and no-hardlinks
	//if opts.Bare {
	//	args = append(args, "--bare")
	//}
	// if opts.Depth > 0 {
	// 	args = append(args, fmt.Sprintf("--depth=%d", opts.Depth))
	// }
	// if opts.Ref != "" {
	// 	args = append(args, "-b", opts.Ref)
	// }
	// if opts.Dissociate {
	// 	args = append(args, "--dissociate")
	// }
	// if opts.NoHardlinks {
	// 	args = append(args, "--no-hardlinks")
	// }

	// add the --progress flag for tracking progress
	args = append(args, "--progress")

	// add the repository URL and destination path
	args = append(args, repoURL, dest)

	// call run to execute the command and capture progress
	_, err := g.run(ctx, "", opts, args...)
	if err != nil {
		return fmt.Errorf("failed to clone: '%s' '%s' '%s' %w", repoURL, opts.Ref, dest, err)
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
	if err := g.Clone(ctx, repoUrl, targetDir, opts); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// checkout to commit
	if err := g.Checkout(ctx, targetDir, commit); err != nil {
		return fmt.Errorf("checkout failed: %w", err)
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
	_, err := g.run(ctx, repoDir, opts, "submodule", "update", "--recursive", "--init", "--progress")
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
