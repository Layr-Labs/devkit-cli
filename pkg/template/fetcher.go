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
	MaxDepth       int
	MaxRetries     int
	MaxConcurrency int
}

type Submodule struct {
	Name   string
	Path   string
	URL    string
	Branch string
}

type SubmoduleFailure struct {
	mod Submodule
	err error
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

	output := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(output, "\n") {
		output += "\n"
	}

	// ANSI red: \033[31m ... \033[0m
	fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m", output)
}

func percentToInt(s string) int {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0
	}
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
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git ls-remote failed: %w\nOutput:\n%s", err, out)
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

func notExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func cloneWithCache(repoURL, branch, cachePath, destPath string, noCache bool, updateProgress func(int)) error {
	// only pull the repo if cache is missing or noCache is provided
	if noCache || notExists(cachePath) {
		if noCache {
			_ = os.RemoveAll(cachePath)
		}

		// clone with --bare to keep cache tree clean
		if err := runClone(repoURL, branch, []string{"--bare"}, cachePath, updateProgress); err != nil {
			return err
		}
	}

	// copy the cached copy to destPath
	return copyFromCache(cachePath, destPath, updateProgress)
}

func copyFromCache(mirrorPath, destPath string, updateProgress func(int)) error {
	// proactively clean up before cloning
	if err := os.RemoveAll(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove destPath %s: %w", destPath, err)
	}

	// perform a clone from the cache to the workingDir
	cmd := exec.Command("git", "clone", "--dissociate", "--no-hardlinks", "--progress", mirrorPath, destPath)
	// force git to flush stderr early
	cmd.Env = append(os.Environ(),
		"GIT_PROGRESS_DELAY=0",
		"GIT_FLUSH=1",
	)
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

	// mark progress as 100 before moving on
	updateProgress(100)
	return nil
}

func runClone(repoURL, branch string, args []string, dest string, updateProgress func(int)) error {
	// lock whilst this clone takes place
	lock := lockForRepo(dest)
	lock.Lock()
	defer lock.Unlock()

	// clone with --progress to update progress report
	args = append([]string{"clone", "--progress"}, args...)
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, repoURL, dest)

	// exec the git command to clone the target repo to cache
	cmd := exec.Command("git", args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("clone start: %w", err)
	}

	// track progress according to `Receiving Objects`
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		if match := receivingRegex.FindStringSubmatch(line); match != nil {
			updateProgress(percentToInt(match[1]))
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stderr scan error: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// mark progress as 100 before moving on
	updateProgress(100)
	return nil
}

func attemptSubmoduleSetup(mod Submodule, grid *ProgressGrid, repoDir string, cacheDir string, noCache bool) error {
	// unpack submodule commit hash from the parent ls-tree
	commitHash, err := getSubmoduleCommit(repoDir, mod.Path)
	if err != nil {
		return fmt.Errorf("failed to get commit for %s: %v\n", mod.Path, err)
	}
	baseName := filepath.Base(mod.Name)
	cacheName := fmt.Sprintf("%s-%s.git", baseName, commitHash)
	cachePath := filepath.Join(cacheDir, cacheName)
	workingCopyPath := filepath.Join(repoDir, mod.Path)

	// lock against workingCopyPath to guard against modules with same path in same parent
	cloneLock := lockForRepo(workingCopyPath)
	cloneLock.Lock()
	defer cloneLock.Unlock()

	// intentionally omit branch; we checkout a specific commit immediately after
	// passing in the Branch could move us to a tree without the commit we want
	if err := cloneWithCache(mod.URL, "", cachePath, workingCopyPath, noCache, func(pct int) {
		if pct < 90 {
			updateProgress(grid, mod.Path, pct)
			renderProgressGrid(grid)
		}
	}); err != nil {
		return fmt.Errorf("clone: %w", err)
	}

	// checkout the submodule on the referenced commitHash
	if err := checkoutCommit(workingCopyPath, commitHash); err != nil {
		return fmt.Errorf("checkout: %w", err)
	}
	// stage the submodule in the parent index at the correct SHA
	if err := stageSubmodule(workingCopyPath, mod.Path, commitHash); err != nil {
		return fmt.Errorf("stage: %w", err)
	}

	// lock against repoDir to guard global state
	repoLock := lockForRepo(repoDir)
	repoLock.Lock()
	defer repoLock.Unlock()

	// set the submodules url in local git config
	if err := setSubmoduleURL(repoDir, mod.Name, mod.URL); err != nil {
		return fmt.Errorf("setSubmoduleURL: %w", err)
	}
	// mark submodule as active in local config
	if err := activateSubmodule(repoDir, mod.Name); err != nil {
		return fmt.Errorf("activateSubmodule: %w", err)
	}

	// move progress to 100%
	updateProgress(grid, mod.Path, 100)
	renderProgressGrid(grid)
	return nil
}

func attemptSubmoduleSetups(submodules []Submodule, grid *ProgressGrid, repoDir string, cacheDir string, noCache bool, maxConcurrency int) []SubmoduleFailure {
	// set up a waitGroup to perform a concurrent fanout and join
	var wg sync.WaitGroup
	var mu sync.Mutex

	// record any failures
	var failures []SubmoduleFailure

	// use buffered channel to bound concurrency
	sem := make(chan struct{}, maxConcurrency)

	for _, mod := range submodules {
		// mark that we're waiting on this routine
		wg.Add(1)
		// define the routine
		go func(mod Submodule) {
			// wait for this goroutine to finish
			defer wg.Done()
			// acquire
			sem <- struct{}{}
			// defer release
			defer func() { <-sem }()

			// clone, checkout, stage, and register submodule locally
			err := attemptSubmoduleSetup(mod, grid, repoDir, cacheDir, noCache)

			// record failure if error occurred
			if err != nil {
				mu.Lock()
				failures = append(failures, SubmoduleFailure{mod: mod, err: err})
				mu.Unlock()
			}
		}(mod)
	}
	// block until all goroutines call Done()
	wg.Wait()

	// return SubmoduleFailures
	return failures
}

func cloneSubmodules(repoURL string, repoName string, repoDir string, noCache bool, depth int, maxDepth int, maxRetries int, maxConcurrency int) error {
	// exit early if the maxDepth is exceeded
	if maxDepth != -1 && depth >= maxDepth {
		return nil
	}

	// read submodules from .gitmodules
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

	// initialize progress grid
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

	// print initial progress
	renderProgressGrid(grid)

	// initial run
	failures := attemptSubmoduleSetups(submodules, grid, repoDir, cacheDir, noCache, maxConcurrency)

	// retry loop
	for attempt := 1; attempt <= maxRetries && len(failures) > 0; attempt++ {
		// set up a new grid to hold progress
		grid = &ProgressGrid{
			order:    []string{},
			progress: map[string]int{},
		}
		var retrySubs []Submodule

		// prepare retry list and progress grid
		printLog("\nRetrying %d failed submodule clones (%d/%d)...\n\n", len(failures), attempt, maxRetries)

		// set up grid and submodules for next attempt
		for _, f := range failures {
			grid.order = append(grid.order, f.mod.Path)
			retrySubs = append(retrySubs, f.mod)
		}

		// on subsequent attempts, skip cache and attempt full clone
		failures = attemptSubmoduleSetups(retrySubs, grid, repoDir, cacheDir, true, maxConcurrency)
	}

	// maxRetries exceeded, report final failure
	if len(failures) > 0 {
		printLog("\n")
		for _, f := range failures {
			printError("❌ submodule setup failed for %s: %v\n", f.mod.Path, f.err)
		}
	}

	// recurse into nested submodules
	for _, mod := range submodules {
		nestedPath := filepath.Join(repoDir, mod.Path)
		if shouldVisit(nestedPath) {
			_ = cloneSubmodules(mod.URL, mod.Name, nestedPath, noCache, depth+1, maxDepth, maxRetries, maxConcurrency)
		}
	}

	// don't halt on error; continue cloning remaining submodules
	return nil
}

func (g *GitFetcher) Fetch(templateURL, targetDir string, verbose bool, noCache bool) error {
	// parse the provided templateURL
	repoURL, branch := parseGitHubURL(templateURL)
	printLog("Cloning template repo: %s → %s\n\n", repoURL, targetDir)

	// get name from the repoUrl
	templateName := filepath.Base(strings.TrimSuffix(repoURL, ".git"))

	// set up a new grid to hold progress
	grid := &ProgressGrid{
		order:    []string{templateName},
		progress: map[string]int{templateName: 0},
	}
	renderProgressGrid(grid)

	// get HEAD commit from remote to id the clone we're about to make, fallback to no-cache if we can't get HEAD
	useCache := true
	commitHash, err := getRemoteHEADCommit(repoURL, branch)
	if err != nil {
		useCache = false
		printError("⚠️  Warning: couldn't resolve HEAD commit, falling back to direct clone: %v\n", err)
	}

	// if we can useCache then cloneWithCache
	if useCache {
		cacheDir := filepath.Join(".", ".git-template-cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			return fmt.Errorf("create template cache dir: %w", err)
		}

		// assign cache id as concat of name+commit
		cacheName := fmt.Sprintf("%s-%s.git", templateName, commitHash)
		cachePath := filepath.Join(cacheDir, cacheName)

		// clone into cache if requested and copy to targetDir
		if err := cloneWithCache(repoURL, branch, cachePath, targetDir, noCache, func(pct int) {
			updateProgress(grid, templateName, pct)
			renderProgressGrid(grid)
		}); err != nil {
			return fmt.Errorf("clone with cache failed: %w", err)
		}
	} else {
		// clone fresh directly to the targetDir
		err = runClone(repoURL, branch, []string{"--depth=1", "--recurse-submodules=0"}, targetDir, func(pct int) {
			updateProgress(grid, templateName, pct)
			renderProgressGrid(grid)
		})
		if err != nil {
			return fmt.Errorf("fallback clone failed: %w", err)
		}
	}

	// recurse into submodules
	if err := cloneSubmodules(repoURL, templateName, targetDir, noCache, 0, g.MaxDepth, g.MaxRetries, g.MaxConcurrency); err != nil {
		return fmt.Errorf("submodule clone with cache failed: %w", err)
	}

	// notify that all cloning is done
	printLog("\nClone complete.\n")
	return nil
}
