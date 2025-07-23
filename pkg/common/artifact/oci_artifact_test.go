package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
)

// mockLogger implements iface.Logger for testing
type mockLogger struct {
	messages []string
}

func (m *mockLogger) Info(format string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("[INFO] "+format, args...))
}

func (m *mockLogger) Error(format string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("[ERROR] "+format, args...))
}

func (m *mockLogger) Debug(format string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("[DEBUG] "+format, args...))
}

func (m *mockLogger) Warn(format string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("[WARN] "+format, args...))
}

func (m *mockLogger) Fatal(format string, args ...interface{}) {
	m.messages = append(m.messages, fmt.Sprintf("[FATAL] "+format, args...))
	panic(fmt.Sprintf(format, args...))
}

func (m *mockLogger) Title(title string, args ...any) {
	m.messages = append(m.messages, fmt.Sprintf("[TITLE] "+title, args...))
}

func TestNewOCIArtifactBuilder(t *testing.T) {
	logger := &mockLogger{}
	builder := NewOCIArtifactBuilder(logger)

	if builder == nil {
		t.Fatal("Expected builder to be created")
	}

	if builder.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestCreateConfigBlob(t *testing.T) {
	logger := &mockLogger{}
	builder := NewOCIArtifactBuilder(logger)

	tests := []struct {
		name    string
		avsName string
		tag     string
		want    map[string]interface{}
	}{
		{
			name:    "basic config",
			avsName: "test-avs",
			tag:     "v1.0.0",
			want: map[string]interface{}{
				"formatVersion":          "1.0",
				"eigenRuntimeAPIVersion": "v1",
				"kind":                   "Hourglass",
				"avsName":                "test-avs",
				"validationSchema":       "https://eigenruntime.io/schemas/hourglass/v1/manifest.json",
			},
		},
		{
			name:    "special characters in name",
			avsName: "my-avs_123",
			tag:     "opset-0-v2",
			want: map[string]interface{}{
				"formatVersion":          "1.0",
				"eigenRuntimeAPIVersion": "v1",
				"kind":                   "Hourglass",
				"avsName":                "my-avs_123",
				"validationSchema":       "https://eigenruntime.io/schemas/hourglass/v1/manifest.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configBytes := builder.createConfigBlob(tt.avsName, tt.tag)

			// Verify it's valid JSON
			var config map[string]interface{}
			if err := json.Unmarshal(configBytes, &config); err != nil {
				t.Fatalf("Failed to unmarshal config blob: %v", err)
			}

			// Check required fields
			for key, expectedValue := range tt.want {
				if actualValue, ok := config[key]; !ok {
					t.Errorf("Missing key %s in config", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %s, got %v, want %v", key, actualValue, expectedValue)
				}
			}

			// Check metadata exists
			metadata, ok := config["metadata"].(map[string]interface{})
			if !ok {
				t.Fatal("metadata field missing or not a map")
			}

			// Verify metadata fields
			if _, ok := metadata["createdAt"]; !ok {
				t.Error("metadata.createdAt missing")
			}
			if metadata["releaseVersion"] != tt.tag {
				t.Errorf("metadata.releaseVersion = %v, want %v", metadata["releaseVersion"], tt.tag)
			}
			if _, ok := metadata["devkitVersion"]; !ok {
				t.Error("metadata.devkitVersion missing")
			}
		})
	}
}

func TestAddToStore(t *testing.T) {
	logger := &mockLogger{}
	builder := NewOCIArtifactBuilder(logger)
	ctx := context.Background()

	tests := []struct {
		name      string
		mediaType string
		content   []byte
		wantErr   bool
	}{
		{
			name:      "add JSON content",
			mediaType: "application/json",
			content:   []byte(`{"test": "data"}`),
			wantErr:   false,
		},
		{
			name:      "add YAML content",
			mediaType: "text/yaml",
			content:   []byte("key: value\narray:\n  - item1\n  - item2"),
			wantErr:   false,
		},
		{
			name:      "add empty content",
			mediaType: "application/octet-stream",
			content:   []byte{},
			wantErr:   false,
		},
		{
			name:      "add large content",
			mediaType: "application/octet-stream",
			content:   bytes.Repeat([]byte("a"), 10000),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memory.New()

			desc, err := builder.addToStore(ctx, store, tt.mediaType, tt.content)
			if (err != nil) != tt.wantErr {
				t.Fatalf("addToStore() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify descriptor
				if desc.MediaType != tt.mediaType {
					t.Errorf("MediaType = %v, want %v", desc.MediaType, tt.mediaType)
				}
				if desc.Size != int64(len(tt.content)) {
					t.Errorf("Size = %v, want %v", desc.Size, len(tt.content))
				}
				if !strings.HasPrefix(string(desc.Digest), "sha256:") {
					t.Errorf("Digest should start with sha256:, got %v", desc.Digest)
				}

				// Verify content can be fetched from store
				rc, err := store.Fetch(ctx, desc)
				if err != nil {
					t.Fatalf("Failed to fetch content from store: %v", err)
				}
				defer rc.Close()

				buf := new(bytes.Buffer)
				if _, err := buf.ReadFrom(rc); err != nil {
					t.Fatalf("Failed to read content: %v", err)
				}

				if !bytes.Equal(buf.Bytes(), tt.content) {
					t.Error("Fetched content doesn't match original")
				}
			}
		})
	}
}

func TestComputeRuntimeSpecDigest(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name:    "simple text",
			content: []byte("hello world"),
			want:    "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:    "empty content",
			content: []byte{},
			want:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:    "yaml content",
			content: []byte("apiVersion: v1\nkind: Test"),
			want:    "sha256:10573128831c13c517c4f8ee28a02440058c7f8eaaa163c24ad65dc7e0852b88",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeRuntimeSpecDigest(tt.content)
			if got != tt.want {
				t.Errorf("ComputeRuntimeSpecDigest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateEigenRuntimeArtifact_ManifestStructure(t *testing.T) {
	logger := &mockLogger{}
	builder := NewOCIArtifactBuilder(logger)
	ctx := context.Background()

	// Create a test runtime spec
	runtimeSpec := []byte(`apiVersion: eigenruntime.io/v1
kind: Hourglass
name: test-avs
version: 1
spec:
  aggregator:
    registry: test-registry
    digest: sha256:abc123
`)

	// Create in-memory store and simulate the manifest creation
	store := memory.New()

	// Add config
	configContent := builder.createConfigBlob("test-avs", "v1.0.0")
	configDesc, err := builder.addToStore(ctx, store, "application/vnd.eigenruntime.manifest.config.v1+json", configContent)
	if err != nil {
		t.Fatalf("Failed to add config: %v", err)
	}

	// Add runtime spec
	specDesc, err := builder.addToStore(ctx, store, "text/yaml", runtimeSpec)
	if err != nil {
		t.Fatalf("Failed to add spec: %v", err)
	}

	// Create manifest
	manifest := ocispec.Manifest{
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: "application/vnd.eigenruntime.manifest.v1",
		Config:       configDesc,
		Layers:       []ocispec.Descriptor{specDesc},
		Annotations: map[string]string{
			"org.opencontainers.image.source":      "https://github.com/Layr-Labs/devkit-cli",
			"org.opencontainers.image.description": "EigenRuntime specification for AVS test-avs",
			"org.opencontainers.image.created":     time.Now().UTC().Format(time.RFC3339),
			"io.eigenruntime.spec.version":         "v1",
		},
	}

	// Create manifest map as in the actual implementation
	manifestMap := map[string]interface{}{
		"schemaVersion": 2,
		"mediaType":     manifest.MediaType,
		"artifactType":  manifest.ArtifactType,
		"config": map[string]interface{}{
			"mediaType": manifest.Config.MediaType,
			"digest":    manifest.Config.Digest.String(),
			"size":      manifest.Config.Size,
		},
		"layers": func() []map[string]interface{} {
			layers := make([]map[string]interface{}, len(manifest.Layers))
			for i, layer := range manifest.Layers {
				layers[i] = map[string]interface{}{
					"mediaType": layer.MediaType,
					"digest":    layer.Digest.String(),
					"size":      layer.Size,
				}
			}
			return layers
		}(),
		"annotations": manifest.Annotations,
	}

	// Marshal and verify
	manifestBytes, err := json.Marshal(manifestMap)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	// Unmarshal to verify structure
	var parsedManifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &parsedManifest); err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	// Verify required fields
	if v, ok := parsedManifest["schemaVersion"].(float64); !ok || v != 2 {
		t.Error("schemaVersion should be 2")
	}

	if v, ok := parsedManifest["mediaType"].(string); !ok || v != "application/vnd.oci.image.manifest.v1+json" {
		t.Error("mediaType incorrect")
	}

	if v, ok := parsedManifest["artifactType"].(string); !ok || v != "application/vnd.eigenruntime.manifest.v1" {
		t.Error("artifactType incorrect or missing")
	}

	// Verify config
	config, ok := parsedManifest["config"].(map[string]interface{})
	if !ok {
		t.Fatal("config field missing or not a map")
	}

	if v, ok := config["mediaType"].(string); !ok || v != "application/vnd.eigenruntime.manifest.config.v1+json" {
		t.Error("config.mediaType incorrect")
	}

	// Verify layers
	layers, ok := parsedManifest["layers"].([]interface{})
	if !ok || len(layers) != 1 {
		t.Fatal("layers field missing or incorrect length")
	}

	layer0, ok := layers[0].(map[string]interface{})
	if !ok {
		t.Fatal("layer[0] not a map")
	}

	if v, ok := layer0["mediaType"].(string); !ok || v != "text/yaml" {
		t.Error("layer[0].mediaType incorrect")
	}

	// Verify annotations exist
	if _, ok := parsedManifest["annotations"].(map[string]interface{}); !ok {
		t.Error("annotations field missing")
	}
}

func TestCreateEigenRuntimeArtifact_Errors(t *testing.T) {
	logger := &mockLogger{}
	builder := NewOCIArtifactBuilder(logger)

	tests := []struct {
		name        string
		runtimeSpec []byte
		registry    string
		avsName     string
		tag         string
		wantErr     string
	}{
		{
			name:        "invalid registry format",
			runtimeSpec: []byte("test"),
			registry:    "invalid registry!@#",
			avsName:     "test",
			tag:         "v1",
			wantErr:     "failed to create repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := builder.CreateEigenRuntimeArtifact(
				tt.runtimeSpec,
				tt.registry,
				tt.avsName,
				tt.tag,
			)

			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %v, want error containing %v", err, tt.wantErr)
			}
		})
	}
}
