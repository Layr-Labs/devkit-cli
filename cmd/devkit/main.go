package main

import (
	"context"
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/commands/keystore"
	"devkit-cli/pkg/common"
	devcontext "devkit-cli/pkg/context"
	"devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx := devcontext.WithShutdown(context.Background())

	app := &cli.App{
		Name:                   "devkit",
		Usage:                  "EigenLayer Development Kit",
		Flags:                  common.GlobalFlags,
		Commands:               []*cli.Command{commands.AVSCommand, keystore.KeystoreCommand},
		UseShortOptionHandling: true,
	}

	middleware := hooks.NewCommandMiddleware()

	middleware.AddPreProcessor(hooks.WithEnvLoader())
	middleware.AddPreProcessor(hooks.WithTelemetryPreProcessor())

	middleware.AddPostProcessor(hooks.WithTelemetryPostProcessor())

	hooks.ApplyMiddleware(app.Commands, middleware)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
