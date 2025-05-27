package devnet

import (
	"log"

	"github.com/Layr-Labs/devkit-cli/docker/anvil"
)

// WriteEmbeddedArtifacts writes the embedded docker-compose.yaml.
// Returns the paths to the written files.
func WriteEmbeddedArtifacts() (composePath string) {
	var err error

	composePath, err = assets.WriteDockerComposeToPath()
	if err != nil {
		log.Fatalf("❌ Could not write embedded docker-compose.yaml: %v", err)
	}

	return composePath
}
