package template

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

type GitRepoCache interface {
	CacheKey(repoURL, commit string) string
	Get(repoURL, commit string) (path string, ok bool)
	Set(repoURL, commit, sourcePath string) error
}

type execGitRepoCache struct {
	basePath string
}

func NewGitRepoCache(basePath string) *execGitRepoCache {
	return &execGitRepoCache{basePath: basePath}
}

func (c *execGitRepoCache) CacheKey(repoURL, commit string) string {
	h := sha1.New()
	h.Write([]byte(repoURL + ":" + commit))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *execGitRepoCache) Get(repoURL, commit string) (string, bool) {
	key := c.CacheKey(repoURL, commit)
	path := filepath.Join(c.basePath, key)
	if _, err := os.Stat(path); err == nil {
		return path, true
	}
	return "", false
}

func (c *execGitRepoCache) Set(repoURL, commit, sourcePath string) error {
	key := c.CacheKey(repoURL, commit)
	target := filepath.Join(c.basePath, key)
	_ = os.RemoveAll(target)
	return copyDir(sourcePath, target)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		tgt := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(tgt, info.Mode())
		}
		srcF, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcF.Close()

		dstF, err := os.OpenFile(tgt, os.O_CREATE|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer dstF.Close()

		_, err = io.Copy(dstF, srcF)
		return err
	})
}
