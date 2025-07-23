# OCI Artifact Package

This package provides functionality for creating and managing OCI (Open Container Initiative) artifacts specifically designed for EigenRuntime specifications. It uses the `oras-go` library to create standards-compliant OCI artifacts with custom media types.

## Overview

The `artifact` package enables DevKit to package EigenRuntime specifications as OCI artifacts, making them distributable through standard container registries like Docker Hub, GitHub Container Registry (GHCR), Amazon ECR, and others.

## Key Features

- **Standards-Compliant OCI Artifacts**: Creates artifacts that fully comply with the OCI specification
- **Custom Media Types**: Supports EigenRuntime-specific media types for configs and layers
- **Registry Authentication**: Integrates with Docker credential helpers for seamless authentication
- **In-Memory Construction**: Uses memory stores for efficient artifact building before pushing

## Architecture

### OCI Artifact Structure

The package creates OCI artifacts with the following manifest structure:

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.eigenruntime.manifest.v1",
  "config": {
    "mediaType": "application/vnd.eigenruntime.manifest.config.v1+json",
    "digest": "sha256:<config-digest>",
    "size": <config-size>
  },
  "layers": [{
    "mediaType": "text/yaml",
    "digest": "sha256:<runtime-spec-digest>",
    "size": <spec-size>
  }],
  "annotations": {
    "org.opencontainers.image.source": "https://github.com/Layr-Labs/devkit-cli",
    "org.opencontainers.image.description": "EigenRuntime specification for <name>",
    "org.opencontainers.image.created": "<timestamp>",
    "io.eigenruntime.spec.version": "v1"
  }
}
```

### Config Blob Format

The config blob contains metadata about the EigenRuntime specification:

```json
{
  "formatVersion": "1.0",
  "eigenRuntimeAPIVersion": "v1",
  "kind": "template-type",
  "avsName": "<avs-name>",
  "validationSchema": "https://eigenruntime.io/schemas/template-type/v1/manifest.json",
  "metadata": {
    "createdAt": "2024-01-01T00:00:00Z",
    "releaseVersion": "<release-version>",
    "devkitVersion": "<devkit-version>"
  }
}
```

## Authentication

The package automatically uses Docker's credential helpers for authentication through `auth.DefaultClient`. This means:

1. It will use credentials from `~/.docker/config.json`
2. It supports Docker credential helpers (e.g., `docker-credential-desktop`, `docker-credential-ecr-login`)
3. No manual credential configuration is needed if you're already logged in via `docker login`
4. The DOCKER_CONFIG environment variable is respected if set

To authenticate with a registry, simply use `docker login` before running DevKit:
```bash
docker login ghcr.io
# or
docker login <your-registry>
```

## Troubleshooting

## Inspecting OCI Artifacts

**Important**: Don't use `docker pull` to inspect OCI artifacts, as Docker may transform the manifest for compatibility. Instead, use:

### Using oras CLI:
```bash
# Fetch the manifest
oras manifest fetch ghcr.io/myorg/my-avs:opset-0-v1

# Pull the artifact locally
oras pull ghcr.io/myorg/my-avs:opset-0-v1
```

### Using crane CLI:
```bash
# View the manifest
crane manifest ghcr.io/myorg/my-avs:opset-0-v1 | jq .

# Export the artifact
crane export ghcr.io/myorg/my-avs:opset-0-v1 - | tar -tv
```

These tools preserve the original OCI artifact structure, including the `artifactType` field and proper media types.

## Registry Compatibility

All major registries support OCI artifacts with `artifactType`. However, some registry web UIs may display artifacts differently than their actual stored format:

### Verifying Your Artifact

To see the actual OCI artifact manifest stored in the registry:

```bash
# Use oras to inspect the manifest
oras manifest fetch ghcr.io/myorg/my-avs:tag | jq .

# Or use crane
crane manifest ghcr.io/myorg/my-avs:tag | jq .

# Even docker shows the correct manifest
docker manifest inspect ghcr.io/myorg/my-avs:tag | jq .
```

```json
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.eigenruntime.manifest.v1",
  "config": {
    "mediaType": "application/vnd.eigenruntime.manifest.config.v1+json",
    "digest": "sha256:...",
    "size": 323
  },
  "layers": [{
    "mediaType": "text/yaml",
    "digest": "sha256:...",
    "size": 663
  }],
  "annotations": {
    "org.opencontainers.image.source": "https://github.com/Layr-Labs/devkit-cli",
    "org.opencontainers.image.description": "EigenRuntime specification for ...",
    "org.opencontainers.image.created": "2025-01-01T00:00:00Z",
    "io.eigenruntime.spec.version": "v1"
  }
}
``` 