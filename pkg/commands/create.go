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
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	progresslogger "github.com/Layr-Labs/devkit-cli/pkg/common/logger"
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
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// exit early if no project name is provided
		if cCtx.NArg() == 0 {
			return fmt.Errorf("project name is required\nUsage: avs create <project-name> [flags]")
		}
		projectName := cCtx.Args().First()
		dest := cCtx.Args().Get(1)

		// get logger and tracker
		logger := common.LoggerFromContext(cCtx.Context)
		tracker := common.ProgressTrackerFromContext(cCtx.Context)

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
			logger.InfoWithActor("User", "Creating new AVS project: %s", projectName)
			logger.InfoWithActor("User", "Directory: %s", cCtx.String("dir"))
			logger.InfoWithActor("User", "Language: %s", cCtx.String("lang"))
			logger.InfoWithActor("User", "Architecture: %s", cCtx.String("arch"))
			logger.InfoWithActor("User", "Environment: %s", cCtx.String("env"))
			if cCtx.String("template-url") != "" {
				logger.InfoWithActor("User", "Template URL: %s", cCtx.String("template-url"))
			}
		}

		// Get template URLs
		mainBaseURL, mainVersion, err := getTemplateURLs(cCtx)
		if err != nil {
			return err
		}

		// Create project directories
		if err := createProjectDir(logger, targetDir, cCtx.Bool("overwrite"), cCtx.Bool("verbose")); err != nil {
			return err
		}

		if cCtx.Bool("verbose") {
			logger.InfoWithActor("User", "Using template: %s", mainBaseURL)
			if mainVersion != "" {
				logger.InfoWithActor("User", "Template version: %s", mainVersion)
			}
		}

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
		if err := fetcher.Fetch(cCtx.Context, mainBaseURL, mainVersion, targetDir); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", mainBaseURL, err)
		}

		// Copy DevKit README.md to templates README.md
		readMePath := filepath.Join(targetDir, "README.md")
		readMeTemplate, err := os.ReadFile(readMePath)
		if err != nil {
			log.WarnWithActor("System", "Project README.md is missing: %w", err)
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
		logger.InfoWithActor("User", "Installing template dependencies\n\n")

		// Run init on the template init script
		if _, err = common.CallTemplateScript(cCtx.Context, logger, targetDir, scriptPath, common.ExpectNonJSONResponse, nil); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", scriptPath, err)
		}

		// Tidy the logs
		if cCtx.Bool("verbose") {
			logger.InfoWithActor("User", "\nFinalising new project\n\n")
		}

		// Copy config.yaml to the project directory
		if err := copyDefaultConfigToProject(logger, targetDir, projectName, mainBaseURL, mainVersion); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", common.BaseConfig, err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(logger, targetDir); err != nil {
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
		if err := initGitRepo(cCtx, targetDir, logger); err != nil {
			logger.WarnWithActor("User", "Failed to initialize Git repository in %s: %v", targetDir, err)
		}

		logger.Info("\nProject %s created successfully in %s. Run 'cd %s' to get started.", projectName, targetDir, targetDir)
		return nil
	},
}

func getTemplateURLs(cCtx *cli.Context) (string, string, error) {
	templateBaseOverride := cCtx.String("template-url")
	templateVersionOverride := cCtx.String("template-version")

	cfg, err := template.LoadConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to load templates cfg: %w", err)
	}

	arch := cCtx.String("arch")
	lang := cCtx.String("lang")

	mainBaseURL, mainVersion, err := template.GetTemplateURLs(cfg, arch, lang)
	if err != nil {
		return "", "", fmt.Errorf("failed to get template URLs: %w", err)
	}
	if templateBaseOverride != "" {
		mainBaseURL = templateBaseOverride
	}
	if mainBaseURL == "" {
		return "", "", fmt.Errorf("no template found for architecture %s and language %s", arch, lang)
	}

	// If templateVersionOverride is provided, it takes precedence over the version from templates.yaml
	if templateVersionOverride != "" {
		mainVersion = templateVersionOverride
	}

	return mainBaseURL, mainVersion, nil
}

func createProjectDir(logger iface.Logger, targetDir string, overwrite, verbose bool) error {
	// Check if directory exists and handle overwrite
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		if !overwrite {
			return fmt.Errorf("directory %s already exists. Use --overwrite flag to force overwrite", targetDir)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
		if verbose {
			logger.InfoWithActor("User", "Removed existing directory: %s", targetDir)
		}
	}
	// Create main project directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	return nil
}

// copyDefaultConfigToProject copies config to the project directory with updated project name
func copyDefaultConfigToProject(logger iface.Logger, targetDir, projectName string, templateBaseURL, templateVersion string) error {
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

	logger.InfoWithActor("User", "Created config/%s in project directory", common.BaseConfig)

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
			log.InfoWithActor("User", "Copied context file: %s", entryName)
		}
	}

	return nil
}

// / Creates a keystores directory with default keystore json files
func copyDefaultKeystoresToProject(logger iface.Logger, targetDir string) error {
	// Construct keystore dest
	destKeystoreDir := filepath.Join(targetDir, "keystores")

	// Create the destination keystore directory
	if err := os.MkdirAll(destKeystoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create keystores directory: %w", err)
	}
	if verbose {
		log.InfoWithActor("User", "Created directory: %s", destKeystoreDir)
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
			log.InfoWithActor("User", "Copied keystore: %s", fileName)
		}
	}

	return nil
}

const contractsBasePath = ".devkit/contracts"

// initGitRepo initializes a new Git repository in the target directory.
func initGitRepo(ctx *cli.Context, targetDir string, logger iface.Logger) error {
	logger.Debug("Removing existing .git directory in %s (if any)...", targetDir)

	// remove the old .git dir
	if verbose {
		log.InfoWithActor("User", "Removing existing .git directory in %s (if any)...", targetDir)
	}
	gitDir := filepath.Join(targetDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove existing .git directory: %w", err)
	}

	// init a new .git repo
	if verbose {
		log.InfoWithActor("User", "Initializing Git repository in %s...", targetDir)
	}
	cmd := exec.CommandContext(ctx.Context, "git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}

	// write a .gitignore into the new dir
	err = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(config.GitIgnore), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// add all changes and commit
	cmd = exec.CommandContext(ctx.Context, "git", "add", ".")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}
	cmd = exec.CommandContext(ctx.Context, "git", "commit", "-m", "feat: initial commit")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("❌ Failed to start devnet: %w", err)
	}

	if verbose {
		log.InfoWithActor("User", "Git repository initialized successfully.")
		if len(output) > 0 {
			log.InfoWithActor("User", "Git init output: \"%s\"", strings.Trim(string(output), "\n"))
		}
	}
	return nil
}

/*
func collectSubmoduleInfo(ctx *cli.Context, git template.GitClient, targetDir, pathPrefix string) ([]template.Submodule, error) {
	// collect all submodules info
	var submoduleInfos []template.Submodule
	submodules, err := git.SubmoduleList(ctx.Context, targetDir)
	// get commit for the submodule
	if err != nil {
		return nil, fmt.Errorf("failed list submodules: %w", err)
	}
	// collect the referenced commit in the submodule list
	for _, m := range submodules {
		submoduleInfos = append(submoduleInfos, template.Submodule{
			Name: m.Name,
			Path: fmt.Sprintf("%s/%s", pathPrefix, m.Path),
			URL:  m.URL,
		})
	}
	return submoduleInfos, nil
}

func registerSubmodules(ctx *cli.Context, git template.GitClient, targetDir string, submoduleInfos []template.Submodule) error {
	// reinstate gitmodules
	for _, mod := range submoduleInfos {
		// init the submodule at path in parent
		if err := git.AddSubmodule(ctx.Context, targetDir, mod.URL, mod.Path); err != nil {
			return fmt.Errorf("failed to init submodule: %w", err)
		}
	}

	return nil
}

// replaceGitmodules replaces root .gitmodules with the one under ./contracts
func replaceGitmodules(targetDir string, verbose bool) error {
	log, _ := common.GetLogger()

	// Remove old root file
	if verbose {
		log.InfoWithActor("User", "rm %s/.gitmodules", targetDir)
	}
	if err := os.Remove(filepath.Join(targetDir, ".gitmodules")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rm old .gitmodules: %w", err)
	}

	// Load contracts/.gitmodules
	src := filepath.Join(targetDir, contractsBasePath, ".gitmodules")
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}

	// Prefix section names: [submodule "X"] → [submodule "contracts/X"]
	reSection := regexp.MustCompile(`(?m)^\[submodule\s+"([^"]+)"\]`)
	out := reSection.ReplaceAll(raw, []byte(`[submodule "contracts/$1"]`))

	// Prefix path = X → path = contracts/X
	rePath := regexp.MustCompile(`(?m)^(\s*path\s*=\s*)(.+)$`)
	out = rePath.ReplaceAll(out, []byte(`${1}contracts/${2}`))

	// Write to root
	dest := filepath.Join(targetDir, ".gitmodules")
	if verbose {
		log.InfoWithActor("User", "write %s", dest)
	}
	if err := os.WriteFile(dest, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}

	// Remove from contracts
	if err := os.Remove(filepath.Join(targetDir, contractsBasePath, ".gitmodules")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("rm old .gitmodules: %w", err)
	}

	return nil
}*/
