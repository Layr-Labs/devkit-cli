package devnet

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

// IsPortAvailable checks if a TCP port is not already bound by another service.
func IsPortAvailable(port int) bool {
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		// If dialing fails, port is likely available
		return true
	}
	_ = conn.Close()
	return false
}

// / Stops the container and removes it
func StopAndRemoveContainer(ctx *cli.Context, containerName string) {
	if err := exec.CommandContext(ctx.Context, "docker", "stop", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to stop container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Stopped container %s", containerName)
	}
	if err := exec.CommandContext(ctx.Context, "docker", "rm", containerName).Run(); err != nil {
		log.Printf("⚠️ Failed to remove container %s: %v", containerName, err)
	} else {
		log.Printf("✅ Removed container %s", containerName)
	}
}

// GetDockerPsDevnetArgs returns the arguments needed to list all running
// devkit devnet Docker containers along with their exposed ports.
// It filters containers by name prefix ("devkit-devnet") and formats
// the output to show container name and port mappings in a readable form.
func GetDockerPsDevnetArgs() []string {
	return []string{
		"ps",
		"--filter", "name=devkit-devnet",
		"--format", "{{.Names}}: {{.Ports}}",
	}
}

// GetDockerHost returns the appropriate Docker host based on environment and platform.
// Uses DOCKERS_HOST environment variable if set, otherwise detects OS:
// - Linux: defaults to localhost (Docker containers can access host via localhost)
// - macOS/Windows: defaults to host.docker.internal (required for Docker Desktop)
func GetDockerHost() string {
	if dockersHost := os.Getenv("DOCKERS_HOST"); dockersHost != "" {
		return dockersHost
	}

	// Detect OS and set appropriate default
	if runtime.GOOS == "linux" {
		return "localhost"
	} else {
		return "host.docker.internal"
	}
}

// EnsureDockerHost replaces localhost/127.0.0.1 in URLs with the appropriate Docker host.
// Only replaces when localhost/127.0.0.1 are the actual hostname, not substrings.
// This ensures URLs work correctly when passed to Docker containers across platforms.
func EnsureDockerHost(inputUrl string) string {
	dockerHost := GetDockerHost()

	// Parse the URL to work with components safely
	parsedUrl, err := url.Parse(inputUrl)
	if err != nil {
		// If URL parsing fails, fall back to regex-based replacement
		return ensureDockerHostRegex(inputUrl, dockerHost)
	}

	// Extract hostname (without port)
	hostname := parsedUrl.Hostname()

	// Only replace if hostname is exactly localhost or 127.0.0.1
	if hostname == "localhost" || hostname == "127.0.0.1" {
		// Replace just the hostname part
		if parsedUrl.Port() != "" {
			parsedUrl.Host = fmt.Sprintf("%s:%s", dockerHost, parsedUrl.Port())
		} else {
			parsedUrl.Host = dockerHost
		}
		return parsedUrl.String()
	}

	// Return original URL if hostname doesn't match
	return inputUrl
}

// ensureDockerHostRegex provides regex-based fallback for malformed URLs
func ensureDockerHostRegex(inputUrl string, dockerHost string) string {
	// Pattern to match localhost or 127.0.0.1 as hostname (not substring)
	// Matches: localhost:8545, localhost/, localhost, 127.0.0.1:8545, etc.
	// Doesn't match: my-localhost.com, localhost.domain.com, etc.
	localhostPattern := regexp.MustCompile(`\blocalhost(:[0-9]+)?(/|$|\?)`)
	ipPattern := regexp.MustCompile(`\b127\.0\.0\.1(:[0-9]+)?(/|$|\?)`)

	// Replace localhost patterns
	result := localhostPattern.ReplaceAllStringFunc(inputUrl, func(match string) string {
		return strings.Replace(match, "localhost", dockerHost, 1)
	})

	// Replace 127.0.0.1 patterns
	result = ipPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "127.0.0.1", dockerHost, 1)
	})

	return result
}

// GetRPCURL returns the RPC URL for accessing the devnet container from the host.
// Always uses localhost since Docker maps container ports to localhost on all platforms.
func GetRPCURL(port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}
