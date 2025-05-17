package assets

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed docker-compose.yaml
var DockerCompose []byte

// WriteDockerComposeToPath writes docker-compose.yaml to a fixed path.
func WriteDockerComposeToPath() (string, error) {
	dir := filepath.Join(os.TempDir(), "devkit-compose")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "docker-compose.yaml")
	err := os.WriteFile(path, DockerCompose, 0o644)
	return path, err
}


// GetDockerComposePath returns the known path to docker-compose.yaml without writing.
func GetDockerComposePath() string {
	return filepath.Join(os.TempDir(), "devkit-compose", "docker-compose.yaml")
}

// GetStateJSONPath returns the known path to state.json without writing.
func GetStateJSONPath() string {
	return filepath.Join(os.TempDir(), "devkit-state", "state.json")
}
