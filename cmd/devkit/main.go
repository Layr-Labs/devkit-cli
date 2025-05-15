package main

import (
	"context"
	"devkit-cli/pkg/plugin"
	"fmt"
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

	plugins, err := plugin.LoadPlugins("~/.devkit/plugins")
	if err != nil {
		log.Fatalf("Error loading plugins: %v", err)
	}
	fmt.Printf("Plugins loaded: %+v\n", plugins)
	for _, p := range plugins {
		if p != nil {
			app.Commands = append(app.Commands, p.GetCommands()...)
		}
	}

	// Apply both middleware functions to all commands
	hooks.ApplyMiddleware(app.Commands, hooks.WithEnvLoader, hooks.WithTelemetry)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
