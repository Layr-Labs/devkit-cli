package main

import (
	"context"
	"log"
	"os"

	"devkit-cli/pkg/commands"
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
		Commands:               []*cli.Command{commands.AVSCommand},
		UseShortOptionHandling: true,
	}

	// Apply both middleware functions to all commands
	hooks.ApplyMiddleware(app.Commands, hooks.WithEnvLoader, hooks.WithTelemetry)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
