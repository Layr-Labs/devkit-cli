package template

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	progresslogger "github.com/Layr-Labs/devkit-cli/pkg/common/logger"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// For testability, we'll define interfaces for our dependencies
type templateInfoGetter interface {
	GetInfo() (string, string, string, error)
	GetInfoDefault() (string, string, string, error)
	GetTemplateVersionFromConfig(arch, lang string) (string, error)
}

// defaultTemplateInfoGetter implements templateInfoGetter using the real functions
type defaultTemplateInfoGetter struct{}

func (g *defaultTemplateInfoGetter) GetInfo() (string, string, string, error) {
	return GetTemplateInfo()
}

func (g *defaultTemplateInfoGetter) GetInfoDefault() (string, string, string, error) {
	return GetTemplateInfoDefault()
}

func (g *defaultTemplateInfoGetter) GetTemplateVersionFromConfig(arch, lang string) (string, error) {
	cfg, err := template.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load templates cfg: %w", err)
	}
	a, ok := cfg.Architectures[arch]
	if !ok {
		return "", fmt.Errorf("architecture %s not found", arch)
	}
	l, ok := a.Languages[lang]
	if !ok {
		return "", fmt.Errorf("language %s not found under architecture %s", lang, arch)
	}
	return l.Version, nil
}

// createUpgradeCommand creates an upgrade command with the given dependencies
func createUpgradeCommand(
	infoGetter templateInfoGetter,
) *cli.Command {
	return &cli.Command{
		Name:  "upgrade",
		Usage: "Upgrade project to a newer template version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "version",
				Usage: "Template version (Git ref: tag, branch, or commit) to upgrade to",
				Value: "latest",
			},
			&cli.StringFlag{
				Name:  "lang",
				Usage: "Programming language used to generate project files",
				Value: "go",
			},
			&cli.StringFlag{
				Name:  "arch",
				Usage: "AVS architecture used to generate project files (task-based/hourglass, epoch-based, etc.)",
				Value: "task",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// Get logger
			logger := common.LoggerFromContext(cCtx.Context)
			tracker := common.ProgressTrackerFromContext(cCtx.Context)

			arch := cCtx.String("arch")
			lang := cCtx.String("lang")
			latestVersion, err := infoGetter.GetTemplateVersionFromConfig(arch, lang)
			if err != nil {
				return fmt.Errorf("failed to get latest version: %w", err)
			}

			// Get the requested version
			requestedVersion := cCtx.String("version")
			if requestedVersion == "" || requestedVersion == "latest" {
				// Set requestedVersion to configs version (this is the latest this version of devkit is aware of)
				requestedVersion = latestVersion
			}
			// Check again for nil requestedVersion after attempting to pull from template
			if requestedVersion == "" {
				return fmt.Errorf("template version is required. Use --version to specify")
			}

			// Check if the requested version is valid and known to DevKit
			if common.IsSemver(requestedVersion) && common.IsSemver(latestVersion) {
				// Compare semver strings
				requestedIsBeyondKnown, err := common.CompareVersions(requestedVersion, latestVersion)

				// On error log but don't exit
				if err != nil {
					logger.Error("comparing versions failed: %w", err)
				}
				// Return error and prevent upgrade
				if requestedIsBeyondKnown {
					return fmt.Errorf("requested version is greater than the latest version known to DevKit (%s)", latestVersion)
				}
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
				logger.Info("No template URL found in config, using default: %s", templateBaseURL)
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

			tempCacheDir, err := os.MkdirTemp("", "devkit-template-cache-*")
			if err != nil {
				return fmt.Errorf("failed to create temporary cache directory: %w", err)
			}
			defer os.RemoveAll(tempCacheDir) // Clean up on exit

			logger.Info("Upgrading project template:")
			logger.Info("  Project: %s", projectName)
			logger.Info("  Template URL: %s", templateBaseURL)
			logger.Info("  Current version: %s", currentVersion)
			logger.Info("  Target version: %s", requestedVersion)
			logger.Info("")

			// Extract base URL without .git suffix for consistency
			baseRepoURL := strings.TrimSuffix(templateBaseURL, ".git")

			// Fetch main template
			fetcher := &template.GitFetcher{
				Client: template.NewGitClient(),
				Logger: *progresslogger.NewProgressLogger(
					logger,
					tracker,
				),
				Config: template.GitFetcherConfig{
					Verbose: cCtx.Bool("verbose"),
				},
			}
			logger.Info("Cloning template repository...")
			if err := fetcher.Fetch(cCtx.Context, baseRepoURL, requestedVersion, tempDir); err != nil {
				return fmt.Errorf("failed to fetch template from %s with version %s: %w", baseRepoURL, requestedVersion, err)
			}

			// Check if the upgrade script exists
			upgradeScriptPath := filepath.Join(tempDir, ".devkit", "scripts", "upgrade")
			if _, err := os.Stat(upgradeScriptPath); os.IsNotExist(err) {
				return fmt.Errorf("upgrade script not found in template version %s", requestedVersion)
			}

			logger.Info("Running upgrade script...")

			// Execute the upgrade script, passing the project path as an argument
			_, err = common.CallTemplateScript(cCtx.Context, logger, tempDir, upgradeScriptPath, common.ExpectNonJSONResponse, []byte(absProjectPath), []byte(currentVersion), []byte(requestedVersion))
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
						logger.Info("Added missing template URL to config")
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

			logger.Info("")
			logger.Info("Template upgrade completed successfully!")
			logger.Info("Project is now using template version: %s", requestedVersion)

			return nil
		},
	}
}

// UpgradeCommand defines the "template upgrade" subcommand
var UpgradeCommand = createUpgradeCommand(
	&defaultTemplateInfoGetter{},
)
