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
// Returns projectName, templateBaseURL, templateVersion, templateLanguage, error
func GetTemplateInfo() (string, string, string, string, error) {
	// Check for config file
	configPath := filepath.Join("config", common.BaseConfig)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", "", "", "", fmt.Errorf("config/config.yaml not found. Make sure you're in a devkit project directory")
	}

	// Read and parse config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to read config file: %w", err)
	}

	var configMap map[string]interface{}
	if err := yaml.Unmarshal(configData, &configMap); err != nil {
		return "", "", "", "", fmt.Errorf("failed to parse config file: %w", err)
	}

	// Extract values with defaults
	projectName := ""
	templateBaseURL := ""
	templateVersion := "unknown"
	templateLanguage := "go"

	// Navigate to config.project section and extract values
	if config, ok := configMap["config"].(map[string]interface{}); ok {
		if project, ok := config["project"].(map[string]interface{}); ok {
			projectName, _ = project["name"].(string)
			templateBaseURL, _ = project["templateBaseUrl"].(string)
			templateVersion = getStringOrDefault(project, "templateVersion", templateVersion)
			templateLanguage = getStringOrDefault(project, "templateLanguage", templateLanguage)
		}
	}

	// Use defaults if templateBaseURL is empty
	if templateBaseURL == "" {
		templateBaseURL = "https://github.com/Layr-Labs/hourglass-avs-template"

		// Try to get from template config (optional)
		if cfg, err := template.LoadConfig(); err == nil {
			if url, _, _ := template.GetTemplateURLs(cfg, "hourglass", "go"); url != "" {
				templateBaseURL = url
			}
		}
	}

	return projectName, templateBaseURL, templateVersion, templateLanguage, nil
}

// GetTemplateInfoDefault returns default template information without requiring a config file
// Returns projectName, templateBaseURL, templateVersion, error
func GetTemplateInfoDefault() (string, string, string, string, error) {
	// Default values
	projectName := ""
	templateBaseURL := ""
	templateVersion := ""
	templateLanguage := "go"

	// Try to load templates configuration
	templateConfig, err := template.LoadConfig()
	if err == nil {
		// Default to "hourglass" framework and "go" language
		defaultFramework := "hourglass"
		defaultLang := "go"

		// Look up the default template URL
		mainBaseURL, _, _ := template.GetTemplateURLs(templateConfig, defaultFramework, defaultLang)

		// Use the default values
		templateBaseURL = mainBaseURL
	}

	return projectName, templateBaseURL, templateVersion, templateLanguage, nil
}

// Helper function to get string value or return default
func getStringOrDefault(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
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
