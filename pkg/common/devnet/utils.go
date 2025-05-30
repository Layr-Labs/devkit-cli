package devnet

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
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
	logger := common.LoggerFromContext(ctx.Context)

	if err := exec.CommandContext(ctx.Context, "docker", "stop", containerName).Run(); err != nil {
		logger.Error("⚠️  Failed to stop container %s: %v", containerName, err)
	} else {
		logger.Info("✅ Stopped container %s", containerName)
	}
	if err := exec.CommandContext(ctx.Context, "docker", "rm", containerName).Run(); err != nil {
		logger.Error("⚠️  Failed to remove container %s: %v", containerName, err)
	} else {
		logger.Info("✅ Removed container %s", containerName)
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
// - Linux: defaults to 172.17.0.1 (Docker containers can access host via localhost)
// - macOS/Windows: defaults to host.docker.internal (required for Docker Desktop)
func GetDockerHost() string {
	if dockersHost := os.Getenv("DOCKERS_HOST"); dockersHost != "" {
		return dockersHost
	}

	// Detect OS and set appropriate default
	if runtime.GOOS == "linux" {
		return "172.17.0.1"
	} else {
		return "host.docker.internal"
	}
}

// EnsureDockerHost replaces localhost/127.0.0.1 in URLs with the appropriate Docker host.
// Only replaces when localhost/127.0.0.1 are the actual hostname, not substrings.
// This ensures URLs work correctly when passed to Docker containers across platforms.
func EnsureDockerHost(inputUrl string) string {
	dockerHost := GetDockerHost()

	// Handle edge cases first: bare localhost/127.0.0.1 strings
	trimmed := strings.TrimSpace(inputUrl)
	if trimmed == "localhost" || trimmed == "127.0.0.1" {
		return dockerHost
	}

	// Parse the URL to work with components safely
	parsedUrl, err := url.Parse(inputUrl)
	if err != nil {
		// If URL parsing fails, fall back to regex-based replacement
		return ensureDockerHostRegex(inputUrl, dockerHost)
	}

	// Extract hostname (without port)
	hostname := parsedUrl.Hostname()

	// Handle the case where URL parsing succeeded but hostname is empty
	// This happens with strings like "localhost:8545" (parsed as scheme:opaque)
	if hostname == "" {
		// Check if the scheme is localhost or 127.0.0.1 (meaning it was parsed as scheme:opaque)
		if parsedUrl.Scheme == "localhost" || parsedUrl.Scheme == "127.0.0.1" {
			// Reconstruct as host:port format
			if parsedUrl.Opaque != "" {
				return fmt.Sprintf("%s:%s", dockerHost, parsedUrl.Opaque)
			} else {
				return dockerHost
			}
		}
		// If hostname is empty but it's not the scheme:opaque case, fall back to regex
		return ensureDockerHostRegex(inputUrl, dockerHost)
	}

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
	// Pattern to match URLs with schemes (http, https, ws, wss) followed by localhost
	// This ensures we only rewrite actual localhost URLs, not subdomains like "api.localhost.company.com"
	schemeLocalhostPattern := regexp.MustCompile(`(https?|wss?)://localhost(:[0-9]+)?(/\S*)?`)
	schemeIPPattern := regexp.MustCompile(`(https?|wss?)://127\.0\.0\.1(:[0-9]+)?(/\S*)?`)

	// Pattern to match malformed scheme-like strings with localhost/127.0.0.1
	// This handles cases like "ht tp://localhost" or "ht\x00tp://localhost"
	malformedSchemeLocalhostPattern := regexp.MustCompile(`\S*tp://localhost(:[0-9]+)?(/\S*)?`)
	malformedSchemeIPPattern := regexp.MustCompile(`\S*tp://127\.0\.0\.1(:[0-9]+)?(/\S*)?`)

	// Pattern to match standalone localhost (no scheme) at start of string or after whitespace/equals
	// This avoids matching localhost as part of a larger domain name
	standaloneLocalhostPattern := regexp.MustCompile(`(?:^|[\s=])localhost(:[0-9]+)?(?:[\s/=?#]|$)`)
	standaloneIPPattern := regexp.MustCompile(`(?:^|[\s=])127\.0\.0\.1(:[0-9]+)?(?:[\s/=?#]|$)`)

	result := inputUrl

	// Replace scheme-based localhost URLs
	result = schemeLocalhostPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "localhost", dockerHost, 1)
	})

	// Replace scheme-based 127.0.0.1 URLs
	result = schemeIPPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "127.0.0.1", dockerHost, 1)
	})

	// Replace malformed scheme localhost patterns
	result = malformedSchemeLocalhostPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "localhost", dockerHost, 1)
	})

	// Replace malformed scheme 127.0.0.1 patterns
	result = malformedSchemeIPPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "127.0.0.1", dockerHost, 1)
	})

	// Replace standalone localhost patterns
	result = standaloneLocalhostPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "localhost", dockerHost, 1)
	})

	// Replace standalone 127.0.0.1 patterns
	result = standaloneIPPattern.ReplaceAllStringFunc(result, func(match string) string {
		return strings.Replace(match, "127.0.0.1", dockerHost, 1)
	})

	return result
}

// GetRPCURL returns the RPC URL for accessing the devnet container from the host.
// This should always use localhost since it's for host→container communication
func GetRPCURL(port int) string {
	return fmt.Sprintf("http://localhost:%d", port)
}
