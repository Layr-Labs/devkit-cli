package commands

import (
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
			Action: StartDevnetAction,
		},
		{
			Name:   "stop",
			Usage:  "Stops and removes all containers and resources",
			Action: StopDevnetAction,
		},
		{
			Name:   "list",
			Usage:  "Lists all running devkit devnet containers with their ports",
			Action: ListDevnetContainersAction,
		},
	},
}
