package container

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

// InitCommand: devkit container init [name] [dockerfile]
var InitCommand = &cli.Command{
    Name:      "init",
    Usage:     "Initialize existing project with Dockerfile",
    ArgsUsage: "<name> <dockerfile>",
    Flags:     append([]cli.Flag{}, common.GlobalFlags...),
    Action: func(cCtx *cli.Context) error {
        logger := common.LoggerFromContext(cCtx)

        name := cCtx.Args().Get(0)
        df := cCtx.Args().Get(1)

        if name == "" {
            var err error
            name, err = output.InputString(
                "Project name:",
                "Name of the existing project",
                "",
                func(s string) error {
                    if s == "" {
                        return fmt.Errorf("name cannot be empty")
                    }
                    return nil
                },
            )
            if err != nil {
                return fmt.Errorf("prompt failed: %w", err)
            }
        }

        if df == "" {
            if _, err := os.Stat("Dockerfile"); err == nil {
                df = "Dockerfile"
            } else {
                var err error
                df, err = output.InputString(
                    "Path to Dockerfile:",
                    "Provide a Dockerfile path",
                    "",
                    func(s string) error {
                        if s == "" {
                            return fmt.Errorf("dockerfile path required")
                        }
                        if _, e := os.Stat(s); e != nil {
                            return fmt.Errorf("dockerfile not accessible: %v", e)
                        }
                        return nil
                    },
                )
                if err != nil {
                    return fmt.Errorf("prompt failed: %w", err)
                }
            }
        }

        if _, err := os.Stat(df); err != nil {
            return fmt.Errorf("dockerfile not accessible: %w", err)
        }

        dest := filepath.Join(".", "Dockerfile")
        if _, err := os.Stat(dest); err == nil && df != dest {
            ok, err := output.Confirm("Dockerfile already exists. Overwrite?")
            if err != nil {
                return fmt.Errorf("confirmation failed: %w", err)
            }
            if !ok {
                logger.Info("Skipped Dockerfile overwrite")
                return nil
            }
        }

        if df != dest {
            if err := copyFile(df, dest); err != nil {
                return fmt.Errorf("failed to place Dockerfile: %w", err)
            }
        }

        logger.Info("Initialized Dockerfile for %s", name)
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


