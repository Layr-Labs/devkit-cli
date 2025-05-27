package template

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/template"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// GetTemplateInfo reads the template information from the project config
// Returns projectName, templateBaseURL, templateVersion, error
func GetTemplateInfo() (string, string, string, error) {
	// Ensure we're in a project directory (check for config/config.yaml)
	configPath := filepath.Join("config", common.BaseConfig)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", "", "", fmt.Errorf("config/config.yaml not found. Make sure you're in a devkit project directory")
	}

	// Read the config file to get the template URL
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read config file: %w", err)
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return "", "", "", fmt.Errorf("failed to parse config file: %w", err)
	}

	// Extract project name and template info
	projectName := ""
	templateBaseURL := ""
	templateVersion := "unknown" // Default version

	if configSection, ok := configMap["config"].(map[string]interface{}); ok {
		if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
			if name, ok := projectMap["name"].(string); ok {
				projectName = name
			}
			if url, ok := projectMap["templateBaseUrl"].(string); ok {
				templateBaseURL = url
			}
			if version, ok := projectMap["templateVersion"].(string); ok {
				templateVersion = version
			}
		}
	}

	// If no template URL was found in the config, use the default from templates.yaml
	if templateBaseURL == "" {
		// Load templates configuration
		templateConfig, err := template.LoadConfig()
		if err == nil {
			// Default to "task" architecture and "go" language
			defaultArch := "task"
			defaultLang := "go"

			// Look up the default template URL
			mainBaseURL, _, _ := template.GetTemplateURLs(templateConfig, defaultArch, defaultLang)

			// Use the default values
			templateBaseURL = mainBaseURL
		}

		// If we still don't have a URL, use a hardcoded fallback
		if templateBaseURL == "" {
			templateBaseURL = "https://github.com/Layr-Labs/hourglass-avs-template"
		}
	}

	return projectName, templateBaseURL, templateVersion, nil
}

// GetTemplateInfoDefault returns default template information without requiring a config file
// Returns projectName, templateBaseURL, templateVersion, error
func GetTemplateInfoDefault() (string, string, string, error) {
	// Default values
	projectName := ""
	templateBaseURL := ""
	templateVersion := "unknown"

	// Try to load templates configuration
	templateConfig, err := template.LoadConfig()
	if err == nil {
		// Default to "task" architecture and "go" language
		defaultArch := "task"
		defaultLang := "go"

		// Look up the default template URL
		mainBaseURL, _, _ := template.GetTemplateURLs(templateConfig, defaultArch, defaultLang)

		// Use the default values
		templateBaseURL = mainBaseURL
	}

	// If we still don't have a URL, use a hardcoded fallback
	if templateBaseURL == "" {
		templateBaseURL = "https://github.com/Layr-Labs/hourglass-avs-template"
	}

	return projectName, templateBaseURL, templateVersion, nil
}

// Command defines the main "template" command for template operations
var Command = &cli.Command{
	Name:  "template",
	Usage: "Manage project templates",
	Subcommands: []*cli.Command{
		InfoCommand,
		UpgradeCommand,
	},
}
