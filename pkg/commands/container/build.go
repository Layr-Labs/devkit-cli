package container

import (
	"fmt"
	"os"
	"os/exec"

	templcmd "github.com/Layr-Labs/devkit-cli/pkg/commands/template"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// BuildCommand: devkit container build [app]
var BuildCommand = &cli.Command{
    Name:  "build",
    Usage: "Build container image locally (app optional in project dir)",
    Flags: append([]cli.Flag{
        &cli.StringFlag{Name: "tag", Usage: "Image tag", Value: "latest"},
        &cli.StringFlag{Name: "registry", Usage: "Registry host (optional)"},
        &cli.StringFlag{Name: "owner", Usage: "Registry owner/org (optional)"},
        &cli.StringFlag{Name: "repo", Usage: "Full repo override (e.g., ghcr.io/org/app)"},
    }, common.GlobalFlags...),
    Action: func(cCtx *cli.Context) error {
        if err := common.EnsureDockerIsRunning(cCtx); err != nil {
            return err
        }

        logger := common.LoggerFromContext(cCtx)

        app := cCtx.Args().Get(0)
        if app == "" {
            n, _, _, _, _ := templcmd.GetTemplateInfo()
            app = n
        }
        if app == "" {
            return fmt.Errorf("unable to determine app name; pass [app] or run in a project directory")
        }

        imageRef := cCtx.String("repo")
        if imageRef == "" {
            if cCtx.String("registry") != "" && cCtx.String("owner") != "" {
                imageRef = fmt.Sprintf("%s/%s/%s", cCtx.String("registry"), cCtx.String("owner"), app)
            } else {
                imageRef = app
            }
        }

        cmd := exec.CommandContext(cCtx.Context, "docker", "build", "-t", fmt.Sprintf("%s:%s", imageRef, cCtx.String("tag")), ".")
        cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("docker build failed: %w", err)
        }

        logger.Info("Built image %s:%s", imageRef, cCtx.String("tag"))
        return nil
    },
}


