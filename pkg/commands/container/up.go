package container

import (
	"fmt"
	"os"
	"os/exec"

	templcmd "github.com/Layr-Labs/devkit-cli/pkg/commands/template"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// UpCommand: devkit container up [app]
var UpCommand = &cli.Command{
    Name:  "up",
    Usage: "Run container locally for testing (app optional in project dir)",
    Flags: append([]cli.Flag{
        &cli.StringFlag{Name: "tag", Usage: "Image tag", Value: "latest"},
        &cli.StringSliceFlag{Name: "port", Usage: "Port mappings (e.g. 8080:8080)", Value: cli.NewStringSlice()},
        &cli.StringSliceFlag{Name: "env", Usage: "Env vars (KEY=VALUE)", Value: cli.NewStringSlice()},
        &cli.StringFlag{Name: "registry", Usage: "Registry host (optional)"},
        &cli.StringFlag{Name: "owner", Usage: "Registry owner/org (optional)"},
        &cli.StringFlag{Name: "repo", Usage: "Full repo override (e.g., ghcr.io/org/app)"},
        &cli.BoolFlag{Name: "detach", Aliases: []string{"d"}, Usage: "Run detached"},
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

        args := []string{"run", "--rm", "--name", app}
        for _, p := range cCtx.StringSlice("port") {
            args = append(args, "-p", p)
        }
        for _, e := range cCtx.StringSlice("env") {
            args = append(args, "-e", e)
        }
        if cCtx.Bool("detach") {
            args = append(args, "-d")
        }
        args = append(args, fmt.Sprintf("%s:%s", imageRef, cCtx.String("tag")))

        cmd := exec.CommandContext(cCtx.Context, "docker", args...)
        cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
        if err := cmd.Run(); err != nil {
            return fmt.Errorf("docker run failed: %w", err)
        }

        if cCtx.Bool("detach") {
            logger.Info("Container %s started", app)
        }
        return nil
    },
}


