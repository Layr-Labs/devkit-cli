package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Layr-Labs/devkit-cli/internal/version"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/credentials"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// OCIArtifactBuilder creates OCI artifacts for EigenRuntime specs
type OCIArtifactBuilder struct {
	logger iface.Logger
}

// NewOCIArtifactBuilder creates a new OCI artifact builder
func NewOCIArtifactBuilder(logger iface.Logger) *OCIArtifactBuilder {
	return &OCIArtifactBuilder{
		logger: logger,
	}
}

// CreateEigenRuntimeArtifact creates and pushes an OCI artifact containing the runtime spec
// Using oras-go for full OCI artifact support including custom media types and artifactType
//
// This implementation produces an OCI artifact with the following structure:
//
//	{
//	  "schemaVersion": 2,
//	  "mediaType": "application/vnd.oci.image.manifest.v1+json",
//	  "artifactType": "application/vnd.eigenruntime.manifest.v1",
//	  "config": {
//	    "mediaType": "application/vnd.eigenruntime.manifest.config.v1+json",
//	    "digest": "sha256:<config-digest>",
//	    "size": <config-size>
//	  },
//	  "layers": [{
//	    "mediaType": "text/yaml",
//	    "digest": "sha256:<runtime-spec-digest>",
//	    "size": <spec-size>
//	  }],
//	  "annotations": {
//	    "org.opencontainers.image.source": "https://github.com/Layr-Labs/devkit-cli",
//	    "org.opencontainers.image.description": "EigenRuntime specification for <name>",
//	    "org.opencontainers.image.created": "<timestamp>",
//	    "io.eigenruntime.spec.version": "v1"
//	  }
//	}
func (b *OCIArtifactBuilder) CreateEigenRuntimeArtifact(
	runtimeSpec []byte,
	registry string,
	avsName string,
	tag string,
) (string, error) {
	ctx := context.Background()

	// Construct the full image reference
	imageRef := fmt.Sprintf("%s:%s", registry, tag)

	b.logger.Info("Creating EigenRuntime OCI artifact for %s", imageRef)

	// Create an in-memory store for building the artifact
	memStore := memory.New()

	// Create the config JSON
	configContent := b.createConfigBlob(avsName, tag)
	configMediaType := "application/vnd.eigenruntime.manifest.config.v1+json"

	// Add config to store
	configDesc, err := b.addToStore(ctx, memStore, configMediaType, configContent)
	if err != nil {
		return "", fmt.Errorf("failed to add config to store: %w", err)
	}

	// Add runtime spec layer to store
	specMediaType := "text/yaml"
	specDesc, err := b.addToStore(ctx, memStore, specMediaType, runtimeSpec)
	if err != nil {
		return "", fmt.Errorf("failed to add runtime spec to store: %w", err)
	}

	// Create the manifest
	manifest := ocispec.Manifest{
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: "application/vnd.eigenruntime.manifest.v1",
		Config:       configDesc,
		Layers:       []ocispec.Descriptor{specDesc},
		Annotations: map[string]string{
			"org.opencontainers.image.source":      "https://github.com/Layr-Labs/devkit-cli",
			"org.opencontainers.image.description": "EigenRuntime specification for AVS " + avsName,
			"org.opencontainers.image.created":     time.Now().UTC().Format(time.RFC3339),
			"io.eigenruntime.spec.version":         "v1",
		},
	}

	// Create a proper manifest with schemaVersion
	// We need to manually construct the JSON to ensure artifactType is preserved
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

	// Marshal the manifest
	manifestBytes, err := json.Marshal(manifestMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Add manifest to store
	manifestDesc, err := b.addToStore(ctx, memStore, ocispec.MediaTypeImageManifest, manifestBytes)
	if err != nil {
		return "", fmt.Errorf("failed to add manifest to store: %w", err)
	}

	// Tag the manifest in the memory store so oras.Copy can find it
	err = memStore.Tag(ctx, manifestDesc, tag)
	if err != nil {
		return "", fmt.Errorf("failed to tag manifest in memory store: %w", err)
	}

	// Parse the repository reference
	repo, err := remote.NewRepository(imageRef)
	if err != nil {
		return "", fmt.Errorf("failed to create repository: %w", err)
	}

	// Set up authentication using Docker's credential store
	repo.Client = &auth.Client{
		Cache: auth.DefaultCache,
		Credential: func(ctx context.Context, reg string) (auth.Credential, error) {
			// Try to load Docker config
			dockerConfigDir := os.Getenv("DOCKER_CONFIG")
			if dockerConfigDir == "" {
				homeDir, _ := os.UserHomeDir()
				dockerConfigDir = fmt.Sprintf("%s/.docker", homeDir)
			}

			cfg, err := config.Load(dockerConfigDir)
			if err != nil {
				b.logger.Debug("Failed to load Docker config from %s: %v", dockerConfigDir, err)
				// Return empty credentials for anonymous access
				return auth.Credential{}, nil
			}

			// Try 1: Get auth directly from config file
			authConfig, err := cfg.GetAuthConfig(reg)

			// Try 2: If that didn't work and we have a credential store, try the native store
			if (authConfig.Username == "" && authConfig.Password == "") && cfg.CredentialsStore != "" {
				b.logger.Debug("No auth in config file, trying credential store")

				// For native store, normalize to the exact key used in GitHub Actions
				registryToLookup := reg
				if reg == "docker.io" || reg == "registry-1.docker.io" {
					registryToLookup = "https://index.docker.io/v1/"
				}

				store := credentials.NewNativeStore(cfg, cfg.CredentialsStore)
				authConfig, err = store.Get(registryToLookup)
			}

			if err != nil {
				b.logger.Debug("No credentials found for registry %s: %v", reg, err)
				// Return empty credentials for anonymous access
				return auth.Credential{}, nil
			}

			if authConfig.Username == "" && authConfig.Password == "" {
				b.logger.Debug("Empty credentials for registry %s", reg)
				// Return empty credentials for anonymous access
				return auth.Credential{}, nil
			}

			b.logger.Debug("Found credentials for registry %s - Username: %s", reg, authConfig.Username)

			// Convert to oras auth.Credential
			cred := auth.Credential{
				Username: authConfig.Username,
				Password: authConfig.Password,
			}

			// Handle token-based auth (e.g., for Docker Hub)
			if authConfig.IdentityToken != "" {
				cred.RefreshToken = authConfig.IdentityToken
				cred.AccessToken = authConfig.RegistryToken
			}

			return cred, nil
		},
	}

	// Use HTTPS by default
	repo.PlainHTTP = false

	// Push the artifact
	b.logger.Info("Pushing EigenRuntime artifact to %s", imageRef)

	// Use oras.Copy to push the complete artifact graph from memory store to registry
	// This preserves the artifactType and all custom media types in the manifest
	// oras.Copy handles:
	// - Walking the dependency graph from the manifest
	// - Pushing all referenced blobs (config and layers)
	// - Pushing the manifest itself with proper media type
	// - Tagging the manifest in the registry
	_, err = oras.Copy(ctx, memStore, tag, repo, tag,
		oras.CopyOptions{
			CopyGraphOptions: oras.CopyGraphOptions{
				Concurrency: 3,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to push artifact: %w", err)
	}

	digestStr := manifestDesc.Digest.String()
	b.logger.Info("Successfully pushed EigenRuntime artifact with digest: %s", digestStr)

	return digestStr, nil
}

// addToStore adds content to the memory store and returns its descriptor
func (b *OCIArtifactBuilder) addToStore(ctx context.Context, store *memory.Store, mediaType string, content []byte) (ocispec.Descriptor, error) {
	// Calculate digest
	d := digest.FromBytes(content)

	// Create descriptor
	desc := ocispec.Descriptor{
		MediaType: mediaType,
		Digest:    d,
		Size:      int64(len(content)),
	}

	// Push to store
	err := store.Push(ctx, desc, bytes.NewReader(content))
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	return desc, nil
}

// createConfigBlob creates the OCI config JSON blob as specified in the TDD
func (b *OCIArtifactBuilder) createConfigBlob(name, tag string) []byte {
	// TODO: parameterize to template.
	c := map[string]interface{}{
		"formatVersion":          "1.0",
		"eigenRuntimeAPIVersion": "v1",
		"kind":                   "Hourglass",
		"avsName":                name,
		"validationSchema":       "https://eigenruntime.io/schemas/hourglass/v1/manifest.json",
		"metadata": map[string]string{
			"createdAt":      time.Now().UTC().Format(time.RFC3339),
			"releaseVersion": tag,
			"devkitVersion":  getDevkitVersion(),
		},
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		// This should never happen with the static structure above
		b.logger.Error("Failed to marshal config JSON: %v", err)
		return []byte("{}")
	}

	return data
}

// getDevkitVersion returns the current DevKit version
func getDevkitVersion() string {
	if version.Version != "" {
		return version.Version
	}
	return "dev"
}

// ComputeRuntimeSpecDigest computes the SHA256 digest of a runtime spec
func ComputeRuntimeSpecDigest(runtimeSpec []byte) string {
	hash := sha256.Sum256(runtimeSpec)
	return "sha256:" + hex.EncodeToString(hash[:])
}
