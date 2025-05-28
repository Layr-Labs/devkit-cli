package devnet

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetDockerHost tests the GetDockerHost function for different platforms and environment variables
func TestGetDockerHost(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	tests := []struct {
		name        string
		dockersHost string
		expected    string
	}{
		{
			name:        "Custom DOCKERS_HOST environment variable",
			dockersHost: "custom.docker.host",
			expected:    "custom.docker.host",
		},
		{
			name:        "Empty DOCKERS_HOST should fallback to platform default",
			dockersHost: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.dockersHost != "" {
				os.Setenv("DOCKERS_HOST", tt.dockersHost)
			} else {
				os.Unsetenv("DOCKERS_HOST")
			}

			result := GetDockerHost()

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				// When DOCKERS_HOST is empty, should return platform-specific default
				assert.Contains(t, []string{"172.17.0.1", "host.docker.internal"}, result)
			}
		})
	}
}

// TestEnsureDockerHost tests the EnsureDockerHost function with various URL patterns
func TestEnsureDockerHost(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	tests := []struct {
		name        string
		inputURL    string
		dockersHost string
		expectedURL string
		description string
	}{
		{
			name:        "Replace localhost with custom host",
			inputURL:    "http://localhost:8545",
			dockersHost: "custom.docker.host",
			expectedURL: "http://custom.docker.host:8545",
			description: "Should replace localhost with custom Docker host",
		},
		{
			name:        "Replace 127.0.0.1 with custom host",
			inputURL:    "https://127.0.0.1:3000",
			dockersHost: "custom.docker.host",
			expectedURL: "https://custom.docker.host:3000",
			description: "Should replace 127.0.0.1 with custom Docker host",
		},
		{
			name:        "Do not replace localhost in subdomain",
			inputURL:    "https://localhost.mycooldomain.com:8545",
			dockersHost: "custom.docker.host",
			expectedURL: "https://localhost.mycooldomain.com:8545",
			description: "Should NOT replace localhost when it's part of a domain name",
		},
		{
			name:        "Do not replace localhost in API subdomain",
			inputURL:    "https://api.localhost.network:3000",
			dockersHost: "custom.docker.host",
			expectedURL: "https://api.localhost.network:3000",
			description: "Should NOT replace localhost when it's part of a subdomain",
		},
		{
			name:        "Do not replace localhost in service name",
			inputURL:    "https://my-localhost-service.com:8080",
			dockersHost: "custom.docker.host",
			expectedURL: "https://my-localhost-service.com:8080",
			description: "Should NOT replace localhost when it's part of a service name",
		},
		{
			name:        "Do not change external URLs",
			inputURL:    "http://mainnet.infura.io/v3/key",
			dockersHost: "custom.docker.host",
			expectedURL: "http://mainnet.infura.io/v3/key",
			description: "Should not change external URLs",
		},
		{
			name:        "Replace localhost without port",
			inputURL:    "http://localhost",
			dockersHost: "custom.docker.host",
			expectedURL: "http://custom.docker.host",
			description: "Should replace localhost without port",
		},
		{
			name:        "Replace localhost with path",
			inputURL:    "http://localhost/api/v1",
			dockersHost: "custom.docker.host",
			expectedURL: "http://custom.docker.host/api/v1",
			description: "Should replace localhost and preserve path",
		},
		{
			name:        "Replace localhost with query params",
			inputURL:    "http://localhost:8545?param=value",
			dockersHost: "custom.docker.host",
			expectedURL: "http://custom.docker.host:8545?param=value",
			description: "Should replace localhost and preserve query parameters",
		},
		{
			name:        "WebSocket localhost replacement",
			inputURL:    "ws://localhost:8546",
			dockersHost: "custom.docker.host",
			expectedURL: "ws://custom.docker.host:8546",
			description: "Should replace localhost in WebSocket URLs",
		},
		{
			name:        "Secure WebSocket localhost replacement",
			inputURL:    "wss://localhost:8546/ws",
			dockersHost: "custom.docker.host",
			expectedURL: "wss://custom.docker.host:8546/ws",
			description: "Should replace localhost in secure WebSocket URLs",
		},
		{
			name:        "Do not replace localhost in complex subdomain",
			inputURL:    "https://dev.localhost.internal.company.com:3000",
			dockersHost: "custom.docker.host",
			expectedURL: "https://dev.localhost.internal.company.com:3000",
			description: "Should NOT replace localhost when it's part of a complex subdomain",
		},
		{
			name:        "Do not replace localhost-like service names",
			inputURL:    "https://localhost-dev.myservice.com:8080",
			dockersHost: "custom.docker.host",
			expectedURL: "https://localhost-dev.myservice.com:8080",
			description: "Should NOT replace localhost when it's part of a hyphenated service name",
		},
		{
			name:        "Replace localhost in fragment",
			inputURL:    "http://localhost:8545/api#section",
			dockersHost: "custom.docker.host",
			expectedURL: "http://custom.docker.host:8545/api#section",
			description: "Should replace localhost and preserve URL fragment",
		},
		{
			name:        "Replace standalone localhost in complex string",
			inputURL:    "Connect to localhost:8545 for RPC",
			dockersHost: "custom.docker.host",
			expectedURL: "Connect to custom.docker.host:8545 for RPC",
			description: "Should replace standalone localhost in descriptive text",
		},
		{
			name:        "Do not replace when localhost is part of word",
			inputURL:    "Visit our-localhost-cluster.example.com",
			dockersHost: "custom.docker.host",
			expectedURL: "Visit our-localhost-cluster.example.com",
			description: "Should NOT replace localhost when it's part of a hyphenated word",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DOCKERS_HOST", tt.dockersHost)

			result := EnsureDockerHost(tt.inputURL)
			assert.Equal(t, tt.expectedURL, result, tt.description)
		})
	}
}

// TestEnsureDockerHostCrossPlatform tests cross-platform behavior
func TestEnsureDockerHostCrossPlatform(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	platforms := []struct {
		name        string
		dockersHost string
		description string
	}{
		{
			name:        "Linux behavior",
			dockersHost: "localhost",
			description: "Linux should use localhost",
		},
		{
			name:        "macOS/Windows behavior",
			dockersHost: "host.docker.internal",
			description: "macOS and Windows should use host.docker.internal",
		},
	}

	testCases := []struct {
		input       string
		description string
	}{
		{"http://localhost:8545", "Should replace localhost"},
		{"https://127.0.0.1:3000", "Should replace 127.0.0.1"},
		{"https://localhost.mycooldomain.com:8545", "Should NOT replace localhost in domain"},
		{"https://api.localhost.network:3000", "Should NOT replace localhost in subdomain"},
		{"https://my-localhost-service.com:8080", "Should NOT replace localhost in service name"},
		{"http://mainnet.infura.io/v3/key", "Should not change external URLs"},
	}

	for _, platform := range platforms {
		t.Run(platform.name, func(t *testing.T) {
			os.Setenv("DOCKERS_HOST", platform.dockersHost)

			for _, tc := range testCases {
				t.Run(tc.description, func(t *testing.T) {
					result := EnsureDockerHost(tc.input)

					// Verify the transformation logic
					if tc.input == "http://localhost:8545" {
						expected := fmt.Sprintf("http://%s:8545", platform.dockersHost)
						assert.Equal(t, expected, result)
					} else if tc.input == "https://127.0.0.1:3000" {
						expected := fmt.Sprintf("https://%s:3000", platform.dockersHost)
						assert.Equal(t, expected, result)
					} else {
						// These URLs should not be modified
						assert.Equal(t, tc.input, result)
					}
				})
			}
		})
	}
}

// TestEnsureDockerHostRegexFallback tests the regex fallback for malformed URLs
func TestEnsureDockerHostRegexFallback(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	os.Setenv("DOCKERS_HOST", "test.docker.host")

	tests := []struct {
		name        string
		inputURL    string
		expectedURL string
		description string
	}{
		{
			name:        "URL with control characters and localhost",
			inputURL:    "ht\x00tp://localhost:8545",
			expectedURL: "ht\x00tp://test.docker.host:8545",
			description: "Should use regex fallback for URLs with control characters",
		},
		{
			name:        "URL with invalid scheme and 127.0.0.1",
			inputURL:    "ht tp://127.0.0.1:3000/path",
			expectedURL: "ht tp://test.docker.host:3000/path",
			description: "Should use regex fallback for URLs with spaces in scheme",
		},
		{
			name:        "Plain text with localhost port",
			inputURL:    "Connect to localhost:8545 for RPC",
			expectedURL: "Connect to test.docker.host:8545 for RPC",
			description: "Should replace localhost in plain text",
		},
		{
			name:        "Configuration value with 127.0.0.1",
			inputURL:    "RPC_URL=127.0.0.1:3000",
			expectedURL: "RPC_URL=test.docker.host:3000",
			description: "Should replace 127.0.0.1 in configuration-style strings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureDockerHost(tt.inputURL)
			assert.Equal(t, tt.expectedURL, result, tt.description)
		})
	}
}

// TestDockerNetworkingEdgeCases tests edge cases in URL parsing and transformation
func TestDockerNetworkingEdgeCases(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	os.Setenv("DOCKERS_HOST", "test.docker.host")

	tests := []struct {
		name        string
		input       string
		expected    string
		description string
	}{
		{
			name:        "Empty string",
			input:       "",
			expected:    "",
			description: "Empty string should remain empty",
		},
		{
			name:        "Just localhost",
			input:       "localhost",
			expected:    "test.docker.host",
			description: "Bare localhost should be replaced",
		},
		{
			name:        "Just 127.0.0.1",
			input:       "127.0.0.1",
			expected:    "test.docker.host",
			description: "Bare 127.0.0.1 should be replaced",
		},
		{
			name:        "URL with fragment",
			input:       "http://localhost:8545#section",
			expected:    "http://test.docker.host:8545#section",
			description: "URL with fragment should preserve fragment",
		},
		{
			name:        "URL with user info",
			input:       "http://user:pass@localhost:8545",
			expected:    "http://user:pass@test.docker.host:8545",
			description: "URL with user info should preserve user info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureDockerHost(tt.input)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// TestEnsureDockerHostParsing tests URL parsing behavior
func TestEnsureDockerHostParsing(t *testing.T) {
	// Save original environment
	originalDockerHost := os.Getenv("DOCKERS_HOST")
	defer func() {
		if originalDockerHost != "" {
			os.Setenv("DOCKERS_HOST", originalDockerHost)
		} else {
			os.Unsetenv("DOCKERS_HOST")
		}
	}()

	os.Setenv("DOCKERS_HOST", "docker.host")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Valid HTTP URL",
			input:    "http://localhost:8545/path?query=value",
			expected: "http://docker.host:8545/path?query=value",
		},
		{
			name:     "Valid HTTPS URL",
			input:    "https://127.0.0.1:443/secure",
			expected: "https://docker.host:443/secure",
		},
		{
			name:     "WebSocket URL",
			input:    "ws://localhost:8546",
			expected: "ws://docker.host:8546",
		},
		{
			name:     "URL without scheme",
			input:    "localhost:8545",
			expected: "docker.host:8545",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EnsureDockerHost(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify the result is a valid URL (if it was valid to begin with)
			if _, err := url.Parse(tt.input); err == nil {
				_, err := url.Parse(result)
				assert.NoError(t, err, "Result should be a valid URL")
			}
		})
	}
}

func TestNetworkingRegression(t *testing.T) {
	t.Log("🔍 Running Docker networking regression protection tests...")

	// Test 1: Cross-platform Docker host behavior
	t.Run("CrossPlatformBehavior", func(t *testing.T) {
		testPlatformBehavior := func(t *testing.T, platformName, expectedDockerHost string) {
			// Save original environment
			originalDockerHost := os.Getenv("DOCKERS_HOST")
			defer func() {
				if originalDockerHost != "" {
					os.Setenv("DOCKERS_HOST", originalDockerHost)
				} else {
					os.Unsetenv("DOCKERS_HOST")
				}
			}()

			t.Logf("🔧 Testing %s behavior (DOCKERS_HOST=%s)...", platformName, expectedDockerHost)
			os.Setenv("DOCKERS_HOST", expectedDockerHost)

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

			for _, tc := range testCases {
				t.Run(tc.desc, func(t *testing.T) {
					result := EnsureDockerHost(tc.input)
					assert.Equal(t, tc.expected, result,
						"FAILED: %s\nInput: %s\nExpected: %s\nGot: %s",
						tc.desc, tc.input, tc.expected, result)
					t.Logf("✅ PASSED: %s", tc.desc)
				})
			}
		}

		t.Run("Linux", func(t *testing.T) {
			testPlatformBehavior(t, "Linux", "localhost")
		})

		t.Run("macOS", func(t *testing.T) {
			testPlatformBehavior(t, "macOS", "host.docker.internal")
		})
	})

	// Test 2: Verify regression protection
	t.Run("RegressionProtection", func(t *testing.T) {
		// Save original environment
		originalDockerHost := os.Getenv("DOCKERS_HOST")
		defer func() {
			if originalDockerHost != "" {
				os.Setenv("DOCKERS_HOST", originalDockerHost)
			} else {
				os.Unsetenv("DOCKERS_HOST")
			}
		}()

		// Test that GetRPCURL always returns localhost
		t.Run("GetRPCURLAlwaysUsesLocalhost", func(t *testing.T) {
			testPorts := []int{8545, 9545, 3000}
			dockerHosts := []string{"localhost", "host.docker.internal"}

			for _, dockerHost := range dockerHosts {
				for _, port := range testPorts {
					t.Run(fmt.Sprintf("DOCKERS_HOST=%s_port=%d", dockerHost, port), func(t *testing.T) {
						os.Setenv("DOCKERS_HOST", dockerHost)
						result := GetRPCURL(port)
						expected := fmt.Sprintf("http://localhost:%d", port)
						assert.Equal(t, expected, result,
							"GetRPCURL should always use localhost, not %s", dockerHost)
					})
				}
			}
		})

		// Test that Docker containers can still access host services
		t.Run("DockerHostConfiguration", func(t *testing.T) {
			// Simulate what would happen in docker-compose.yaml generation
			os.Setenv("DOCKERS_HOST", "host.docker.internal")

			// Fork URL should be transformed for container access
			forkURL := "http://localhost:8545"
			dockerForkURL := EnsureDockerHost(forkURL)
			expected := "http://host.docker.internal:8545"
			assert.Equal(t, expected, dockerForkURL,
				"Fork URL should be transformed for Docker container access")

			// But RPC URL for host access should remain localhost
			rpcURL := GetRPCURL(8545)
			expectedRPC := "http://localhost:8545"
			assert.Equal(t, expectedRPC, rpcURL,
				"RPC URL for host access should always use localhost")
		})
	})

}
