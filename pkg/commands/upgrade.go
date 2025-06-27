package commands

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Layr-Labs/devkit-cli/internal/version"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// LatestRelease defines the subset of GitHub release fields we care about
type LatestRelease struct {
	TagName      string `json:"tag_name"`
	TargetCommit string `json:"target_commitish"`
}

// UpgradeCommand defines the CLI command to upgrade the devkit binary and templates
var UpgradeCommand = &cli.Command{
	Name:  "upgrade",
	Usage: "Upgrade devkit and template",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "version",
			Usage: "Version to upgrade to (e.g. v0.0.8)",
			Value: "latest",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		return UpgradeDevkit(cCtx)
	},
}

// buildDownloadURL generates the download URL for a specific version and platform
var buildDownloadURL = func(version, arch, distro string) string {
	return "https://s3.amazonaws.com/eigenlayer-devkit-releases/" +
		version + "/devkit-" + distro + "-" + arch + "-" + version + ".tar.gz"
}

// githubReleasesURL points to the latest GitHub release metadata
var githubReleasesURL = func(version string) string {
	baseUrl := "https://api.github.com/repos/Layr-Labs/devkit-cli/releases/"
	if version == "latest" {
		return baseUrl + version
	} else {
		return baseUrl + "tags/" + version
	}
}

// UpgradeDevkit resolves the latest version if needed and invokes PerformUpgrade to install the new version
func UpgradeDevkit(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Get current version
	currentVersion := version.GetVersion()
	currentCommit := version.GetCommit()

	// Get the version to be installed
	requestedVersion := cCtx.String("version")
	// Default requestedVersion to "latest"
	if requestedVersion == "" {
		requestedVersion = "latest"
	}

	// Pull release details from github
	targetVersion, targetCommit, err := GetLatestVersionFromGitHub(requestedVersion)
	if err != nil {
		return fmt.Errorf("requested version %s does not exist: %w", requestedVersion, err)
	}

	// Log upgrade
	logger.Info("Upgrading devkit from %s (%s) to %s (%s)...", currentVersion, currentCommit, targetVersion, targetCommit[:7])

	// Avoid redundant upgrade
	if currentVersion == targetVersion && currentCommit == targetCommit[:7] {
		return fmt.Errorf("already on latest version: %s commit: %s", currentVersion, currentCommit)
	}

	// Determine install location
	path, err := exec.LookPath("devkit")
	if err != nil {
		return fmt.Errorf("could not locate current devkit binary: %w", err)
	}
	binDir := filepath.Dir(path)

	// Perform the upgrade and source
	return PerformUpgrade(targetVersion, binDir, logger)
}

// PerformUpgrade downloads and installs the target version of the devkit binary.
// It supports both .tar.gz and raw .tar archive formats.
func PerformUpgrade(version, binDir string, logger iface.Logger) error {
	arch := strings.ToLower(runtime.GOARCH)
	distro := strings.ToLower(runtime.GOOS)

	url := buildDownloadURL(version, arch, distro)

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin dir: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response from server: %s", resp.Status)
	}

	// Detect format by content type and initialize appropriate tar reader
	var tr *tar.Reader
	contentType := resp.Header.Get("Content-Type")
	switch {
	case strings.Contains(contentType, "gzip"):
		gzr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("gzip parse error: %w", err)
		}
		defer gzr.Close()
		tr = tar.NewReader(gzr)

	case strings.Contains(contentType, "x-tar"), strings.Contains(contentType, "application/octet-stream"):
		tr = tar.NewReader(resp.Body)

	default:
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected content type: %s\nBody: %s", contentType, string(body))
	}

	// Extract all files from the archive into the binDir
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tarball: %w", err)
		}

		targetPath := filepath.Join(binDir, filepath.Base(hdr.Name))
		outFile, err := os.Create(targetPath)
		if err != nil {
			return fmt.Errorf("error creating file: %w", err)
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return fmt.Errorf("error writing file: %w", err)
		}
		outFile.Close()

		if err := os.Chmod(targetPath, 0755); err != nil {
			return fmt.Errorf("error setting permissions: %w", err)
		}
		logger.Info("Installed: %s", targetPath)
	}

	logger.Info("Upgrade complete.")
	return nil
}

// GetLatestVersionFromGitHub queries the GitHub releases API and returns the latest tag and commit
func GetLatestVersionFromGitHub(version string) (string, string, error) {
	resp, err := http.Get(githubReleasesURL(version))
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("failed to fetch release for version %s: %s", version, resp.Status)
	}

	var data LatestRelease
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	if data.TagName == "" {
		return "", "", fmt.Errorf("no tag_name in GitHub response")
	}

	if data.TargetCommit == "" {
		return "", "", fmt.Errorf("no target_commitish in GitHub response")
	}

	return data.TagName, data.TargetCommit, nil
}
