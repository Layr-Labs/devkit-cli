package main

import (
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                   "devkit",
		Usage:                  "EigenLayer Development Kit",
		Flags:                  common.GlobalFlags,
		Commands:               []*cli.Command{commands.AVSCommand},
		UseShortOptionHandling: true,
	}

	// Apply both middleware functions to all commands
	hooks.ApplyMiddleware(app.Commands, hooks.WithEnvLoader, hooks.WithTelemetry)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
