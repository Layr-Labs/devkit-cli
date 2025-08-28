package container

import (
	"fmt"
	"os"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

// InitCommand: devkit container init [name] [dockerfile]
var InitCommand = &cli.Command{
	Name:      "init",
	Usage:     "Initialize existing project with compute tee project",
	ArgsUsage: "",
	Flags:     append([]cli.Flag{}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		ok, err := output.Confirm("Are you sure you want to initialize a project with compute tee project?")
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !ok {
			return fmt.Errorf("user cancelled")
		}

		// check if Dockerfile exists
		if _, err := os.Stat("Dockerfile"); err == nil {
			logger.Info("Dockerfile already exists. Continuing...")
		} else {
			logger.Info("Dockerfile does not exist. Creating...")
			// create Dockerfile
			err := os.WriteFile("Dockerfile", []byte("FROM ubuntu:20.04\nRUN apt-get update && apt-get install -y curl\nRUN curl -sL https://deb.nodesource.com/setup_14.x | bash - && apt-get install -y nodejs\nRUN npm install -g yarn\nRUN yarn install\nRUN yarn build\nRUN yarn start"), 0o644)
			if err != nil {
				return fmt.Errorf("failed to create Dockerfile: %w", err)
			}
			logger.Info("Dockerfile created successfully")
		}

		// check if .compute-tee folder exists
		if _, err := os.Stat(".compute-tee"); err == nil {
			logger.Info(".compute-tee folder already exists. Continuing...")
		} else {
			logger.Info(".compute-tee folder does not exist. Creating...")
			// create .compute-tee/context folder
			err := os.MkdirAll(".compute-tee/context", 0o755)
			if err != nil {
				return fmt.Errorf("failed to create .compute-tee folder: %w", err)
			}
			logger.Info(".compute-tee/context folder created successfully")
		}

		logger.Info("Initialized Dockerfile and .compute-tee/context folder")
		return nil
	},
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o644)
}
