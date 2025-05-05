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

	printBanner()

	app := &cli.App{
		Name:                   "DevKit",
		Usage:                  "EigenLayer Development Kit",
		Flags:                  common.GlobalFlags,
		Commands:               []*cli.Command{commands.AVSCommand},
		UseShortOptionHandling: true,
	}

	// Apply telemetry middleware to all commands
	hooks.ApplyTelemetryToCommands(app.Commands)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func printBanner() {
	banner := `
 ╭─────────────────────────────────────────╮
 │  E I G E N L A Y E R   D E V K I T      │
 ╰─────────────────────────────────────────╯
`
	println(banner)
}
