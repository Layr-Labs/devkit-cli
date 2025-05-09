package template

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type Fetcher interface {
	Fetch(templateURL, targetDir string, verbose bool) error
}

type GitFetcher struct {
	MaxDepth int
}

type Submodule struct {
	Name   string
	Path   string
	URL    string
	Branch string
}

type ProgressGrid struct {
	order    []string
	progress map[string]int
	lines    int
	frozen   bool
}

var (
	outputMu          sync.Mutex
	repoLocksMu       sync.Mutex
	visitMu           sync.Mutex
	repoLocks         = map[string]*sync.Mutex{}
	visitedSubmodules = map[string]struct{}{}
	receivingRegex    = regexp.MustCompile(`Receiving objects:\s+(\d+)%`)
)

func shouldVisit(path string) bool {
	visitMu.Lock()
	defer visitMu.Unlock()
	if _, ok := visitedSubmodules[path]; ok {
		return false
	}
	visitedSubmodules[path] = struct{}{}
	return true
}

func lockForRepo(repo string) *sync.Mutex {
	repoLocksMu.Lock()
	defer repoLocksMu.Unlock()
	mu, ok := repoLocks[repo]
	if !ok {
		mu = &sync.Mutex{}
		repoLocks[repo] = mu
	}
	return mu
}

func printLog(format string, args ...any) {
	outputMu.Lock()
	defer outputMu.Unlock()

	output := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}
	fmt.Print(output)
}

func printError(format string, args ...any) {
	outputMu.Lock()
	defer outputMu.Unlock()

	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	// ANSI red: \033[31m ... \033[0m
	fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m", msg)
}

func printGitError(grid *ProgressGrid, mu *sync.Mutex, modPath, reason string, err error) {
	// freeze the progressGrid before printing error
	mu.Lock()
	renderProgressGrid(grid)
	grid.frozen = true
	mu.Unlock()

	// print log with ANSI red formatting wrapper
	printError("\n❌ %s for %s: %v\n", reason, modPath, err)
}

func percentToInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func buildBar(pct int) string {
	total := 20
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * total / 100
	unfilled := total - filled
	if unfilled > 0 {
		unfilled--
	}
	return fmt.Sprintf("[%s>%s]", strings.Repeat("=", filled), strings.Repeat(" ", unfilled))
}

func updateProgress(grid *ProgressGrid, name string, pct int) {
	outputMu.Lock()
	defer outputMu.Unlock()
	// prevent progress moving backwards
	if grid != nil && grid.progress[name] < pct {
		grid.progress[name] = pct
	}
}

func renderProgressGrid(grid *ProgressGrid) {
	outputMu.Lock()
	defer outputMu.Unlock()

	if grid == nil || len(grid.order) == 0 || grid.frozen {
		return
	}
	if grid.lines > 0 {
		fmt.Printf("\033[%dA", grid.lines)
	}
	grid.lines = len(grid.order)

	for _, name := range grid.order {
		pct := grid.progress[name]
		bar := buildBar(pct)
		fmt.Printf("\r\033[K%s %3d%%   %s\n", bar, pct, name)
	}
}

func completeGrid(grid *ProgressGrid) {
	if grid == nil {
		return
	}
	for _, name := range grid.order {
		grid.progress[name] = 100
	}
	renderProgressGrid(grid)
}

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

func getRemoteHEADCommit(repoURL, branch string) (string, error) {
	args := []string{"ls-remote", repoURL}
	if branch != "" {
		args = append(args, branch)
	} else {
		args = append(args, "HEAD")
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		return "", fmt.Errorf("unexpected ls-remote output: %q", out)
	}
	return fields[0], nil
}

func listSubmodules(repoDir string) ([]Submodule, error) {
	cmd := exec.Command("git", "config", "-f", ".gitmodules", "--get-regexp", "^submodule\\..*\\.path$")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read .gitmodules: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	var subs []Submodule
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		name := strings.TrimPrefix(fields[0], "submodule.")
		name = strings.TrimSuffix(name, ".path")
		path := fields[1]

		urlCmd := exec.Command("git", "config", "-f", ".gitmodules", "--get", fmt.Sprintf("submodule.%s.url", name))
		urlCmd.Dir = repoDir
		urlOut, err := urlCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to clone template: %w", err)
		}

		branchCmd := exec.Command("git", "config", "-f", ".gitmodules", "--get", fmt.Sprintf("submodule.%s.branch", name))
		branchCmd.Dir = repoDir
		branchOut, err := branchCmd.Output()
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

func getSubmoduleCommit(repoDir, subPath string) (string, error) {
	cmd := exec.Command("git", "ls-tree", "HEAD", subPath)
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ls-tree for %s: %w", subPath, err)
	}
	fields := strings.Fields(string(out))
	if len(fields) < 3 {
		return "", fmt.Errorf("unexpected ls-tree output: %s", out)
	}
	return fields[2], nil // object SHA
}

func checkoutCommit(repoDir, commitHash string) error {
	cmd := exec.Command("git", "checkout", commitHash)
	cmd.Dir = repoDir
	return cmd.Run()
}

func stageSubmodule(repoDir, path, sha string) error {
	cmd := exec.Command("git", "update-index", "--add", "--cacheinfo", "160000", sha, path)
	cmd.Dir = repoDir
	return cmd.Run()
}

func setSubmoduleURL(repoDir, name, url string) error {
	cmd := exec.Command("git", "config", "--local", fmt.Sprintf("submodule.%s.url", name), url)
	cmd.Dir = repoDir
	return cmd.Run()
}

func activateSubmodule(repoDir, name string) error {
	cmd := exec.Command("git", "config", "--local", fmt.Sprintf("submodule.%s.active", name), "true")
	cmd.Dir = repoDir
	return cmd.Run()
}

func absorbGitDirs(repoDir string) error {
	cmd := exec.Command("git", "submodule", "absorbgitdirs")
	cmd.Dir = repoDir
	return cmd.Run()
}

func notExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func copyFromCache(mirrorPath, destPath string, updateProgress func(int)) error {
	// lock the destination path
	lock := lockForRepo(destPath)
	lock.Lock()
	defer lock.Unlock()

	// proactively clean up before cloning
	_ = os.RemoveAll(destPath)

	// perform a clone from the cache to the workingDir
	cmd := exec.Command("git", "clone", "--progress", mirrorPath, destPath)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// track progress by scanning stderr for Receiving objects msgs
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if match := receivingRegex.FindStringSubmatch(line); match != nil {
			pct := percentToInt(match[1])
			updateProgress(pct)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stderr scan error: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		_ = os.RemoveAll(destPath)
		return err
	}

	updateProgress(100)
	return nil
}

func cloneWithCache(repoURL, branch, cachePath, destPath string, noCache bool, updateProgress func(int)) error {
	// only pull the repo if cache is missing or noCache is provided
	if noCache || notExists(cachePath) {
		if !notExists(cachePath) {
			_ = os.RemoveAll(cachePath)
		}

		// clone with --bare to keep cache tree clean
		args := []string{"clone", "--bare", "--progress"}
		if branch != "" {
			args = append(args, "-b", branch)
		}
		args = append(args, repoURL, cachePath)

		cmd := exec.Command("git", args...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return fmt.Errorf("stderr pipe: %w", err)
		}
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("clone start: %w", err)
		}
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := receivingRegex.FindStringSubmatch(line); match != nil {
				updateProgress(percentToInt(match[1]))
			}
		}
		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("clone failed: %s: %w", repoURL, err)
		}
	}

	// copy the cached copy to destPath
	return copyFromCache(cachePath, destPath, updateProgress)
}

func cloneSubmodules(repoURL string, repoName string, repoDir string, noCache bool, depth int, maxDepth int) error {
	// exit early if the maxDepth is exceeded
	if maxDepth != -1 && depth >= maxDepth {
		return nil
	}

	// pull the submodules for this repo
	submodules, err := listSubmodules(repoDir)
	if err != nil {
		return fmt.Errorf("list submodules: %w", err)
	}
	if len(submodules) == 0 {
		return nil
	}

	// create the submodule cache if it is missing
	cacheDir := filepath.Join(".", ".git-submodule-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	// set up a new grid to hold progress
	grid := &ProgressGrid{
		order:    []string{},
		progress: map[string]int{},
	}

	// print discoveries
	printLog("\nDiscovered submodules in %s (%s):\n", repoName, repoURL)
	for _, mod := range submodules {
		printLog(" - %s → %s (%s)\n", mod.Name, mod.Path, mod.URL)
		grid.order = append(grid.order, mod.Path)
	}
	printLog("")

	// set up a waitGroup to perform a concurrent fanout and join
	var wg sync.WaitGroup
	var mu sync.Mutex

	// for each submodule, perform cloneWithCache and copy to workingCopyPath
	for _, mod := range submodules {
		// shadow to avoid race
		mod := mod
		// mark that we're waiting on this routine
		wg.Add(1)
		// define the routine
		go func() {
			// wait for this goroutine to finish
			defer wg.Done()

			commitHash, err := getSubmoduleCommit(repoDir, mod.Path)
			if err != nil {
				printLog("failed to get commit for %s: %v\n", mod.Path, err)
				return
			}
			cacheName := fmt.Sprintf("%s-%s.git", mod.Name, commitHash)
			cachePath := filepath.Join(cacheDir, cacheName)
			workingCopyPath := filepath.Join(repoDir, mod.Path)

			// we do not clone from Branch because we immediately checkout the commit ref'd in parents submodule
			// passing in the Branch could move us to a tree without the commit we want, we can safely ignore it
			err = cloneWithCache(mod.URL, "", cachePath, workingCopyPath, noCache, func(pct int) {
				mu.Lock()
				updateProgress(grid, mod.Path, pct)
				renderProgressGrid(grid)
				mu.Unlock()
			})
			if err != nil {
				printGitError(grid, &mu, mod.Path, "clone failed", err)
				return
			}

			// lock before modifying shared repo state
			lock := lockForRepo(repoDir)
			lock.Lock()
			defer lock.Unlock()

			// checkout the submodule on the referenced commitHash
			if err := checkoutCommit(workingCopyPath, commitHash); err != nil {
				printGitError(grid, &mu, mod.Path, "checkout failed", err)
				return
			}

			// stage the submodule in the parent index at the correct SHA
			if err := stageSubmodule(workingCopyPath, mod.Path, commitHash); err != nil {
				printGitError(grid, &mu, mod.Path, "stage submodule failed", err)
				return
			}

			// set the submodules url in local git config
			if err := setSubmoduleURL(repoDir, mod.Name, mod.URL); err != nil {
				printGitError(grid, &mu, mod.Path, "set submodule URL failed", err)
				return
			}

			// mark submodule as active in local config
			if err := activateSubmodule(repoDir, mod.Name); err != nil {
				printGitError(grid, &mu, mod.Path, "activate submodule failed", err)
				return
			}

			// absorb .git directories into the parent repo
			if err := absorbGitDirs(repoDir); err != nil {
				printGitError(grid, &mu, mod.Path, "absorb git dirs failed", err)
				return
			}
		}()
	}
	// block until all goroutines call Done()
	wg.Wait()

	// complete this grid
	completeGrid(grid)

	// guard against duplicate submodules with same name but different paths
	for _, mod := range submodules {
		nestedPath := filepath.Join(repoDir, mod.Path)
		if shouldVisit(nestedPath) {
			_ = cloneSubmodules(mod.URL, mod.Name, nestedPath, noCache, depth+1, maxDepth)
		}
	}

	return nil
}

func (g *GitFetcher) Fetch(templateURL, targetDir string, verbose bool, noCache bool) error {
	// parse the provided templateURL
	repoURL, branch := parseGitHubURL(templateURL)
	printLog("Cloning template repo: %s → %s\n\n", repoURL, targetDir)

	// set up a new grid to hold progress
	grid := &ProgressGrid{
		order:    []string{targetDir},
		progress: map[string]int{targetDir: 0},
	}
	renderProgressGrid(grid)

	// create the .git-template-cache dir if missing
	cacheDir := filepath.Join(".", ".git-template-cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("create template cache dir: %w", err)
	}

	// get HEAD commit from remote to id the clone we're about to make
	commitHash, err := getRemoteHEADCommit(repoURL, branch)
	if err != nil {
		return fmt.Errorf("get remote HEAD commit failed: %w", err)
	}
	templateName := filepath.Base(strings.TrimSuffix(repoURL, ".git"))
	cacheName := fmt.Sprintf("%s-%s.git", templateName, commitHash)
	cachePath := filepath.Join(cacheDir, cacheName)

	// clone into cache if needed
	err = cloneWithCache(repoURL, branch, cachePath, targetDir, noCache, func(pct int) {
		grid.progress[targetDir] = pct
		renderProgressGrid(grid)
	})
	if err != nil {
		return fmt.Errorf("clone with cache failed: %w", err)
	}

	// complete progress in the grid
	completeGrid(grid)

	// recurse into submodules
	if err := cloneSubmodules(repoURL, templateName, targetDir, noCache, 0, g.MaxDepth); err != nil {
		return fmt.Errorf("submodule clone with cache failed: %w", err)
	}

	// notify that all cloning is done
	printLog("\nClone complete.\n")
	return nil
}
