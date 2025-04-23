package commands

import (
	"devkit-cli/pkg/common"
	"log"

	"github.com/urfave/cli/v2"
)

// DevnetCommand defines the "devnet" command
var DevnetCommand = &cli.Command{
	Name:  "devnet",
	Usage: "Manage local AVS development network (Docker-based)",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "Starts Docker containers and deploys local contracts",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "reset",
					Usage: "Wipe and restart the devnet from scratch",
				},
				&cli.StringFlag{
					Name:  "fork",
					Usage: "Fork from a specific chain (e.g. Base, OP)",
				},
				&cli.BoolFlag{
					Name:  "headless",
					Usage: "Run without showing logs or interactive TUI",
				},
				&cli.IntFlag{
					Name:  "port",
					Usage: "Specify a custom port for local devnet",
					Value: 8545,
				},
			},
			Action: func(cCtx *cli.Context) error {
				config := cCtx.Context.Value(ConfigContextKey).(*common.EigenConfig)

				if cCtx.Bool("verbose") {
					log.Printf("Starting devnet... ")

					if cCtx.Bool("reset") {
						log.Printf("Resetting devnet...")
					}
					if fork := cCtx.String("fork"); fork != "" {
						log.Printf("Forking from chain: %s", fork)
					}
					if cCtx.Bool("headless") {
						log.Printf("Running in headless mode")
					}
					log.Printf("Port: %d", cCtx.Int("port"))
					chain_image := config.Env["devnet"].ChainImage
					if chain_image == "" {
						log.Printf("chain image not provided")
					} else {
						log.Printf("Chain Image: %s", chain_image)
					}
				}

				log.Printf("Devnet started successfully")
				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "Stops and removes all containers and resources",
			Action: func(cCtx *cli.Context) error {
				if cCtx.Bool("verbose") {
					log.Printf("Stopping devnet...")
				}
				log.Printf("Devnet stopped successfully")
				return nil
			},
		},
	},
}
