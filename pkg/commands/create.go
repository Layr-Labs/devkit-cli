package commands

import (
	"fmt"
	"io"
	"log"
	"os"
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

		if err := createProjectDir(targetDir, cCtx.Bool("overwrite"), cCtx.Bool("verbose")); err != nil {
			return err
		}

		templateURL, err := getTemplateURL(cCtx)
		if err != nil {
			return err
		}

		if cCtx.Bool("verbose") {
			log.Printf("Using template: %s", templateURL)
		}

		// Fetch template
		fetcher := &template.GitFetcher{}
		if err := fetcher.Fetch(templateURL, targetDir); err != nil {
			return fmt.Errorf("failed to fetch template from %s: %w", templateURL, err)
		}

		// Copy default.eigen.toml to the project directory
		if err := copyDefaultTomlToProject(targetDir, projectName, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initialize eigen.toml: %w", err)
		}

		// Copies the default keystore json files in the keystores/ directory
		if err := copyDefaultKeystoresToProject(targetDir, cCtx.Bool("verbose")); err != nil {
			return fmt.Errorf("failed to initilize keystores: %w", err)
		}

		// Save project settings with telemetry preference
		telemetryEnabled := !cCtx.Bool("no-telemetry")
		if err := common.SaveProjectSettings(targetDir, telemetryEnabled); err != nil {
			return fmt.Errorf("failed to save project settings: %w", err)
		}

		log.Printf("Project %s created successfully in %s. Run 'cd %s' to get started.", projectName, targetDir, targetDir)
		return nil
	},
}

func getTemplateURL(cCtx *cli.Context) (string, error) {
	if templatePath := cCtx.String("template-path"); templatePath != "" {
		return templatePath, nil
	}

	arch := cCtx.String("arch")

	config, err := template.LoadConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load templates config: %w", err)
	}

	url, err := template.GetTemplateURL(config, arch, cCtx.String("lang"))
	if err != nil {
		return "", fmt.Errorf("failed to get template URL: %w", err)
	}

	if url == "" {
		return "", fmt.Errorf("no template found for architecture %s and language %s", arch, cCtx.String("lang"))
	}

	return url, nil
}

func createProjectDir(targetDir string, overwrite, verbose bool) error {
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
	return os.MkdirAll(targetDir, 0755)
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
