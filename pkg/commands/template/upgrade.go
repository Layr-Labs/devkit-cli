package template

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// gitCloneRepo clones the repository without specifying a branch
var gitCloneRepo = func(ctx context.Context, repoURL, targetDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, targetDir)
	return cmd.Run()
}

// gitFetch runs git fetch to update refs
var gitFetch = func(ctx context.Context, repoDir string) error {
	cmd := exec.CommandContext(ctx, "git", "fetch")
	cmd.Dir = repoDir
	return cmd.Run()
}

// gitCheckout checks out the specified reference
var gitCheckout = func(ctx context.Context, repoDir, version string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", version)
	cmd.Dir = repoDir
	return cmd.Run()
}

// UpgradeCommand defines the "template upgrade" subcommand
var UpgradeCommand = &cli.Command{
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
		projectName, templateBaseURL, currentVersion, err := GetTemplateInfo()
		if err != nil {
			return err
		}

		// If the template URL is missing, use the default URL from the getter function
		if templateBaseURL == "" {
			_, templateBaseURL, _, _ = GetTemplateInfoDefault()
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

		log.Info("Cloning template repository...")
		// 1. Clone the repository (full clone)
		err = gitCloneRepo(cCtx.Context, baseRepoURL, tempDir)
		if err != nil {
			return fmt.Errorf("failed to clone template repository: %w", err)
		}

		log.Info("Fetching latest refs from remote...")
		// 2. Fetch to ensure we have all refs
		err = gitFetch(cCtx.Context, tempDir)
		if err != nil {
			return fmt.Errorf("failed to fetch refs: %w", err)
		}

		log.Info("Checking out version: %s", requestedVersion)
		// 3. Checkout the requested version
		err = gitCheckout(cCtx.Context, tempDir, requestedVersion)
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
