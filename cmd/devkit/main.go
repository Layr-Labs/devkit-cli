package main

import (
	"context"
	"fmt"
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
		EnableBashCompletion: true,
		Name:                 "devkit",
		Usage:                "EigenLayer Development Kit",
		Flags:                common.GlobalFlags,
		Before: func(cCtx *cli.Context) error {
			err := hooks.LoadEnvFile(cCtx)
			if err != nil {
				return err
			}
			common.WithAppEnvironment(cCtx)

			// Parse verbose flags from raw argv to capture from subcommand flags
			verbose := common.PeelBoolFromFlags(os.Args[1:], "--verbose", "-v")
			// Set verbose directly if it appears in subcommand flags
			if verbose {
				err := cCtx.Set("verbose", "true")
				if err != nil {
					return fmt.Errorf("failed to set verbose flag globally: %w", err)
				}
			}

			// Get logger based on CLI context (handles verbosity internally)
			logger, tracker := common.GetLoggerFromCLIContext(cCtx)

			// Store logger and tracker in the context
			cCtx.Context = common.WithLogger(cCtx.Context, logger)
			cCtx.Context = common.WithProgressTracker(cCtx.Context, tracker)

			// Handle first-run telemetry prompt (only for non-telemetry commands)
			if cCtx.Command.Name != "telemetry" && cCtx.Command.Name != "help" && cCtx.Command.Name != "version" {
				if err := hooks.WithFirstRunTelemetryPrompt(cCtx); err != nil {
					// Log error but don't fail the command
					logger.Debug("First-run telemetry prompt failed: %v", err)
				}
			}

			return hooks.WithCommandMetricsContext(cCtx)
		},
		Commands: []*cli.Command{
			commands.AVSCommand,
			keystore.KeystoreCommand,
			version.VersionCommand,
			commands.UpgradeCommand,
			commands.TelemetryCommand,
		},
		UseShortOptionHandling: true,
	}

	actionChain := hooks.NewActionChain()
	actionChain.Use(hooks.WithMetricEmission)

	hooks.ApplyMiddleware(app.Commands, actionChain)

	if err := app.RunContext(ctx, os.Args); err != nil {
		log.Fatal(err)
	}
}
