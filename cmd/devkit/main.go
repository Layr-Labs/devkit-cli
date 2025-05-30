package main

import (
	"context"
	"log"
	"os"

	"github.com/Layr-Labs/devkit-cli/pkg/commands"
	"github.com/Layr-Labs/devkit-cli/pkg/commands/keystore"
	"github.com/Layr-Labs/devkit-cli/pkg/commands/version"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/hooks"

	"github.com/urfave/cli/v2"
)

func main() {
	ctx := common.WithShutdown(context.Background())

	app := &cli.App{
		Name:  "devkit",
		Usage: "EigenLayer Development Kit",
		Flags: common.GlobalFlags,
		Before: func(cCtx *cli.Context) error {
			err := hooks.LoadEnvFile(cCtx)
			if err != nil {
				return err
			}
			common.WithAppEnvironment(cCtx)

			// Get logger based on CLI context (handles verbosity internally)
			logger, tracker := common.GetLoggerFromCLIContext(cCtx)

			// Store logger and tracker in the context
			cCtx.Context = common.WithLogger(cCtx.Context, logger)
			cCtx.Context = common.WithProgressTracker(cCtx.Context, tracker)

			return hooks.WithCommandMetricsContext(cCtx)
		},
		Commands: []*cli.Command{
			commands.AVSCommand,
			keystore.KeystoreCommand,
			version.VersionCommand,
		},
		UseShortOptionHandling: true,
	}

	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithCommandDependencyCheck)
	actionChain.Use(hooks.WithMetricEmission)

	hooks.ApplyMiddleware(app.Commands, actionChain)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
