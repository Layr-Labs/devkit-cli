package template

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type GitFetcher struct {
	Git     GitClient
	Cache   GitRepoCache
	Logger  logger.ProgressLogger
	Metrics GitMetrics
	Config  GitFetcherConfig
}

type GitFetcherConfig struct {
	MaxDepth       int
	MaxRetries     int
	MaxConcurrency int
	CacheDir       string
	UseCache       bool
	Verbose        bool
}

// TODO: implement metric transport
type GitMetrics interface {
	SubmoduleCloneStarted(name string)
	SubmoduleCloneFinished(name string, err error)
}

func (g *GitFetcher) Fetch(ctx context.Context, repoBaseUrl, ref, targetDir string) error {
	if ref == "" {
		return fmt.Errorf("ref is required")
	}
	// parse GitHub URL to extract repo URL and ref
	templateName := filepath.Base(strings.TrimSuffix(repoBaseUrl, ".git"))
	g.Logger.Info("Cloning repo: %s → %s\n\n", repoBaseUrl, targetDir)

	// resolve commit (get HEAD commit or a specific one based on the ref)
	commit, err := g.Git.ResolveRemoteCommit(ctx, repoBaseUrl, ref)
	if err != nil {
		g.Logger.Warn("Could not resolve remote commit", "error", err)
		commit = "HEAD"
	}

	// try fetching the main repository
	fromCache, err := g.fetchMainRepo(ctx, repoBaseUrl, ref, commit, templateName, targetDir)
	if err != nil {
		return err
	}

	// if not fetched from cache, proceed with fetching submodules
	if !fromCache {
		return g.fetchSubmodules(ctx, templateName, repoBaseUrl, targetDir, 0)
	}

	// clone of template complete
	g.Logger.Info("Clone repo complete: %s\n\n", repoBaseUrl)

	return nil
}

func (g *GitFetcher) fetchMainRepo(ctx context.Context, repoURL, branch, commit, templateName, targetDir string) (bool, error) {
	// define cache paths
	cacheDir := g.Config.CacheDir
	cacheKey := g.Cache.CacheKey(repoURL, commit)
	cachePath := filepath.Join(cacheDir, cacheKey)

	// if cache is missing or UseCache is false, perform a bare clone into the cache
	if g.Config.UseCache && commit != "HEAD" {
		if _, ok := g.Cache.Get(repoURL, commit); !ok {
			// call Clone with progress tracking
			err := g.Git.RetryClone(ctx, repoURL, cachePath, CloneOptions{
				Branch: branch,
				Depth:  1,
				Bare:   true,
				ProgressCB: func(p int) {
					g.Logger.SetProgress(cachePath, p, templateName)
					g.Logger.PrintProgress()
				},
			}, g.Config.MaxRetries)

			// if we failed after all attempts log error
			if err != nil {
				return false, fmt.Errorf("failed to clone into cache: %w", err)
			}
		}
		// move repoUrl to cachePath
		repoURL = cachePath
	}

	// call Clone to copy cached repo to targetDir
	err := g.Git.Clone(ctx, repoURL, targetDir, CloneOptions{
		Branch:      branch,
		Depth:       1,
		Dissociate:  true,
		NoHardlinks: true,
		ProgressCB: func(p int) {
			g.Logger.SetProgress(cachePath, p, templateName)
			g.Logger.PrintProgress()
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to clone into cache: %w", err)
	}

	// set progress to complete in logger if we cloned fresh
	g.Logger.SetProgress(cachePath, 100, templateName)
	g.Logger.PrintProgress()

	// clear progress reporting
	g.Logger.ClearProgress()

	// always process submodules after cloning or copying from cache
	if err := g.fetchSubmodules(ctx, templateName, repoURL, targetDir, 0); err != nil {
		return false, err
	}

	return true, nil
}

func (g *GitFetcher) fetchSubmodules(ctx context.Context, repoName string, repoURL string, repoDir string, depth int) error {
	// if no submodules file exists, skip submodule fetching and continue
	if _, err := os.Stat(filepath.Join(repoDir, ".gitmodules")); os.IsNotExist(err) {
		return nil
	}

	// list submodules defined in .gitmodules
	submodules, err := g.Git.SubmoduleList(ctx, repoDir)
	if err != nil {
		g.Logger.Warn("Failed to list submodules", "repo", repoDir, "error", err)
		return nil
	}

	// if we're at or beyond max-depth end early
	if g.Config.MaxDepth != -1 && depth >= g.Config.MaxDepth {
		return nil
	}

	// print discoveries
	g.Logger.Info("Discovered submodules in %s (%s):", repoName, repoURL)
	for _, mod := range submodules {
		g.Logger.Info(" - %s → %s (%s)\n", mod.Name, mod.Path, mod.URL)
		g.Logger.SetProgress(mod.Path, 0, mod.Path)
	}
	g.Logger.Info("")

	// define cache paths
	cacheDir := g.Config.CacheDir

	// attempt to clone all submodules and collect failures
	failures := g.cloneSubmodules(ctx, submodules, repoDir, cacheDir)

	// retry loop upto a max of g.Config.MaxRetries, stop when no failures occur
	for attempt := 1; attempt <= g.Config.MaxRetries && len(failures) > 0; attempt++ {
		// construct new submodules to retry
		var retrySubs []Submodule

		// prepare retry list and progress grid
		g.Logger.Info("Retrying %d failed submodule clones (%d/%d)...\n\n", len(failures), attempt, g.Config.MaxRetries)

		// set up grid and submodules for next attempt
		for _, f := range failures {
			g.Logger.SetProgress(f.mod.Path, 0, f.mod.Path)
			retrySubs = append(retrySubs, f.mod)
		}

		// on subsequent attempts, skip cache and attempt full clone
		failures = g.cloneSubmodules(ctx, retrySubs, repoDir, cacheDir)
	}

	// maxRetries exceeded, report final failure
	if len(failures) > 0 {
		for _, f := range failures {
			g.Logger.Error("❌ submodule setup failed for %s: %v\n", f.mod.Path, f.err)
		}
	}

	// recurse into nested submodules
	for _, mod := range submodules {
		subdir := filepath.Join(repoDir, mod.Path)
		_ = g.fetchSubmodules(ctx, mod.Name, mod.URL, subdir, depth+1)
	}

	return nil
}

func (g *GitFetcher) cloneSubmodules(
	ctx context.Context,
	submodules []Submodule,
	repoDir string,
	cacheDir string,
) []SubmoduleFailure {
	// record any failures
	var failures []SubmoduleFailure

	// use buffered channel to bound concurrency
	sem := make(chan struct{}, g.Config.MaxConcurrency)

	// foreach submodule, clone and register
	var mu sync.Mutex
	var wg sync.WaitGroup
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

			// ref location we're pointing to in template dir
			targetDir := filepath.Join(repoDir, mod.Path)

			// start submodule cloning and tracking progress
			if g.Metrics != nil {
				g.Metrics.SubmoduleCloneStarted(mod.Path)
			}

			// fetch the commit from parent repos .gitmodules
			commit, err := g.Git.SubmoduleCommit(ctx, repoDir, mod.Path)
			if err != nil {
				g.Logger.Warn("Failed to get submodule commit", "path", mod.Path, "error", err)
				return
			}

			// set submoduleUrl to current modUrl
			submoduleUrl := mod.URL
			cloneOpts := CloneOptions{
				ProgressCB: func(p int) {
					g.Logger.SetProgress(mod.Path, p, mod.Path)
					g.Logger.PrintProgress()
				},
			}

			// get cache location
			cacheKey := g.Cache.CacheKey(submoduleUrl, commit)
			cachePath := filepath.Join(cacheDir, cacheKey)

			// set/get from cache if enabled...
			if g.Config.UseCache {
				// call RetryClone with progress tracking
				if _, ok := g.Cache.Get(submoduleUrl, commit); !ok {
					err = g.Git.RetryClone(ctx, submoduleUrl, cachePath, CloneOptions{
						Bare: true,
						ProgressCB: func(p int) {
							g.Logger.SetProgress(mod.Path, p, mod.Path)
							g.Logger.PrintProgress()
						},
					}, g.Config.MaxRetries)

					// if we failed after all attempts log error
					if err != nil {
						g.Logger.Error("Failed to clone submodule", "path", mod.Path, "error", err)
						return
					}
				}

				// replace submoduleUrl with cachePath
				submoduleUrl = cachePath
			}

			// clone from cache/submoduleUrl to target with retries
			err = g.Git.SubmoduleClone(ctx, mod, commit, submoduleUrl, targetDir, repoDir, cloneOpts)

			// record failures
			if err != nil {
				mu.Lock()
				// append original failure
				failures = append(failures, SubmoduleFailure{mod: mod, err: err})
				// if err is for a clone failure, we should reattempt the original clone
				if strings.Contains(err.Error(), "clone failed") {
					// clean up cache target
					_ = os.RemoveAll(cachePath)
					// clone to cache location as bare
					if err := g.Git.RetryClone(ctx, mod.URL, cachePath, CloneOptions{
						Bare:       true,
						ProgressCB: cloneOpts.ProgressCB,
					}, g.Config.MaxRetries); err != nil {
						failures = append(failures, SubmoduleFailure{mod: mod, err: err})
					}
				}
				mu.Unlock()
			}

			// report in metrics
			if g.Metrics != nil {
				g.Metrics.SubmoduleCloneFinished(mod.Path, err)
			}

			// set progress to complete in logger
			g.Logger.SetProgress(mod.Path, 100, mod.Path)
			g.Logger.PrintProgress()
		}(mod)
	}
	wg.Wait()

	// clear progress reporting
	g.Logger.ClearProgress()

	return failures
}
