package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// For testability, we'll define interfaces for our dependencies
type templateInfoGetter interface {
	GetInfo() (string, string, string, error)
	GetInfoDefault() (string, string, string, error)
}

// defaultTemplateInfoGetter implements templateInfoGetter using the real functions
type defaultTemplateInfoGetter struct{}

func (g *defaultTemplateInfoGetter) GetInfo() (string, string, string, error) {
	return GetTemplateInfo()
}

func (g *defaultTemplateInfoGetter) GetInfoDefault() (string, string, string, error) {
	return GetTemplateInfoDefault()
}

// gitClientGetter is an interface for getting GitClient instances
type gitClientGetter interface {
	GetClient() template.GitClient
}

// defaultGitClientGetter implements gitClientGetter using the real function
type defaultGitClientGetter struct{}

func (g *defaultGitClientGetter) GetClient() template.GitClient {
	return template.NewGitClient()
}

// createUpgradeCommand creates an upgrade command with the given dependencies
func createUpgradeCommand(
	infoGetter templateInfoGetter,
	clientGetter gitClientGetter,
) *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Upgrade project to a newer template version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "version",
				Usage:    "Template version (Git ref: tag, branch, or commit) to upgrade to",
				Required: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			// Get logger
			log, _ := common.GetLogger()

			// Get the requested version
			requestedVersion := cCtx.String("version")
			if requestedVersion == "" {
				return fmt.Errorf("template version is required. Use --version to specify")
			}

			// Get template information
			projectName, templateBaseURL, currentVersion, err := infoGetter.GetInfo()
			if err != nil {
				return err
			}

			// If the template URL is missing, use the default URL from the getter function
			if templateBaseURL == "" {
				_, templateBaseURL, _, _ = infoGetter.GetInfoDefault()
				if templateBaseURL == "" {
					return fmt.Errorf("no template URL found in config and no default available")
				}
				log.Info("No template URL found in config, using default: %s", templateBaseURL)
			}

			// Get project's absolute path
			absProjectPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current working directory: %w", err)
			}

			// Create temporary directory for cloning the template
			tempDir, err := os.MkdirTemp("", "devkit-template-upgrade-*")
			if err != nil {
				return fmt.Errorf("failed to create temporary directory: %w", err)
			}
			defer os.RemoveAll(tempDir) // Clean up on exit

			log.Info("Upgrading project template:")
			log.Info("  Project: %s", projectName)
			log.Info("  Template URL: %s", templateBaseURL)
			log.Info("  Current version: %s", currentVersion)
			log.Info("  Target version: %s", requestedVersion)
			log.Info("")

			// Extract base URL without .git suffix for consistency
			baseRepoURL := strings.TrimSuffix(templateBaseURL, ".git")

			// Add .git suffix if not present for compatibility with git command
			if !strings.HasSuffix(baseRepoURL, ".git") {
				baseRepoURL = baseRepoURL + ".git"
			}

			// Initialize GitClient
			gitClient := clientGetter.GetClient()

			log.Info("Cloning template repository...")
			// Clone the repository without specifying a branch (we'll checkout after)
			err = gitClient.Clone(cCtx.Context, baseRepoURL, tempDir, template.CloneOptions{
				ProgressCB: func(progress int) {
					if progress%20 == 0 { // Log every 20% progress
						log.Info("Cloning progress: %d%%", progress)
					}
				},
			})
			if err != nil {
				return fmt.Errorf("failed to clone template repository: %w", err)
			}

			log.Info("Checking out version: %s", requestedVersion)
			// Checkout the requested version
			err = gitClient.Checkout(cCtx.Context, tempDir, requestedVersion)
			if err != nil {
				return fmt.Errorf("failed to checkout version %s: %w", requestedVersion, err)
			}

			// Check if the upgrade script exists
			upgradeScriptPath := filepath.Join(tempDir, ".devkit", "scripts", "upgrade")
			if _, err := os.Stat(upgradeScriptPath); os.IsNotExist(err) {
				return fmt.Errorf("upgrade script not found in template version %s", requestedVersion)
			}

			// Make sure the script is executable
			err = os.Chmod(upgradeScriptPath, 0755)
			if err != nil {
				return fmt.Errorf("failed to make upgrade script executable: %w", err)
			}

			log.Info("Running upgrade script...")

			// Execute the upgrade script, passing the project path as an argument
			_, err = common.CallTemplateScript(cCtx.Context, tempDir, upgradeScriptPath, common.ExpectNonJSONResponse, []byte(absProjectPath))
			if err != nil {
				return fmt.Errorf("upgrade script execution failed: %w", err)
			}

			// Update the project's config to reflect the new template version
			configPath := filepath.Join("config", common.BaseConfig)
			configData, err := os.ReadFile(configPath)
			if err != nil {
				return fmt.Errorf("failed to read config file: %w", err)
			}

			var configMap map[string]interface{}
			if err := yaml.Unmarshal(configData, &configMap); err != nil {
				return fmt.Errorf("failed to parse config file: %w", err)
			}

			// Update template version in config
			if configSection, ok := configMap["config"].(map[string]interface{}); ok {
				if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
					// Always update the template version
					projectMap["templateVersion"] = requestedVersion

					// Also set the templateBaseUrl if it's missing
					if _, ok := projectMap["templateBaseUrl"]; !ok {
						// Use the non-.git version for the config
						projectMap["templateBaseUrl"] = strings.TrimSuffix(baseRepoURL, ".git")
						log.Info("Added missing template URL to config")
					}
				}
			}

			// Write updated config
			updatedConfigData, err := yaml.Marshal(configMap)
			if err != nil {
				return fmt.Errorf("failed to marshal updated config: %w", err)
			}

			err = os.WriteFile(configPath, updatedConfigData, 0644)
			if err != nil {
				return fmt.Errorf("failed to write updated config: %w", err)
			}

			log.Info("")
			log.Info("Template upgrade completed successfully!")
			log.Info("Project is now using template version: %s", requestedVersion)

			return nil
		},
	}
}

// UpgradeCommand defines the "template upgrade" subcommand
var UpgradeCommand = createUpgradeCommand(
	&defaultTemplateInfoGetter{},
	&defaultGitClientGetter{},
)
