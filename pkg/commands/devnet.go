package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
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
			Flags: append([]cli.Flag{
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
				&cli.BoolFlag{
					Name:  "skip-avs-run",
					Usage: "Skip starting offchain AVS components",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "skip-transporter",
					Usage: "Skip starting/submitting Stake Root via transporter",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "skip-deploy-contracts",
					Usage: "Skip deploying contracts and only start local devnet",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "skip-setup",
					Usage: "Skip AVS setup steps (metadata update, registrar setup, etc.) after contract deployment",
					Value: false,
				},
				&cli.BoolFlag{
					Name:  "use-zeus",
					Usage: "Use Zeus CLI to fetch holesky core addresses",
					Value: false,
				},
			}, common.GlobalFlags...),
			Action: StartDevnetAction,
		},
		{
			Name:   "deploy-contracts",
			Usage:  "Deploy all L1/L2 and AVS contracts to devnet",
			Flags:  []cli.Flag{},
			Action: DeployContractsAction,
		},
		{
			Name:  "stop",
			Usage: "Stops and removes all containers and resources",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "all",
					Usage: "Stop all running devnet containers",
				},
				&cli.StringFlag{
					Name:  "project.name",
					Usage: "Stop containers associated with the given project name",
				},
				&cli.IntFlag{
					Name:  "port",
					Usage: "Stop container running on the specified port",
				},
			},
			Action: StopDevnetAction,
		},
		{
			Name:   "list",
			Usage:  "Lists all running devkit devnet containers with their ports",
			Action: ListDevnetContainersAction,
		},
		{
			Name:   "fetch-addresses",
			Usage:  "Fetches current EigenLayer core addresses from holesky using Zeus CLI",
			Action: FetchZeusAddressesAction,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Context to update with Zeus addresses",
					Value: "devnet",
				},
			},
		},
		// TODO: Surface the following actions as separate commands:
		// - update-avs-metadata: Updates the AVS metadata URI on the devnet
		// - set-avs-registrar: Sets the AVS registrar address on the devnet
		// - create-avs-operator-sets: Creates AVS operator sets on the devnet
		// - register-operators-from-config: Registers operators defined in config to Eigenlayer and the AVS on the devnet
	},
}
