package commands

import (
	"fmt"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"
	"gopkg.in/yaml.v3"

	"github.com/urfave/cli/v2"
)

// BuildCommand defines the "build" command
var BuildCommand = &cli.Command{
	Name:  "build",
	Usage: "Compiles AVS components (smart contracts via Foundry, Go binaries for operators/aggregators)",
	Flags: append([]cli.Flag{
		// TBD: Release flag will be implemented in future
		/*&cli.BoolFlag{
			Name:  "release",
			Usage: "Produce production-optimized artifacts",
		},*/
		&cli.StringFlag{
			Name:  "context",
			Usage: "devnet ,testnet or mainnet",
			Value: "devnet",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		// Run scriptPath from cwd
		const dir = ""

		// Get the config (based on if we're in a test or not)
		var cfg *common.ConfigWithContextConfig

		// First check if config is in context (for testing)
		if cfgValue := cCtx.Context.Value(testutils.ConfigContextKey); cfgValue != nil {
			// Use test config from context
			cfg = cfgValue.(*common.ConfigWithContextConfig)
		} else {
			// Load selected context
			context := cCtx.String("context")

			// Load from file if not in context
			var err error
			cfg, err = common.LoadConfigWithContextConfig(context)
			if err != nil {
				return err
			}
		}

		logger.Debug("Project Name: %s", cfg.Config.Project.Name)
		logger.Debug("Building AVS components...")

		// All scripts contained here
		scriptsDir := filepath.Join(".devkit", "scripts")

		// Execute build via .devkit scripts and capture JSON output
		output, err := common.CallTemplateScript(cCtx.Context, logger, dir, filepath.Join(scriptsDir, "build"), common.ExpectJSONResponse)
		if err != nil {
			logger.Error("Build script failed with error: %v", err)
			return fmt.Errorf("build failed: %w", err)
		}

		// Load the context yaml file
		contextPath := filepath.Join("config", "contexts", fmt.Sprintf("%s.yaml", cCtx.String("context")))
		contextNode, err := common.LoadYAML(contextPath)
		if err != nil {
			return fmt.Errorf("failed to load context yaml: %w", err)
		}

		// Get the root node (first content node)
		rootNode := contextNode.Content[0]

		// Get or create the context section
		contextSection := common.GetChildByKey(rootNode, "context")
		if contextSection == nil {
			contextSection = &yaml.Node{Kind: yaml.MappingNode}
			rootNode.Content = append(rootNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "context"},
				contextSection,
			)
		}

		// Update artifacts in context, preserving existing non-empty values
		if err := updateArtifactsFromBuild(contextSection, output); err != nil {
			return fmt.Errorf("failed to update artifacts: %w", err)
		}

		// Write the merged yaml back to file
		if err := common.WriteYAML(contextPath, contextNode); err != nil {
			return fmt.Errorf("failed to write merged yaml: %w", err)
		}

		logger.Info("Build completed successfully")
		return nil
	},
}

// updateArtifactsFromBuild updates the artifacts section with build output, preserving existing non-empty values
func updateArtifactsFromBuild(contextSection *yaml.Node, buildOutput interface{}) error {
	// Convert build output to map for easier access
	outputMap, ok := buildOutput.(map[string]interface{})
	if !ok {
		return fmt.Errorf("build output is not a map")
	}

	// Get or create artifacts section
	artifactsSection := common.GetChildByKey(contextSection, "artifacts")
	if artifactsSection == nil {
		artifactsSection = &yaml.Node{Kind: yaml.MappingNode}
		common.SetMappingValue(contextSection,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "artifacts"},
			artifactsSection)
	}

	// Update artifacts fields from build output, preserving existing non-empty values
	if artifacts, ok := outputMap["artifacts"].(map[string]interface{}); ok {
		for key, value := range artifacts {
			// Skip updating registry_url if it's empty and we already have a non-empty value
			if key == "registry_url" {
				existingValue := common.GetChildByKey(artifactsSection, key)
				if existingValue != nil && existingValue.Value != "" &&
					(value == nil || value == "") {
					continue // Preserve existing non-empty registry_url
				}
			}

			// Convert value to string
			valueStr := ""
			if value != nil {
				valueStr = fmt.Sprintf("%v", value)
			}

			// Update the field
			common.SetMappingValue(artifactsSection,
				&yaml.Node{Kind: yaml.ScalarNode, Value: key},
				&yaml.Node{Kind: yaml.ScalarNode, Value: valueStr})
		}
	}

	return nil
}
