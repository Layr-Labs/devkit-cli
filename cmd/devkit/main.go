package main

import (
	"context"
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/commands/keystore"
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx := common.WithShutdown(context.Background())

	app := &cli.App{
		Name:  "devkit",
		Usage: "EigenLayer Development Kit",
		Flags: common.GlobalFlags,
		Before: func(ctx *cli.Context) error {
			err := hooks.LoadEnvFile(ctx)
			if err != nil {
				return err
			}
			common.WithAppEnvironment(ctx)
			return hooks.WithCommandMetricsContext(ctx)
		},
		Commands:               []*cli.Command{commands.AVSCommand, keystore.KeystoreCommand},
		UseShortOptionHandling: true,
	}

	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithMetricEmission)

	hooks.ApplyMiddleware(app.Commands, actionChain)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
