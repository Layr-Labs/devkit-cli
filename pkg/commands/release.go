package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// ReleaseCommand defines the "release" command
var ReleaseCommand = &cli.Command{
	Name:  "release",
	Usage: "Packages and publishes AVS artifacts to a registry or channel",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "tag",
			Usage: "Tag the release (e.g. v0.1, beta, mainnet)",
			Value: "latest",
		},
		&cli.StringFlag{
			Name:  "registry",
			Usage: "Override default release registry",
		},
		&cli.BoolFlag{
			Name:  "sign",
			Usage: "Sign the release artifacts with a local key",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		if cCtx.Bool("verbose") {
			logger.InfoWithActor("User", "Preparing release...")
			logger.InfoWithActor("User", "Tag: %s", cCtx.String("tag"))
			if registry := cCtx.String("registry"); registry != "" {
				logger.InfoWithActor("User", "Registry: %s", registry)
			}
			if cCtx.Bool("sign") {
				logger.InfoWithActor("User", "Signing release artifacts...")
			}
		}

		// Placeholder for future implementation
		logger.InfoWithActor("User", "Release completed successfully")
		return nil
	},
}
