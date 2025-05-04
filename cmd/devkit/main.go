package main

import (
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/commands/keystore"
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                   "devkit",
		Usage:                  "EigenLayer Development Kit",
		Flags:                  common.GlobalFlags,
		Commands:               []*cli.Command{commands.AVSCommand, keystore.KeystoreCommand},
		UseShortOptionHandling: true,
	}

	// Apply telemetry middleware to all commands
	hooks.ApplyTelemetryToCommands(app.Commands)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
