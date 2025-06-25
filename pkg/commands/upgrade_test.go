package commands

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/stretchr/testify/assert"
)

func TestUpgrade_PerformUpgrade(t *testing.T) {
	// Create a fake tar.gz containing a single dummy binary file
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	content := []byte("#!/bin/sh\necho devkit upgraded\n")
	hdr := &tar.Header{
		Name: "devkit",
		Mode: 0755,
		Size: int64(len(content)),
	}
	err := tw.WriteHeader(hdr)
	assert.NoError(t, err)
	_, err = tw.Write(content)
	assert.NoError(t, err)
	tw.Close()
	gz.Close()

	// Start a test server that returns the tarball
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(buf.Bytes())
	}))
	defer ts.Close()

	// Patch the URL builder temporarily (for testing)
	oldURLBuilder := buildDownloadURL
	buildDownloadURL = func(version, arch, distro string) string {
		return ts.URL // fake URL instead of real S3
	}
	defer func() { buildDownloadURL = oldURLBuilder }()

	tmpDir := t.TempDir()
	log := logger.NewNoopLogger()

	err = PerformUpgrade("v0.0.1", tmpDir, log)
	assert.NoError(t, err)

	files, err := os.ReadDir(tmpDir)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "devkit", files[0].Name())

	path := filepath.Join(tmpDir, "devkit")
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(data), "echo devkit upgraded")
}

func TestUpgrade_GetLatestVersionFromGitHub(t *testing.T) {
	// Fake GitHub API server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/Layr-Labs/devkit-cli/releases/latest", r.URL.Path)
		_, _ = w.Write([]byte(`{"tag_name": "v9.9.9", "target_commitish": "aaaaaaa"}`))
	}))
	defer ts.Close()

	// Patch URL to use mock server
	original := githubReleasesURL
	githubReleasesURL = func(version string) string {
		return ts.URL + "/repos/Layr-Labs/devkit-cli/releases/latest"
	}
	defer func() { githubReleasesURL = original }()

	version, commit, err := GetLatestVersionFromGitHub("latest")
	assert.NoError(t, err)
	assert.Equal(t, "v9.9.9", version)
	assert.Equal(t, "aaaaaaa", commit)
}
