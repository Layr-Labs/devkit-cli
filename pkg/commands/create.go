package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devkit-cli/config"
	"devkit-cli/pkg/commands/keystore"
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/logger"
	"devkit-cli/pkg/telemetry"
	"devkit-cli/pkg/template"

	"github.com/urfave/cli/v2"
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
			Name:  "template-path",
			Usage: "Direct GitHub URL to use as template (overrides templates.yml)",
		},
		&cli.BoolFlag{
			Name:  "disable-telemetry",
			Usage: "Opt out of anonymous telemetry collection",
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
			if cCtx.String("template-path") != "" {
				log.Info("Template Path: %s", cCtx.String("template-path"))
			}

			// Log telemetry status (accounting for client type)
			if cCtx.Bool("disable-telemetry") {
				log.Info("Telemetry: disabled (via flag)")
			} else {
				client, ok := telemetry.ClientFromContext(cCtx.Context)
				if !ok || telemetry.IsNoopClient(client) {
					log.Info("Telemetry: disabled")
				} else {
					log.Info("Telemetry: enabled")
				}
			}
		}

		// Get template URLs
		mainURL, contractsURL, err := getTemplateURLs(cCtx)
		if err != nil {
			return err
		}

		// Create project directories
		if err := createProjectDir(targetDir, cCtx.Bool("overwrite"), cCtx.Bool("verbose")); err != nil {
			return err
		}

		if cCtx.Bool("verbose") {
			log.Info("Using template: %s", mainURL)
			if contractsURL != "" {
				log.Info("Using contracts template: %s", contractsURL)
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
		if err := fetcher.Fetch(cCtx.Context, mainURL, targetDir); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", mainURL, err)
		}

		// Check for contracts template and fetch if missing
		if contractsURL != "" {
			contractsDir := filepath.Join(targetDir, common.ContractsDir)
			contractsDirReadme := filepath.Join(contractsDir, "README.md")

			// Fetch the contracts directory if it does not exist in the template
			if _, err := os.Stat(contractsDirReadme); os.IsNotExist(err) {
				if err := fetcher.Fetch(cCtx.Context, contractsURL, contractsDir); err != nil {
					log.Warn("Failed to fetch contracts template: %v", err)
				}
			}
		}

		// Copy config.yaml to the project directory
		if err := copyDefaultConfigToProject(targetDir, projectName, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", common.BaseConfig, err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(targetDir, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize keystores: %w", err)
		}

		// Save project settings with telemetry preference
		telemetryEnabled := !cCtx.Bool("disable-telemetry")
		if err := common.SaveProjectSettings(targetDir, telemetryEnabled); err != nil {
			return fmt.Errorf("failed to save project settings: %w", err)
		}

		// Initialize git repository in the project directory
		if err := initGitRepo(cCtx, targetDir, cCtx.Bool("verbose")); err != nil {
			log.Warn("Failed to initialize Git repository in %s: %v", targetDir, err)
		}

		log.Info("Project %s created successfully in %s. Run 'cd %s' to get started.", projectName, targetDir, targetDir)
		return nil
	},
}

func getTemplateURLs(cCtx *cli.Context) (string, string, error) {
	if templatePath := cCtx.String("template-path"); templatePath != "" {
		return templatePath, "", nil
	}

	config, err := template.LoadConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to load templates config: %w", err)
	}

	arch := cCtx.String("arch")
	lang := cCtx.String("lang")

	mainURL, contractsURL, err := template.GetTemplateURLs(config, arch, lang)
	if err != nil {
		return "", "", fmt.Errorf("failed to get template URLs: %w", err)
	}

	if mainURL == "" {
		return "", "", fmt.Errorf("no template found for architecture %s and language %s", arch, lang)
	}

	return mainURL, contractsURL, nil
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
func copyDefaultConfigToProject(targetDir, projectName string, verbose bool) error {
	// get logger
	log, _ := common.GetLogger()

	// Create and ensure target config directory exists
	destConfigDir := filepath.Join(targetDir, "config")
	if err := os.MkdirAll(destConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create target config directory: %w", err)
	}

	// Read config.yaml from config embed and write to target
	newContent := strings.Replace(config.DefaultConfigYaml, `name = "my-avs"`, fmt.Sprintf(`name = "%s"`, projectName), 1)
	err := os.WriteFile(filepath.Join(destConfigDir, common.BaseConfig), []byte(newContent), 0644)
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
	for name, yaml := range config.ContextYamls {
		content := yaml
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
	files := keystore.KeystoreEmbeds

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
		log.Info("Initializing Git repository in %s...", targetDir)
	}
	cmd := exec.CommandContext(ctx.Context, "git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}
	if verbose {
		log.Info("Git repository initialized successfully.")
		if len(output) > 0 {
			log.Info("Git init output: \"%s\"", strings.Trim(string(output), "\n"))
		}
	}
	return nil
}
