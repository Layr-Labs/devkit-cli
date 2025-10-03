package assets

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed docker-compose.yaml
var DockerCompose []byte

// WriteDockerComposeToPath writes docker-compose.yaml to a fixed path.
func WriteDockerComposeToPath() (string, error) {
	// Get project's absolute path
	absProjectPath, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}
	// Store anvils docker-compose.yaml in devnet dir at project root
	dir := filepath.Join(absProjectPath, "devnet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", dir, err)
	}
	// Write embed each devnet start to ensure any changes are propagated
	path := filepath.Join(dir, "docker-compose.yaml")
	if err = os.WriteFile(path, DockerCompose, 0o644); err != nil {
		return "", fmt.Errorf("failed to write %s: %w", path, err)
	}
	return path, nil
}
