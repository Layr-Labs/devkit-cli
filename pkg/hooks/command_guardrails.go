package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// ProjectStage represents the current stage of project development
type ProjectStage string

const (
	StageUninitialized ProjectStage = "uninitialized" // No project created yet
	StageCreated       ProjectStage = "created"       // Project scaffolded but not built
	StageBuilt         ProjectStage = "built"         // Contracts compiled
	StageDevnetReady   ProjectStage = "devnet_ready"  // Devnet started and contracts deployed
	StageRunning       ProjectStage = "running"       // AVS components running
)

// CommandDependency defines a command dependency relationship
type CommandDependency struct {
	Command              string                          // Command that has dependencies
	RequiredStage        ProjectStage                    // Minimum stage required to run this command
	ErrorMessage         string                          // Helpful message when dependency isn't met
	PromotesToStage      ProjectStage                    // Stage this command promotes to on success (optional)
	ConditionalPromotion func(*cli.Context) ProjectStage // Function to determine stage promotion based on context
}

// CommandFlowDependencies defines the command flow dependencies
var CommandFlowDependencies = []CommandDependency{
	{
		Command:         "create",
		RequiredStage:   StageUninitialized,
		PromotesToStage: StageCreated,
		ErrorMessage:    "Project already exists. Run commands from within the project directory.",
	},
	{
		Command:         "build",
		RequiredStage:   StageCreated,
		PromotesToStage: StageBuilt,
		ErrorMessage:    "The 'build' command requires a project to be created first. Please run 'devkit avs create <project-name>' first.",
	},
	{
		Command:         "start", // This is the devnet start subcommand
		RequiredStage:   StageCreated,
		PromotesToStage: StageDevnetReady, // Default promotion (will be overridden manually if AVS runs)
		ErrorMessage:    "The 'start' command requires a project to be created first. Please run 'devkit avs create <project-name>' first.",
	},
	{
		Command:         "deploy-contracts", // This is the devnet deploy-contracts subcommand
		RequiredStage:   StageCreated,
		PromotesToStage: StageDevnetReady,
		ErrorMessage:    "The 'deploy-contracts' command requires a project to be created first. Please run 'devkit avs create <project-name>' first.",
	},
	{
		Command:         "stop", // This is the devnet stop subcommand
		RequiredStage:   StageCreated,
		PromotesToStage: StageCreated, // Reset back to created when devnet is stopped
		ErrorMessage:    "The 'stop' command requires a project to be created first.",
	},
	{
		Command:       "call",
		RequiredStage: StageRunning,
		ErrorMessage:  "The 'call' command requires AVS components to be running. Please run 'devkit avs run' or 'devkit avs devnet start' first to start the offchain components.",
	},
	{
		Command:         "run",
		RequiredStage:   StageDevnetReady,
		PromotesToStage: StageRunning,
		ErrorMessage:    "The 'run' command requires contracts to be deployed. Please run 'devkit avs devnet start' or 'devkit avs devnet deploy-contracts' first.",
	},
}

// WithCommandDependencyCheck creates middleware that enforces command dependencies
func WithCommandDependencyCheck(action cli.ActionFunc) cli.ActionFunc {
	return func(cCtx *cli.Context) error {
		cmdName := cCtx.Command.Name

		// Get current project stage
		currentStage, err := getCurrentProjectStage()
		if err != nil {
			// If we can't determine stage, only allow create command
			if cmdName != "create" {
				return fmt.Errorf("unable to determine project stage. Are you in a devkit project directory? Try running 'devkit avs create <project-name>' first")
			}
		}

		// Check dependencies for this command
		dep := findCommandDependency(cmdName)
		if dep != nil {
			if !isStageAllowed(currentStage, dep.RequiredStage) {
				return fmt.Errorf("%s\n\nCurrent stage: %s, Required stage: %s",
					dep.ErrorMessage, currentStage, dep.RequiredStage)
			}
		}

		// Execute the command
		result := action(cCtx)

		// If command succeeded and promotes to a new stage, update the stage
		if result == nil && dep != nil {
			var newStage ProjectStage
			// Use conditional promotion if available, otherwise use default PromotesToStage
			if dep.ConditionalPromotion != nil {
				newStage = dep.ConditionalPromotion(cCtx)
			} else if dep.PromotesToStage != "" {
				newStage = dep.PromotesToStage
			}

			if newStage != "" {
				logger := common.LoggerFromContext(cCtx.Context)

				if cmdName == "create" {
					// For create command, update stage in the newly created project directory
					if err := updateProjectStageForCreate(cCtx, newStage, logger); err != nil {
						logger.Warn("Failed to update project stage for new project: %v", err)
					}
				} else {
					// For other commands, update stage in current directory
					if err := updateProjectStage(newStage, logger); err != nil {
						logger.Warn("Failed to update project stage: %v", err)
					}
				}
			}
		}

		return result
	}
}

// findCommandDependency finds the dependency rule for a command
func findCommandDependency(cmdName string) *CommandDependency {
	for _, dep := range CommandFlowDependencies {
		if dep.Command == cmdName {
			return &dep
		}
	}
	return nil
}

// isStageAllowed checks if the current stage meets the required stage
func isStageAllowed(current, required ProjectStage) bool {
	stageOrder := map[ProjectStage]int{
		StageUninitialized: 0,
		StageCreated:       1,
		StageBuilt:         2,
		StageDevnetReady:   3,
		StageRunning:       4,
	}

	currentLevel, currentOk := stageOrder[current]
	requiredLevel, requiredOk := stageOrder[required]

	if !currentOk || !requiredOk {
		return false
	}

	return currentLevel >= requiredLevel
}

// getCurrentProjectStage determines the current project stage
func getCurrentProjectStage() (ProjectStage, error) {
	// Check if we're in a project directory by looking for config/config.yaml
	configPath := filepath.Join("config", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return StageUninitialized, nil
	}

	// Load the base config to get the current context
	cfg, err := common.LoadBaseConfigYaml()
	if err != nil {
		return StageUninitialized, fmt.Errorf("failed to load project config: %w", err)
	}

	// Load the context config to check for stage
	contextPath := filepath.Join("config", "contexts", cfg.Config.Project.Context+".yaml")
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return StageCreated, nil // Project created but context not fully configured
	}

	// Read the context file and check for stage
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return StageCreated, nil
	}

	var contextWrapper struct {
		Context struct {
			Stage             ProjectStage `yaml:"stage,omitempty"`
			DeployedContracts []struct {
				Name    string `yaml:"name"`
				Address string `yaml:"address"`
			} `yaml:"deployed_contracts,omitempty"`
		} `yaml:"context"`
	}

	if err := yaml.Unmarshal(data, &contextWrapper); err != nil {
		return StageCreated, nil
	}

	// If stage is explicitly set, use it
	if contextWrapper.Context.Stage != "" {
		return contextWrapper.Context.Stage, nil
	}

	// Otherwise, infer stage from project state
	return inferStageFromProjectState(&contextWrapper)
}

// inferStageFromProjectState infers the stage based on project artifacts
func inferStageFromProjectState(contextWrapper *struct {
	Context struct {
		Stage             ProjectStage `yaml:"stage,omitempty"`
		DeployedContracts []struct {
			Name    string `yaml:"name"`
			Address string `yaml:"address"`
		} `yaml:"deployed_contracts,omitempty"`
	} `yaml:"context"`
}) (ProjectStage, error) {
	// Check if contracts are deployed
	if len(contextWrapper.Context.DeployedContracts) > 0 {
		// Check if any contract has a valid address
		for _, contract := range contextWrapper.Context.DeployedContracts {
			if contract.Address != "" && strings.HasPrefix(contract.Address, "0x") {
				return StageDevnetReady, nil
			}
		}
	}

	// Check if build artifacts exist
	if _, err := os.Stat("contracts/out"); err == nil {
		return StageBuilt, nil
	}

	// Default to created stage
	return StageCreated, nil
}

// updateProjectStage updates the project stage in the context file
func updateProjectStage(newStage ProjectStage, logger iface.Logger) error {
	// Load the base config to get the current context
	cfg, err := common.LoadBaseConfigYaml()
	if err != nil {
		return fmt.Errorf("failed to load project config: %w", err)
	}

	contextPath := filepath.Join("config", "contexts", cfg.Config.Project.Context+".yaml")

	// Load the existing context as YAML nodes to preserve formatting
	rootNode, err := common.LoadYAML(contextPath)
	if err != nil {
		return fmt.Errorf("failed to load context YAML: %w", err)
	}

	if len(rootNode.Content) == 0 {
		return fmt.Errorf("empty YAML root node")
	}

	// Navigate to the context node
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in context file")
	}

	// Update or add the stage field
	stageNode := common.GetChildByKey(contextNode, "stage")
	if stageNode != nil {
		// Update existing stage
		stageNode.Value = string(newStage)
	} else {
		// Add new stage field
		stageKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "stage"}
		stageValue := &yaml.Node{Kind: yaml.ScalarNode, Value: string(newStage)}
		contextNode.Content = append(contextNode.Content, stageKey, stageValue)
	}

	// Write the updated YAML back
	return common.WriteYAML(contextPath, rootNode)
}

// updateProjectStageForCreate updates the project stage in the context file for a newly created project
func updateProjectStageForCreate(cCtx *cli.Context, newStage ProjectStage, logger iface.Logger) error {
	// Get the project name and target directory from create command arguments
	if cCtx.NArg() == 0 {
		return fmt.Errorf("project name is required for create command")
	}

	projectName := cCtx.Args().First()
	dest := cCtx.Args().Get(1)

	// Use dest from dir flag or positional
	var targetDir string
	if dest != "" {
		targetDir = dest
	} else {
		targetDir = cCtx.String("dir")
	}

	// Ensure provided dir is absolute
	targetDir, err := filepath.Abs(filepath.Join(targetDir, projectName))
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for target directory: %w", err)
	}

	// Load the base config from the target project directory
	configPath := filepath.Join(targetDir, "config", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read project config: %w", err)
	}

	var cfg struct {
		Config struct {
			Project struct {
				Context string `yaml:"context"`
			} `yaml:"project"`
		} `yaml:"config"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse project config: %w", err)
	}

	// Build context path in the target directory
	contextPath := filepath.Join(targetDir, "config", "contexts", cfg.Config.Project.Context+".yaml")

	// Load the existing context as YAML nodes to preserve formatting
	rootNode, err := common.LoadYAML(contextPath)
	if err != nil {
		return fmt.Errorf("failed to load context YAML: %w", err)
	}

	if len(rootNode.Content) == 0 {
		return fmt.Errorf("empty YAML root node")
	}

	// Navigate to the context node
	contextNode := common.GetChildByKey(rootNode.Content[0], "context")
	if contextNode == nil {
		return fmt.Errorf("missing 'context' key in context file")
	}

	// Update or add the stage field
	stageNode := common.GetChildByKey(contextNode, "stage")
	if stageNode != nil {
		// Update existing stage
		stageNode.Value = string(newStage)
	} else {
		// Add new stage field
		stageKey := &yaml.Node{Kind: yaml.ScalarNode, Value: "stage"}
		stageValue := &yaml.Node{Kind: yaml.ScalarNode, Value: string(newStage)}
		contextNode.Content = append(contextNode.Content, stageKey, stageValue)
	}

	// Write the updated YAML back
	return common.WriteYAML(contextPath, rootNode)
}

// UpdateProjectStage manually updates the project stage - can be called from command implementations
func UpdateProjectStage(newStage ProjectStage, logger iface.Logger) error {
	return updateProjectStage(newStage, logger)
}
