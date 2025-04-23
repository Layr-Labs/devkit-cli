package devnet

import (
	"log"

	"devkit-cli/docker/anvil"
)

// WriteEmbeddedArtifacts writes the embedded docker-compose.yaml and state.json files.
// Returns the paths to the written files.
func WriteEmbeddedArtifacts() (composePath string, statePath string) {
	var err error

	composePath, err = assets.WriteDockerComposeToPath()
	if err != nil {
		log.Fatalf("❌ Could not write embedded docker-compose.yaml: %v", err)
	}

	statePath, err = assets.WriteStateJSONToPath()
	if err != nil {
		log.Fatalf("❌ Could not write embedded state.json: %v", err)
	}

	return composePath, statePath
}
