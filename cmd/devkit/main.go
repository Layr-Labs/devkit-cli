package main

import (
	"context"
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/commands/keystore"
	"devkit-cli/pkg/common"
	kitcontext "devkit-cli/pkg/context"
	"devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx := kitcontext.WithShutdown(context.Background())

	app := &cli.App{
		Name:  "devkit",
		Usage: "EigenLayer Development Kit",
		Flags: common.GlobalFlags,
		Before: func(ctx *cli.Context) error {
			err := hooks.LoadEnvFile(ctx)
			if err != nil {
				return err
			}
			hooks.WithAppEnvironment(ctx)
			return hooks.WithCommandMetricsContext(ctx)
		},
		Commands:               []*cli.Command{commands.AVSCommand, keystore.KeystoreCommand},
		UseShortOptionHandling: true,
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
