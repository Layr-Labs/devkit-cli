package devnet

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/ethereum/go-ethereum/rpc"
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
func StopAndRemoveContainer(cCtx *cli.Context, containerName string) {
	logger := common.LoggerFromContext(cCtx)

	if err := exec.CommandContext(cCtx.Context, "docker", "stop", containerName).Run(); err != nil {
		logger.Error("⚠️  Failed to stop container %s: %v", containerName, err)
	} else {
		logger.Info("✅ Stopped container %s", containerName)
	}
	if err := exec.CommandContext(cCtx.Context, "docker", "rm", containerName).Run(); err != nil {
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

// GetL2BlockByNumber retrieves the timestamp of a specific L2 block by its number.
func GetL2BlockByNumber(ctx *cli.Context, l2RpcUrl string, blockNumber uint64, logger iface.Logger) (string, error) {
	rpcClient, err := rpc.DialContext(context.Background(), l2RpcUrl)
	if err != nil {
		return "", fmt.Errorf("failed to dial RPC: %v", err)
	}
	defer rpcClient.Close()

	var blockResult map[string]interface{}
	err = rpcClient.CallContext(context.Background(), &blockResult, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return "", fmt.Errorf("failed to call eth_getBlockByNumber: %v", err)
	}

	timestampHex, ok := blockResult["timestamp"].(string)
	if !ok {
		return "", fmt.Errorf("failed to get timestamp from block")
	}

	// Convert hex timestamp to decimal string
	if len(timestampHex) < 2 || timestampHex[:2] != "0x" {
		return "", fmt.Errorf("invalid timestamp hex format: %s", timestampHex)
	}

	timestampInt, err := strconv.ParseUint(timestampHex[2:], 16, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp hex: %v", err)
	}

	return strconv.FormatUint(timestampInt, 10), nil
}

func AdvanceBlocks(ctx *cli.Context, l1RpcUrl string, numBlocks uint64) error {
	// Connect to provided client
	rpcClient, err := rpc.Dial(l1RpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer rpcClient.Close()

	// Advance numBlocks blocks on client
	err = rpcClient.Call(nil, "anvil_mine", numBlocks)
	if err != nil {
		return fmt.Errorf("failed to advance blocks: %w", err)
	}

	return nil
}

func AdvanceBlocksToTS(client *rpc.Client, name string, fromTS, toTS uint64) error {
	// Set number of blocks to move each iteration
	const blocksPerBatch = 1

	// While timestamps are out of sync mine blocksPerBatch at a time
	for fromTS < toTS {
		if err := client.Call(nil, "anvil_mine", blocksPerBatch); err != nil {
			return fmt.Errorf("failed to mine on %s: %w", name, err)
		}
		newTS, err := GetTimestamp(client, name)
		if err != nil {
			return err
		}
		fromTS = newTS
	}
	return nil
}

func GetTimestamp(client *rpc.Client, name string) (uint64, error) {
	// Collect the block height for provided client
	var block map[string]interface{}
	if err := client.Call(&block, "eth_getBlockByNumber", "latest", false); err != nil {
		return 0, fmt.Errorf("failed to get latest %s block: %w", name, err)
	}
	tsHex, ok := block["timestamp"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid timestamp format for %s", name)
	}
	return strconv.ParseUint(tsHex[2:], 16, 64)
}

func SyncL1L2Timestamps(ctx *cli.Context, l1RpcUrl string, l2RpcUrl string) error {
	// Connect to l1
	l1Client, err := rpc.Dial(l1RpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L1: %w", err)
	}
	defer l1Client.Close()

	// Connect to l2
	l2Client, err := rpc.Dial(l2RpcUrl)
	if err != nil {
		return fmt.Errorf("failed to connect to L2: %w", err)
	}
	defer l2Client.Close()

	// Get l1 and l2 current timestamps
	l1TS, err := GetTimestamp(l1Client, "L1")
	if err != nil {
		return err
	}
	l2TS, err := GetTimestamp(l2Client, "L2")
	if err != nil {
		return err
	}

	// Advance one or the other until we reach sync
	if l1TS > l2TS {
		return AdvanceBlocksToTS(l2Client, "L2", l2TS, l1TS)
	} else if l2TS > l1TS {
		return AdvanceBlocksToTS(l1Client, "L1", l1TS, l2TS)
	}

	// Already in sync
	return nil
}
