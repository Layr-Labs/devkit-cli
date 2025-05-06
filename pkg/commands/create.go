package commands

import (
	"fmt"
	"io"
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
		fetcher := &template.GitFetcher{}
		if err := fetcher.Fetch(mainURL, targetDir); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", mainURL, err)
		}

		// Check for contracts template and fetch if available
		if contractsURL != "" {
			contractsDir := filepath.Join(targetDir, common.ContractsDir)

			// Remove the contracts directory if it exists
			if _, err := os.Stat(contractsDir); !os.IsNotExist(err) {
				if err := os.RemoveAll(contractsDir); err != nil {
					log.Printf("Warning: Failed to remove existing contracts directory: %v", err)
				}
			}

			if err := fetcher.Fetch(contractsURL, contractsDir); err != nil {
				log.Printf("Warning: Failed to fetch contracts template: %v", err)
			}
		}

		// Copy default.eigen.toml to the project directory
		if err := copyDefaultTomlToProject(targetDir, projectName, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize eigen.toml: %w", err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(targetDir, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize keystores: %w", err)
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

		// Install Forge dependencies if contracts directory exists
		contractsDir := filepath.Join(targetDir, common.ContractsDir)
		if _, err := os.Stat(contractsDir); !os.IsNotExist(err) {
			if err := installForgeDependencies(contractsDir, cCtx.Bool("verbose")); err != nil {
				log.Printf("Warning: Failed to install Forge dependencies in %s: %v", contractsDir, err)
			}
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

// copyDefaultTomlToProject copies default.eigen.toml to the project directory with updated project name
func copyDefaultTomlToProject(targetDir, projectName string, verbose bool) error {
	// Read default.eigen.toml from current directory
	content, err := os.ReadFile("default.eigen.toml")
	if err != nil {
		return fmt.Errorf("default.eigen.toml not found: %w", err)
	}

	// Replace project name and write to target
	newContent := strings.Replace(string(content), `name = "my-avs"`, fmt.Sprintf(`name = "%s"`, projectName), 1)
	err = os.WriteFile(filepath.Join(targetDir, "eigen.toml"), []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write eigen.toml: %w", err)
	}

	if verbose {
		log.Printf("Created eigen.toml in project directory")
	}
	return nil
}

// / Creates a keystores directory with default keystore json files
func copyDefaultKeystoresToProject(targetDir string, verbose bool) error {
	srcKeystoreDir := "keystores"
	destKeystoreDir := filepath.Join(targetDir, "keystores")

	// Create the destination keystore directory
	if err := os.MkdirAll(destKeystoreDir, 0755); err != nil {
		return fmt.Errorf("failed to create keystores directory: %w", err)
	}
	if verbose {
		log.Printf("Created directory: %s", destKeystoreDir)
	}

	// Read files from the source keystores directory
	files, err := os.ReadDir(srcKeystoreDir)
	if err != nil {
		return fmt.Errorf("failed to read keystores directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue // skip subdirectories
		}

		srcPath := filepath.Join(srcKeystoreDir, file.Name())
		destPath := filepath.Join(destKeystoreDir, file.Name())

		srcFile, err := os.Open(srcPath)
		if err != nil {
			return fmt.Errorf("failed to open source keystore file %s: %w", srcPath, err)
		}
		defer srcFile.Close()

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination keystore file %s: %w", destPath, err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", file.Name(), err)
		}

		if verbose {
			log.Printf("Copied keystore: %s", file.Name())
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

// installForgeDependencies runs 'forge install' in the specified contracts directory.
func installForgeDependencies(contractsDir string, verbose bool) error {
	if verbose {
		log.Printf("Installing Forge dependencies in %s...", contractsDir)
	}
	cmd := exec.Command("forge", "install")
	cmd.Dir = contractsDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("forge install failed: %w\nOutput: %s", err, string(output))
	}
	if verbose {
		log.Printf("Forge dependencies installed successfully.")
		if len(output) > 0 {
			log.Printf("Forge install output:\n%s", string(output))
		}
	}
	return nil
}
