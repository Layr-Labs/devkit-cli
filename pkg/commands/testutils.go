package commands

import (
	"context"
	"devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

type ctxKey string

// ConfigContextKey identifies the eigenConfig in context
const ConfigContextKey ctxKey = "eigenConfig"

func WithTestConfig(cmd *cli.Command) *cli.Command {
	cmd.Before = func(cCtx *cli.Context) error {
		cfg := &common.EigenConfig{
			// Optionally mock config values if needed
			Project: common.ProjectConfig{
				Name: "test-avs",
			},
		}
		ctx := context.WithValue(cCtx.Context, ConfigContextKey, cfg)
		cCtx.Context = ctx
		return nil
	}
	return cmd
}
