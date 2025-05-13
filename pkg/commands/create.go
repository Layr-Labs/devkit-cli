package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"devkit-cli/pkg/common"
	"devkit-cli/pkg/telemetry"
	"devkit-cli/pkg/template"

	"github.com/urfave/cli/v2"
)

// CreateCommand defines the "create" command
var CreateCommand = &cli.Command{
	Name:      "create",
	Usage:     "Initializes a new AVS project scaffold (Hourglass model)",
	ArgsUsage: "<project-name>",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "dir",
			Usage: "Set output directory for the new project",
			Value: filepath.Join(os.Getenv("HOME"), "avs"),
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
			Name:  "no-telemetry",
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
		if cCtx.NArg() == 0 {
			return fmt.Errorf("project name is required\nUsage: avs create <project-name> [flags]")
		}
		projectName := cCtx.Args().First()
		targetDir := filepath.Join(cCtx.String("dir"), projectName)

		if cCtx.Bool("verbose") {
			log.Printf("Creating new AVS project: %s", projectName)
			log.Printf("Directory: %s", cCtx.String("dir"))
			log.Printf("Language: %s", cCtx.String("lang"))
			log.Printf("Architecture: %s", cCtx.String("arch"))
			log.Printf("Environment: %s", cCtx.String("env"))
			if cCtx.String("template-path") != "" {
				log.Printf("Template Path: %s", cCtx.String("template-path"))
			}

			// Log telemetry status (accounting for client type)
			if cCtx.Bool("no-telemetry") {
				log.Printf("Telemetry: disabled (via flag)")
			} else {
				client, ok := telemetry.FromContext(cCtx.Context)
				if !ok || telemetry.IsNoopClient(client) {
					log.Printf("Telemetry: disabled")
				} else {
					log.Printf("Telemetry: enabled")
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
			log.Printf("Using template: %s", mainURL)
			if contractsURL != "" {
				log.Printf("Using contracts template: %s", contractsURL)
			}
		}

		// Fetch main template
		fetcher := &template.GitFetcher{
			MaxDepth:       cCtx.Int("depth"),
			MaxRetries:     cCtx.Int("retries"),
			MaxConcurrency: cCtx.Int("concurrency"),
		}
		if err := fetcher.Fetch(mainURL, targetDir, cCtx.Bool("verbose"), cCtx.Bool("no-cache")); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", mainURL, err)
		}

		// Check for contracts template and fetch if missing
		if contractsURL != "" {
			contractsDir := filepath.Join(targetDir, common.ContractsDir)

			// Fetch the contracts directory if it does not exist in the template
			if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
				if err := fetcher.Fetch(contractsURL, contractsDir, cCtx.Bool("verbose"), cCtx.Bool("no-cache")); err != nil {
					log.Printf("Warning: Failed to fetch contracts template: %v", err)
				}
			}
		}

		// Copy default.eigen.toml to the project directory
		if err := copyDefaultConfigToProject(targetDir, projectName, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize eigen.toml: %w", err)
		}

		// Save project settings with telemetry preference
		telemetryEnabled := !cCtx.Bool("no-telemetry")
		if err := common.SaveProjectSettings(targetDir, telemetryEnabled); err != nil {
			return fmt.Errorf("failed to save project settings: %w", err)
		}

		// Initialize git repository in the project directory
		if err := initGitRepo(targetDir, cCtx.Bool("verbose")); err != nil {
			log.Printf("Warning: Failed to initialize Git repository in %s: %v", targetDir, err)
		}

		log.Printf("Project %s created successfully in %s. Run 'cd %s' to get started.", projectName, targetDir, targetDir)
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
	// Check if directory exists and handle overwrite
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		if !overwrite {
			return fmt.Errorf("directory %s already exists. Use --overwrite flag to force overwrite", targetDir)
		}
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("failed to remove existing directory: %w", err)
		}
		if verbose {
			log.Printf("Removed existing directory: %s", targetDir)
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
	// Step 1: Copy and modify config.yaml
	configDir := filepath.Join("config")
	contextsDir := filepath.Join(configDir, "contexts")

	content, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
	if err != nil {
		return fmt.Errorf("config/config.yaml not found: %w", err)
	}

	// Replace project name
	newContent := strings.Replace(string(content), `name: "my-avs"`, fmt.Sprintf(`name: "%s"`, projectName), 1)

	// Ensure target config directory exists
	destConfigDir := filepath.Join(targetDir, "config")
	if err := os.MkdirAll(destConfigDir, 0755); err != nil {
		return fmt.Errorf("failed to create target config directory: %w", err)
	}

	// Write modified config.yaml
	destConfigPath := filepath.Join(destConfigDir, "config.yaml")
	if err := os.WriteFile(destConfigPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write config/config.yaml: %w", err)
	}

	if verbose {
		log.Printf("Created config/config.yaml in project directory")
	}

	// Step 2: Copy all context files
	destContextsDir := filepath.Join(destConfigDir, "contexts")
	if err := os.MkdirAll(destContextsDir, 0755); err != nil {
		return fmt.Errorf("failed to create target contexts directory: %w", err)
	}

	entries, err := os.ReadDir(contextsDir)
	if err != nil {
		return fmt.Errorf("failed to read contexts directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // skip subdirectories
		}
		srcPath := filepath.Join(contextsDir, entry.Name())
		destPath := filepath.Join(destContextsDir, entry.Name())

		if err := common.CopyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", entry.Name(), err)
		}

		if verbose {
			log.Printf("Copied context file: %s", entry.Name())
		}
	}

	return nil
}

// initGitRepo initializes a new Git repository in the target directory.
func initGitRepo(targetDir string, verbose bool) error {
	if verbose {
		log.Printf("Initializing Git repository in %s...", targetDir)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git init failed: %w\nOutput: %s", err, string(output))
	}
	if verbose {
		log.Printf("Git repository initialized successfully.")
		if len(output) > 0 {
			log.Printf("Git init output:\n%s", string(output))
		}
	}
	return nil
}
