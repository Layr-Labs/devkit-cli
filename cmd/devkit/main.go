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
		Name:                   "devkit",
		Usage:                  "EigenLayer Development Kit",
		Flags:                  common.GlobalFlags,
		Commands:               []*cli.Command{commands.AVSCommand, keystore.KeystoreCommand},
		UseShortOptionHandling: true,
	}

	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithEnvLoader)
	actionChain.Use(hooks.WithTelemetry)

	hooks.ApplyMiddleware(app.Commands, actionChain)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
