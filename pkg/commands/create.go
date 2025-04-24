package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"devkit-cli/pkg/common"
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
			if cCtx.Bool("no-telemetry") {
				log.Printf("Telemetry: disabled")
			} else {
				log.Printf("Telemetry: enabled")
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
