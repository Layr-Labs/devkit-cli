package commands

import (
	"context"
	"devkit-cli/pkg/common"
	"fmt"

	"github.com/urfave/cli/v2"
)

type ctxKey string

const ConfigContextKey ctxKey = "eigenConfig"

var AVSCommand = &cli.Command{
	Name:  "avs",
	Usage: "Manage EigenLayer AVS (Actively Validated Services) projects",
	Before: func(cCtx *cli.Context) error {
		cfg, err := common.LoadEigenConfig()
		if err != nil {
			return fmt.Errorf("failed to load eigen.toml: %w", err)
		}
		ctx := context.WithValue(cCtx.Context, ConfigContextKey, cfg)
		cCtx.Context = ctx
		return nil
	},
	Subcommands: []*cli.Command{
		CreateCommand,
		ConfigCommand,
		BuildCommand,
		DevnetCommand,
		RunCommand,
		ReleaseCommand,
	},
}
