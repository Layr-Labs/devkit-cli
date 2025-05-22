package contextMigrations

import (
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/config"
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_3_to_0_0_4(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			{Path: []string{"context", "eigenlayer"}, Condition: migration.Always{}},
			{Path: []string{"context", "eigenlayer", "allocation_manager"}, Condition: migration.Always{}},
			{Path: []string{"context", "eigenlayer", "delegation_manager"}, Condition: migration.Always{}},
		},
	}
	err := engine.Apply()
	if err != nil {
		return nil, err
	}



	// Copy Zeus config for projects created before Zeus integration
	zeusConfigSrc := filepath.Join("config", ".zeus")
	zeusConfigDst := ".zeus"

	// Check if source Zeus config exists
	if _, err := os.Stat(zeusConfigSrc); err == nil {
		// Read Zeus config content
		zeusConfigContent := config.ZeusConfig
		if err == nil {
			// Write Zeus config to project root if it doesn't exist already
			if _, err := os.Stat(zeusConfigDst); os.IsNotExist(err) {
				_ = os.WriteFile(zeusConfigDst, []byte(zeusConfigContent), 0644)
			}
		}
	}

	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.4"
	}
	return user, nil
}
