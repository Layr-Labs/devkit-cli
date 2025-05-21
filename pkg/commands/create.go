package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	project "github.com/Layr-Labs/devkit-cli"
	"github.com/Layr-Labs/devkit-cli/config"
	"github.com/Layr-Labs/devkit-cli/config/configs"
	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/template"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// CreateCommand defines the "create" command
var CreateCommand = &cli.Command{
	Name:      "create",
	Usage:     "Initializes a new AVS project scaffold (Hourglass model)",
	ArgsUsage: "<project-name> [target-dir]",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Usage: "Set output directory for the new project",
			Value: ".",
		},
		&cli.StringFlag{
			Name:  "lang",
			Usage: "Programming language to generate project files",
			Value: "go",
		},
		&cli.StringFlag{
			Name:  "arch",
			Usage: "Specifies AVS architecture (task-based/hourglass, epoch-based, etc.)",
			Value: "task",
		},
		&cli.StringFlag{
			Name:  "template-url",
			Usage: "Direct GitHub base URL to use as template (overrides templates.yml)",
		},
		&cli.StringFlag{
			Name:  "template-version",
			Usage: "Git ref (tag, commit, branch) for the template",
		},
		&cli.StringFlag{
			Name:  "env",
			Usage: "Chooses the environment (local, testnet, mainnet)",
			Value: "local",
		},
		&cli.BoolFlag{
			Name:  "overwrite",
			Usage: "Force overwrite if project directory already exists",
		},
		&cli.BoolFlag{
			Name:  "no-cache",
			Usage: "Disable the use of caching mechanisms",
			Value: false,
		},
		&cli.IntFlag{
			Name:  "depth",
			Usage: "Maximum submodule recursion depth",
			Value: -1,
		},
		&cli.IntFlag{
			Name:  "retries",
			Usage: "Maximum number of retries on submodule clone failure",
			Value: 3,
		},
		&cli.IntFlag{
			Name:  "concurrency",
			Usage: "Maximum number of concurrent submodule clones",
			Value: 8,
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// exit early if no project name is provided
		if cCtx.NArg() == 0 {
			return fmt.Errorf("project name is required\nUsage: avs create <project-name> [flags]")
		}
		projectName := cCtx.Args().First()
		dest := cCtx.Args().Get(1)

		// get logger
		log, tracker := common.GetLogger()

		// use dest from dir flag or positional
		var targetDir string
		if dest != "" {
			targetDir = dest
		} else {
			targetDir = cCtx.String("dir")
		}

		// ensure provided dir is absolute
		targetDir, err := filepath.Abs(filepath.Join(targetDir, projectName))
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for target directory: %w", err)
		}

		// in verbose mode, detail the situation
		if cCtx.Bool("verbose") {
			log.Info("Creating new AVS project: %s", projectName)
			log.Info("Directory: %s", cCtx.String("dir"))
			log.Info("Language: %s", cCtx.String("lang"))
			log.Info("Architecture: %s", cCtx.String("arch"))
			log.Info("Environment: %s", cCtx.String("env"))
			if cCtx.String("template-url") != "" {
				log.Info("Template URL: %s", cCtx.String("template-url"))
			}
		}

		// Get template URLs
		mainBaseURL, mainVersion, contractsBaseURL, contractsVersion, err := getTemplateURLs(cCtx)
		if err != nil {
			return err
		}

		// Create project directories
		if err := createProjectDir(targetDir, cCtx.Bool("overwrite"), cCtx.Bool("verbose")); err != nil {
			return err
		}

		// Construct the full URLs with version information
		mainFullURL := mainBaseURL
		if mainVersion != "" {
			// Ensure URL doesn't end with .git before appending
			mainBaseURL = strings.TrimSuffix(mainBaseURL, ".git")
			mainFullURL = fmt.Sprintf("%s/tree/%s", mainBaseURL, mainVersion)
		}

		contractsFullURL := ""
		if contractsBaseURL != "" {
			contractsFullURL = contractsBaseURL
			if contractsVersion != "" {
				// Ensure URL doesn't end with .git before appending
				contractsBaseURL = strings.TrimSuffix(contractsBaseURL, ".git")
				contractsFullURL = fmt.Sprintf("%s/tree/%s", contractsBaseURL, contractsVersion)
			}
		}

		if cCtx.Bool("verbose") {
			log.Info("Using template: %s", mainFullURL)
			if contractsFullURL != "" {
				log.Info("Using contracts template: %s", contractsFullURL)
			}
			if mainVersion != "" {
				log.Info("Template version: %s", mainVersion)
			}
			if contractsVersion != "" && contractsFullURL != "" {
				log.Info("Contracts template version: %s", contractsVersion)
			}
		}

		// Set Cache location as ~/.devkit
		basePath := filepath.Join(os.Getenv("HOME"), ".devkit")

		// Fetch main template
		fetcher := &template.GitFetcher{
			Git:   template.NewGitClient(),
			Cache: template.NewGitRepoCache(basePath),
			Logger: *logger.NewProgressLogger(
				log,
				tracker,
			),
			Config: template.GitFetcherConfig{
				CacheDir:       basePath,
				MaxDepth:       cCtx.Int("depth"),
				MaxRetries:     cCtx.Int("retries"),
				MaxConcurrency: cCtx.Int("concurrency"),
				UseCache:       !cCtx.Bool("no-cache"),
				Verbose:        cCtx.Bool("verbose"),
			},
		}
		if err := fetcher.Fetch(cCtx.Context, mainFullURL, targetDir); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", mainFullURL, err)
		}

		// Check for contracts template and fetch if missing
		if contractsFullURL != "" {
			contractsDir := filepath.Join(targetDir, common.ContractsDir)
			contractsDirReadme := filepath.Join(contractsDir, "README.md")

			// Fetch the contracts directory if it does not exist in the template
			if _, err := os.Stat(contractsDirReadme); os.IsNotExist(err) {
				if err := fetcher.Fetch(cCtx.Context, contractsFullURL, contractsDir); err != nil {
					log.Warn("Failed to fetch contracts template '%s': %v", contractsFullURL, err)
				}
			}
		}

		// Copy DevKit README.md to templates README.md
		readMePath := filepath.Join(targetDir, "README.md")
		readMeTemplate, err := os.ReadFile(readMePath)
		if err != nil {
			log.Warn("Project README.md is missing: %w", err)
		}
		readMeTemplate = append(readMeTemplate, project.RawReadme...)
		err = os.WriteFile(readMePath, readMeTemplate, 0644)
		if err != nil {
			return fmt.Errorf("failed to write README.md: %w", err)
		}

		// Set path for .devkit scripts
		scriptDir := filepath.Join(".devkit", "scripts")
		scriptPath := filepath.Join(scriptDir, "init")

		// Run init to install deps
		log.Info("Installing template dependencies\n\n")

		// Run init on the template init script
		if _, err = common.CallTemplateScript(cCtx.Context, targetDir, scriptPath, common.ExpectNonJSONResponse, nil); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", scriptPath, err)
		}

		// Tidy the logs
		if cCtx.Bool("verbose") {
			log.Info("\nFinalising new project\n\n")
		}

		// Copy config.yaml to the project directory
		if err := copyDefaultConfigToProject(targetDir, projectName, mainBaseURL, mainVersion, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", common.BaseConfig, err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(targetDir, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize keystores: %w", err)
		}

		// Write the example .env file
		err = os.WriteFile(filepath.Join(targetDir, ".env.example"), []byte(config.EnvExample), 0644)
		if err != nil {
			return fmt.Errorf("failed to write .env.example: %w", err)
		}

		// Save project settings with telemetry preference
		appEnv, ok := common.AppEnvironmentFromContext(cCtx.Context)
		if !ok {
			return fmt.Errorf("could not determine application environment")
		}
		if err := common.SaveProjectIdAndTelemetryToggle(targetDir, appEnv.ProjectUUID, true); err != nil {
			return fmt.Errorf("failed to save project settings: %w", err)
		}

		// Initialize git repository in the project directory
		if err := initGitRepo(cCtx, targetDir, cCtx.Bool("verbose")); err != nil {
			log.Warn("Failed to initialize Git repository in %s: %v", targetDir, err)
		}

		log.Info("\nProject %s created successfully in %s. Run 'cd %s' to get started.", projectName, targetDir, targetDir)
		return nil
	},
}

func getTemplateURLs(cCtx *cli.Context) (string, string, string, string, error) {
	templateBaseOverride := cCtx.String("template-url")
	templateVersionOverride := cCtx.String("template-version")

	if templateBaseOverride != "" {
		// Use the override values
		return templateBaseOverride, templateVersionOverride, "", "", nil
	}

	config, err := template.LoadConfig()
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to load templates config: %w", err)
	}

	arch := cCtx.String("arch")
	lang := cCtx.String("lang")

	mainBaseURL, mainVersion, contractsBaseURL, contractsVersion, err := template.GetTemplateURLs(config, arch, lang)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get template URLs: %w", err)
	}

	if mainBaseURL == "" {
		return "", "", "", "", fmt.Errorf("no template found for architecture %s and language %s", arch, lang)
	}

	// If templateVersionOverride is provided, it takes precedence over the version from templates.yaml
	if templateVersionOverride != "" {
		mainVersion = templateVersionOverride
	}

	return mainBaseURL, mainVersion, contractsBaseURL, contractsVersion, nil
}

func createProjectDir(targetDir string, overwrite, verbose bool) error {
	// get logger
	log, _ := common.GetLogger()

	// Check if directory exists and handle overwrite
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {

		if !overwrite {
			return fmt.Errorf("directory %s already exists. Use --overwrite flag to force overwrite", targetDir)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
		if verbose {
			log.Info("Removed existing directory: %s", targetDir)
		}
	}

	// Create main project directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	return nil
}

// copyDefaultConfigToProject copies config to the project directory with updated project name
func copyDefaultConfigToProject(targetDir, projectName string, templateBaseURL, templateVersion string, verbose bool) error {
	// get logger
	log, _ := common.GetLogger()

	// Create and ensure target config directory exists
	destConfigDir := filepath.Join(targetDir, "config")
	if err := os.MkdirAll(destConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create target config directory: %w", err)
	}

	// Read config.yaml from config embed
	configContent := configs.ConfigYamls[configs.LatestVersion]

	// Unmarshal the YAML content into a map
	var configMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(configContent), &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}

	// Access the project section
	if configSection, ok := configMap["config"].(map[string]interface{}); ok {
		if projectMap, ok := configSection["project"].(map[string]interface{}); ok {
			// Update project name
			projectMap["name"] = projectName

			// Add template information if provided
			if templateBaseURL != "" {
				projectMap["templateBaseUrl"] = templateBaseURL
			}
			if templateVersion != "" {
				projectMap["templateVersion"] = templateVersion
			}
		}
	}

	// Marshal the modified configuration back to YAML
	newContentBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal modified config: %w", err)
	}

	// Write the updated config
	err = os.WriteFile(filepath.Join(destConfigDir, common.BaseConfig), newContentBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", common.BaseConfig, err)
	}

	if verbose {
		log.Info("Created config/%s in project directory", common.BaseConfig)
	}

	// Copy all context files
	destContextsDir := filepath.Join(destConfigDir, "contexts")
	if err := os.MkdirAll(destContextsDir, 0755); err != nil {
		return fmt.Errorf("failed to create target contexts directory: %w", err)
	}
	// copy latest version of context to project for default contexts
	for _, name := range contexts.DefaultContexts {
		content := contexts.ContextYamls[contexts.LatestVersion]
		entryName := fmt.Sprintf("%s.yaml", name)

		err := os.WriteFile(filepath.Join(destContextsDir, entryName), []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", entryName, err)
		}

		if verbose {
			log.Info("Copied context file: %s", entryName)
		}
	}

	return nil
}

// / Creates a keystores directory with default keystore json files
func copyDefaultKeystoresToProject(targetDir string, verbose bool) error {
	log, _ := common.GetLogger()

	// Construct keystore dest
	destKeystoreDir := filepath.Join(targetDir, "keystores")

	// Create the destination keystore directory
	if err := os.MkdirAll(destKeystoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create keystores directory: %w", err)
	}
	if verbose {
		log.Info("Created directory: %s", destKeystoreDir)
	}

	// Read files embedded keystore
	files := config.KeystoreEmbeds

	// Write files to destKeystoreDir
	for fileName, file := range files {
		destPath := filepath.Join(destKeystoreDir, fileName)
		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination keystore file %s: %w", destPath, err)
		}
		defer destFile.Close()

		if err := os.WriteFile(destPath, []byte(file), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fileName, err)
		}

		if verbose {
			log.Info("Copied keystore: %s", fileName)
		}
	}

	return nil
}

// initGitRepo initializes a new Git repository in the target directory.
func initGitRepo(ctx *cli.Context, targetDir string, verbose bool) error {
	// get logger
	log, _ := common.GetLogger()

	if verbose {
		log.Info("Removing existing .git directory in %s (if any)...", targetDir)
	}
	gitDir := filepath.Join(targetDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove existing .git directory: %w", err)
	}

	if verbose {
		log.Info("Initializing Git repository in %s...", targetDir)
	}

	cmd := exec.CommandContext(ctx.Context, "git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}

	err = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(config.GitIgnore), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	if verbose {
		log.Info("Git repository initialized successfully.")
		if len(output) > 0 {
			log.Info("Git init output: \"%s\"", strings.Trim(string(output), "\n"))
		}
	}
	return nil
}
