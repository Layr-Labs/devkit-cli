package commands

import (
	"bytes"
	"fmt"
	"io/fs"
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

		// get logger
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
		logger.Debug("Creating new AVS project: %s", projectName)
		logger.Debug("Directory: %s", cCtx.String("dir"))
		logger.Debug("Language: %s", cCtx.String("lang"))
		logger.Debug("Architecture: %s", cCtx.String("arch"))
		logger.Debug("Environment: %s", cCtx.String("env"))
		if cCtx.String("template-path") != "" {
			logger.Debug("Template Path: %s", cCtx.String("template-path"))
		}

		// Get template URLs
		mainBaseURL, mainVersion, err := getTemplateURLs(cCtx)
		if err != nil {
			return err
		}

		// Create project directories
		if err := createProjectDir(logger, targetDir, cCtx.Bool("overwrite")); err != nil {
			return err
		}

		logger.Debug("Using template: %s", mainBaseURL)
		if mainVersion != "" {
			logger.Info("Template version: %s", mainVersion)
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
			logger.Warn("Project README.md is missing: %w", err)
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
		logger.Info("Installing template dependencies\n\n")

		// Run init on the template init script
		if _, err = common.CallTemplateScript(cCtx.Context, logger, targetDir, scriptPath, common.ExpectNonJSONResponse, nil); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", scriptPath, err)
		}

		// Tidy the logs
		logger.Debug("\nFinalising new project\n\n")

		// Write the example .env file
		err = os.WriteFile(filepath.Join(targetDir, ".env.example"), []byte(config.EnvExample), 0644)
		if err != nil {
			return fmt.Errorf("failed to write .env.example: %w", err)
		}

		// Get app environment for UUID
		appEnv, ok := common.AppEnvironmentFromContext(cCtx.Context)
		if !ok {
			return fmt.Errorf("could not determine application environment")
		}

		// Save the users user_uuid to global config
		if err := common.SaveUserId(appEnv.UserUUID); err != nil {
			return fmt.Errorf("failed to save global settings: %w", err)
		}

		// Get global telemetry preference
		globalTelemetryEnabled, err := common.GetGlobalTelemetryPreference()
		if err != nil {
			// If we can't get global preference, default to false for safety
			logger.Debug("Unable to get global telemetry preference, defaulting to false: %v", err)
		}

		// Use global preference if set, otherwise default to false
		telemetryEnabled := false
		if globalTelemetryEnabled != nil {
			telemetryEnabled = *globalTelemetryEnabled
		}

		// Copy config.yaml to the project directory with UUID and telemetry settings
		if err := copyDefaultConfigToProject(logger, targetDir, projectName, appEnv.ProjectUUID, mainBaseURL, mainVersion, telemetryEnabled); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", common.BaseConfig, err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(logger, targetDir); err != nil {
			return fmt.Errorf("failed to initialize keystores: %w", err)
		}

		// Copies the default .zeus file in the .zeus/ directory
		if err := copyZeusFileToProject(logger, targetDir); err != nil {
			return fmt.Errorf("failed to initialize .zeus: %w", err)
		}

		// Initialize git repository in the project directory
		if err := initGitRepo(cCtx, targetDir, logger); err != nil {
			logger.Warn("Failed to initialize Git repository in %s: %v", targetDir, err)
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

func createProjectDir(logger iface.Logger, targetDir string, overwrite bool) error {
	// Check if directory exists and handle overwrite
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {

		if !overwrite {
			return fmt.Errorf("directory %s already exists. Use --overwrite flag to force overwrite", targetDir)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}

		logger.Debug("Removed existing directory: %s", targetDir)
	}

	// Create main project directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	return nil
}

// copyDefaultConfigToProject copies config to the project directory with updated project name, UUID, and telemetry settings
func copyDefaultConfigToProject(logger iface.Logger, targetDir, projectName, projectUUID string, templateBaseURL, templateVersion string, telemetryEnabled bool) error {
	// Create and ensure target config directory exists
	destConfigDir := filepath.Join(targetDir, "config")
	if err := os.MkdirAll(destConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create target config directory: %w", err)
	}

	// Read config.yaml from config embed
	configContent := configs.ConfigYamls[configs.LatestVersion]

	// Unmarshal the YAML content into a map
	var cfg common.Config
	if err := yaml.Unmarshal([]byte(configContent), &cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config YAML: %w", err)
	}
	cfg.Config.Project.Name = projectName
	cfg.Config.Project.ProjectUUID = projectUUID
	cfg.Config.Project.TelemetryEnabled = telemetryEnabled
	cfg.Config.Project.TemplateBaseURL = templateBaseURL
	cfg.Config.Project.TemplateVersion = templateVersion

	// Marshal the modified configuration back to YAML
	newContentBytes, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal modified config: %w", err)
	}

	// Write the updated config
	err = os.WriteFile(filepath.Join(destConfigDir, common.BaseConfig), newContentBytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", common.BaseConfig, err)
	}

	logger.Debug("Created config/%s in project directory", common.BaseConfig)

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

		logger.Debug("Copied context file: %s", entryName)
	}

	return nil
}

// Creates a keystores directory with default keystore json files
func copyDefaultKeystoresToProject(logger iface.Logger, targetDir string) error {
	// Construct keystore dest
	destKeystoreDir := filepath.Join(targetDir, "keystores")

	// Create the destination keystore directory
	if err := os.MkdirAll(destKeystoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create keystores directory: %w", err)
	}

	logger.Debug("Created directory: %s", destKeystoreDir)

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

		logger.Debug("Copied keystore: %s", fileName)
	}

	return nil
}

// Copies the .zeus file to the project directory
func copyZeusFileToProject(logger iface.Logger, targetDir string) error {
	// Destination .zeus file path
	destZeusPath := filepath.Join(targetDir, common.ZeusConfig)

	// Read the embedded zeus config
	zeusConfigContent := config.ZeusConfig

	if err := os.WriteFile(destZeusPath, []byte(zeusConfigContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", common.ZeusConfig, err)
	}

	logger.Debug("Copied zeus config: %s", common.ZeusConfig)

	return nil
}

// initGitRepo initializes a new Git repository in the target directory.
func initGitRepo(ctx *cli.Context, targetDir string, logger iface.Logger) error {
	logger.Debug("Removing existing .git directory in %s (if any)...", targetDir)

	// backup gitmodules before deleting .git
	err := backupSubmodules(targetDir)
	if err != nil {
		return fmt.Errorf("git submodule backup failed: %w", err)
	}

	// remove the old .git dir
	gitDir := filepath.Join(targetDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove existing .git directory: %w", err)
	}

	logger.Debug("Initializing Git repository in %s...", targetDir)

	cmd := exec.CommandContext(ctx.Context, "git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}

	// reinstate gitmodules
	err = registerSubmodules(targetDir)
	if err != nil {
		return fmt.Errorf("git submodule registration failed: %w", err)
	}

	// remove remote origin from config
	cmd = exec.CommandContext(ctx.Context, "git", "remote", "remove", "origin")
	cmd.Dir = targetDir
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove remote origin: %w\nOutput: %s", err, string(output))
	}

	// cleanup submodule backups
	err = deleteBackup(targetDir)
	if err != nil {
		return fmt.Errorf("git submodule cleanup failed: %w", err)
	}

	// write a .gitignore into the new dir
	err = os.WriteFile(filepath.Join(targetDir, ".gitignore"), []byte(config.GitIgnore), 0644)
	if err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	// add all changes and commit if identity is set
	cmd = exec.CommandContext(ctx.Context, "git", "add", ".")
	cmd.Dir = targetDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// check for user.name and user.email
	hasIdentity := func(key string) bool {
		out, _ := exec.CommandContext(ctx.Context, "git", "config", "--get", key).Output()
		return len(bytes.TrimSpace(out)) > 0
	}

	if hasIdentity("user.name") && hasIdentity("user.email") {
		cmd = exec.CommandContext(ctx.Context, "git", "commit", "-m", "feat: initial commit")
		cmd.Dir = targetDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create initial commit: %w", err)
		}
	} else {
		// skip commit if no identity
		logger.Warn("Git identity not set - skipping initial commit")
	}

	logger.Debug("Git repository initialized successfully.")
	if len(output) > 0 {
		logger.Debug("Git init output: \"%s\"", strings.Trim(string(output), "\n"))
	}

	return nil
}

// backupSubmodules copies .git/modules and .git/config for later restoration
func backupSubmodules(targetDir string) error {
	gitDir := filepath.Join(targetDir, ".git")
	modulesDir := filepath.Join(gitDir, "modules")
	configPath := filepath.Join(gitDir, "config")

	// backup .git/config
	configBackup := filepath.Join(targetDir, ".git_config_backup")
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read .git/config: %w", err)
		}
		if err := os.WriteFile(configBackup, data, 0644); err != nil {
			return fmt.Errorf("failed to write .git_config_backup: %w", err)
		}
	}

	// backup .git/modules directory
	modulesBackup := filepath.Join(targetDir, ".git_modules_backup")
	if _, err := os.Stat(modulesDir); err == nil {
		err := copyDir(modulesDir, modulesBackup)
		if err != nil {
			return fmt.Errorf("failed to backup .git/modules: %w", err)
		}
	}

	return nil
}

// registerSubmodules restores .git/config and .git/modules for submodule recognition
func registerSubmodules(targetDir string) error {
	// restore .git/config
	configBackup := filepath.Join(targetDir, ".git_config_backup")
	configTarget := filepath.Join(targetDir, ".git", "config")
	if _, err := os.Stat(configBackup); err == nil {
		data, err := os.ReadFile(configBackup)
		if err != nil {
			return fmt.Errorf("failed to read config backup: %w", err)
		}
		if err := os.WriteFile(configTarget, data, 0644); err != nil {
			return fmt.Errorf("failed to restore .git/config: %w", err)
		}
	}

	// erstore .git/modules directory
	modulesBackup := filepath.Join(targetDir, ".git_modules_backup")
	modulesTarget := filepath.Join(targetDir, ".git", "modules")
	if _, err := os.Stat(modulesBackup); err == nil {
		err := copyDir(modulesBackup, modulesTarget)
		if err != nil {
			return fmt.Errorf("failed to restore .git/modules: %w", err)
		}
	}

	return nil
}

// deleteBackup will delete the backup of .git/modules & .git/config
func deleteBackup(targetDir string) error {
	// remove .git_modules_backup dir
	gitModulesBackupDir := filepath.Join(targetDir, ".git_modules_backup")
	if err := os.RemoveAll(gitModulesBackupDir); err != nil {
		return fmt.Errorf("failed to remove .git_modules_backup directory: %w", err)
	}

	// remove .git_config_backup file
	gitConfigBackupFile := filepath.Join(targetDir, ".git_config_backup")
	if err := os.Remove(gitConfigBackupFile); err != nil {
		return fmt.Errorf("failed to remove .git_config_backup file: %w", err)
	}

	return nil
}

// copyDir is a helper to copy src to dest
func copyDir(src string, dest string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// copy file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0644)
	})
}
