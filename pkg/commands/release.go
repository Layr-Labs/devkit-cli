package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"log"

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

		if cCtx.Bool("verbose") {
			log.Printf("Preparing release...")
			log.Printf("Tag: %s", cCtx.String("tag"))
			if registry := cCtx.String("registry"); registry != "" {
				log.Printf("Registry: %s", registry)
			}
			if cCtx.Bool("sign") {
				log.Printf("Signing release artifacts...")
			}
		}

		// Placeholder for future implementation
		log.Printf("Release completed successfully")
		return nil
	},
}
