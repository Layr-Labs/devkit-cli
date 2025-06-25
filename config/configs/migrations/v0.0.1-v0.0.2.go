package configMigrations

import (
	"github.com/Layr-Labs/devkit-cli/pkg/migration"

	"gopkg.in/yaml.v3"
)

func Migration_0_0_1_to_0_0_2(user, old, new *yaml.Node) (*yaml.Node, error) {
	engine := migration.PatchEngine{
		Old:  old,
		New:  new,
		User: user,
		Rules: []migration.PatchRule{
			// Add project_uuid field (empty string by default)
			{Path: []string{"config", "project", "project_uuid"}, Condition: migration.Always{}},
			// Add telemetry_enabled field (false by default)
			{Path: []string{"config", "project", "telemetry_enabled"}, Condition: migration.Always{}},
			// Add template baseUrl that should be present (leave unchanged if different)
			{Path: []string{"config", "project", "templateBaseUrl"}, Condition: migration.IfUnchanged{}},
			// Add template version that should be present (leave unchanged if different)
			{Path: []string{"config", "project", "templateVersion"}, Condition: migration.IfUnchanged{}},
		},
	}
	err := engine.Apply()
	if err != nil {
		return nil, err
	}

	// bump version node
	if v := migration.ResolveNode(user, []string{"version"}); v != nil {
		v.Value = "0.0.2"
	}
	return user, nil
}
