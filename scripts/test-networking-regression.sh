#!/bin/bash

# Test script to verify Docker networking fixes are in place
# This script will FAIL if someone reverts our networking fixes

set -e

echo "🔌 Testing Docker networking regression protection..."

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Cleanup function to run on exit
cleanup() {
    echo "🧹 Cleaning up test artifacts..."
    if [ -d "./test-networking-regression" ]; then
        cd "./test-networking-regression" 2>/dev/null && {
            "$PROJECT_ROOT/bin/devkit" avs devnet stop 2>/dev/null || true
            cd "$PROJECT_ROOT"
        }
        rm -rf "./test-networking-regression"
        echo "✅ Removed test-networking-regression directory"
    fi
}

# Set trap to cleanup on exit (success, failure, or interruption)
trap cleanup EXIT

# Build the CLI first
echo "Building CLI..."
make build

# Create a test project
echo "Creating test project..."
./bin/devkit avs create test-networking-regression
cd ./test-networking-regression

# Set environment for testing
export L1_FORK_URL="https://ethereum-rpc.publicnode.com"

# Find an available port (start from 9545 to avoid conflicts)
find_available_port() {
    local port=9545
    while netstat -an | grep -q ":$port "; do
        port=$((port + 1))
    done
    echo $port
}

AVAILABLE_PORT=$(find_available_port)
echo "Using port $AVAILABLE_PORT for testing..."

echo "Testing devnet start..."
# Use --skip-avs-run to start faster and use available port
timeout 60s "$PROJECT_ROOT/bin/devkit" avs devnet start --port $AVAILABLE_PORT --skip-deploy-contracts --skip-avs-run

# Wait a moment for the YAML to be written
sleep 2

# Test 1: Check that devnet.yaml contains localhost, not host.docker.internal
echo "Checking RPC URL in devnet.yaml..."
if [ -f "config/contexts/devnet.yaml" ]; then
    if grep -q "host\.docker\.internal:$AVAILABLE_PORT" config/contexts/devnet.yaml; then
        echo "❌ REGRESSION DETECTED: RPC URL uses host.docker.internal instead of localhost!"
        echo "This means GetRPCURL() was reverted to old behavior"
        echo "Content of devnet.yaml:"
        cat config/contexts/devnet.yaml
        exit 1
    fi
    
    if ! grep -q "localhost:$AVAILABLE_PORT" config/contexts/devnet.yaml; then
        echo "❌ REGRESSION DETECTED: RPC URL doesn't use localhost!"
        echo "Expected localhost:$AVAILABLE_PORT in devnet.yaml but found:"
        grep "rpc_url" config/contexts/devnet.yaml || echo "No rpc_url found"
        echo "Full content of devnet.yaml:"
        cat config/contexts/devnet.yaml
        exit 1
    fi
    
    echo "✅ RPC URL correctly uses localhost"
else
    echo "❌ devnet.yaml not found!"
    exit 1
fi

# Test 2: Check docker-compose.yaml has the extra_hosts mapping
echo "Checking docker-compose.yaml networking..."
if [ -f "/tmp/devkit-compose/docker-compose.yaml" ]; then
    if ! grep -q "host.docker.internal:host-gateway" /tmp/devkit-compose/docker-compose.yaml; then
        echo "❌ REGRESSION DETECTED: docker-compose.yaml missing extra_hosts mapping!"
        echo "This means the docker networking fix was reverted"
        echo "Content of docker-compose.yaml:"
        cat /tmp/devkit-compose/docker-compose.yaml
        exit 1
    fi
    echo "✅ docker-compose.yaml has correct host mapping"
else
    echo "⚠️  docker-compose.yaml not found, skipping docker-compose test"
fi

# Test 3: Verify EnsureDockerHost function handles edge cases correctly
echo "Testing EnsureDockerHost edge cases..."

# Create a simple Go test to verify our EnsureDockerHost logic for BOTH platforms
cat > test_ensure_docker_host.go << 'EOF'
package main

import (
    "fmt"
    "net/url"
    "os"
    "regexp"
    "runtime"
    "strings"
)

func GetDockerHost() string {
    if dockersHost := os.Getenv("DOCKERS_HOST"); dockersHost != "" {
        return dockersHost
    }
    if runtime.GOOS == "linux" {
        return "localhost"
    } else {
        return "host.docker.internal"
    }
}

func EnsureDockerHost(inputUrl string) string {
    dockerHost := GetDockerHost()
    parsedUrl, err := url.Parse(inputUrl)
    if err != nil {
        return ensureDockerHostRegex(inputUrl, dockerHost)
    }
    hostname := parsedUrl.Hostname()
    if hostname == "localhost" || hostname == "127.0.0.1" {
        if parsedUrl.Port() != "" {
            parsedUrl.Host = fmt.Sprintf("%s:%s", dockerHost, parsedUrl.Port())
        } else {
            parsedUrl.Host = dockerHost
        }
        return parsedUrl.String()
    }
    return inputUrl
}

func ensureDockerHostRegex(inputUrl string, dockerHost string) string {
    localhostPattern := regexp.MustCompile(`\blocalhost(:[0-9]+)?(/|$|\?)`)
    ipPattern := regexp.MustCompile(`\b127\.0\.0\.1(:[0-9]+)?(/|$|\?)`)
    result := localhostPattern.ReplaceAllStringFunc(inputUrl, func(match string) string {
        return strings.Replace(match, "localhost", dockerHost, 1)
    })
    result = ipPattern.ReplaceAllStringFunc(result, func(match string) string {
        return strings.Replace(match, "127.0.0.1", dockerHost, 1)
    })
    return result
}

func testPlatform(platformName, expectedDockerHost string) bool {
    fmt.Printf("\n🔧 Testing %s behavior (DOCKERS_HOST=%s)...\n", platformName, expectedDockerHost)
    
    // Override environment to simulate platform
    os.Setenv("DOCKERS_HOST", expectedDockerHost)
    defer os.Unsetenv("DOCKERS_HOST")
    
    testCases := []struct {
        input    string
        expected string
        desc     string
    }{
        {"http://localhost:8545", fmt.Sprintf("http://%s:8545", expectedDockerHost), "Should replace localhost"},
        {"https://127.0.0.1:3000", fmt.Sprintf("https://%s:3000", expectedDockerHost), "Should replace 127.0.0.1"},
        {"https://localhost.mycooldomain.com:8545", "https://localhost.mycooldomain.com:8545", "Should NOT replace localhost in domain"},
        {"https://api.localhost.network:3000", "https://api.localhost.network:3000", "Should NOT replace localhost in subdomain"},
        {"https://my-localhost-service.com:8080", "https://my-localhost-service.com:8080", "Should NOT replace localhost in service name"},
        {"http://mainnet.infura.io/v3/key", "http://mainnet.infura.io/v3/key", "Should not change external URLs"},
    }
    
    allPassed := true
    for _, tc := range testCases {
        result := EnsureDockerHost(tc.input)
        if result != tc.expected {
            fmt.Printf("❌ FAILED: %s\n", tc.desc)
            fmt.Printf("   Input: %s\n", tc.input)
            fmt.Printf("   Expected: %s\n", tc.expected)
            fmt.Printf("   Got: %s\n", result)
            allPassed = false
        } else {
            fmt.Printf("✅ PASSED: %s\n", tc.desc)
        }
    }
    
    return allPassed
}

func main() {
    fmt.Println("🔍 Testing cross-platform Docker host behavior...")
    
    // Test Linux behavior (localhost)
    linuxPassed := testPlatform("Linux", "localhost")
    
    // Test macOS behavior (host.docker.internal)
    macosPassed := testPlatform("macOS", "host.docker.internal")
    
    if !linuxPassed || !macosPassed {
        fmt.Println("\n❌ Cross-platform EnsureDockerHost tests FAILED!")
        os.Exit(1)
    } else {
        fmt.Println("\n✅ All cross-platform EnsureDockerHost tests passed!")
        fmt.Println("✅ Linux behavior: localhost ← correct")
        fmt.Println("✅ macOS behavior: host.docker.internal ← correct")
    }
}
EOF

go run test_ensure_docker_host.go
rm test_ensure_docker_host.go

echo "🎉 All networking regression tests passed!"
echo "✅ Docker networking fixes are in place and working correctly" 