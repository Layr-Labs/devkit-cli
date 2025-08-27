package container

import (
	"fmt"
	"path/filepath"

	templcmd "github.com/Layr-Labs/devkit-cli/pkg/commands/template"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/output"
	"github.com/urfave/cli/v2"
)

// CreateCommand: devkit container create [name] [language]
var CreateCommand = &cli.Command{
    Name:      "create",
    Usage:     "Create new container project from template",
    ArgsUsage: "<name> [language]",
    Flags: append([]cli.Flag{
        &cli.StringFlag{Name: "dir", Usage: "Output directory", Value: "."},
        &cli.StringFlag{Name: "template-url", Usage: "Direct template repo URL"},
        &cli.StringFlag{Name: "template-version", Usage: "Git ref for the template"},
    }, common.GlobalFlags...),
    Action: func(cCtx *cli.Context) error {
        logger := common.LoggerFromContext(cCtx)

        name := cCtx.Args().Get(0)
        lang := cCtx.Args().Get(1)

        if name == "" {
            var err error
            name, err = output.InputString("Project name:", "Name for the new container project", "", func(s string) error {
                if s == "" {
                    return fmt.Errorf("name cannot be empty")
                }
                return nil
            })
            if err != nil {
                return fmt.Errorf("prompt failed: %w", err)
            }
        }

        if lang == "" {
            _, _, _, defLang, _ := templcmd.GetTemplateInfoDefault()
            if defLang == "" {
                defLang = "go"
            }
            lang, _ = output.InputString("Language:", "Programming language (e.g., go, rust, ts, python)", defLang, nil)
            if lang == "" {
                lang = defLang
            }
        }

        outDir := cCtx.String("dir")

        dir := ""
        script := filepath.Join(".devkit", "scripts", "container", "create")
        _, err := common.CallTemplateScript(
            cCtx.Context,
            logger,
            dir,
            script,
            common.ExpectNonJSONResponse,
            []byte("--name"), []byte(name),
            []byte("--lang"), []byte(lang),
            []byte("--dir"), []byte(outDir),
            []byte("--template-url"), []byte(cCtx.String("template-url")),
            []byte("--template-version"), []byte(cCtx.String("template-version")),
        )
        if err != nil {
            return fmt.Errorf("create failed: %w", err)
        }
        logger.Info("Container project created: %s", name)
        return nil
    },
}


