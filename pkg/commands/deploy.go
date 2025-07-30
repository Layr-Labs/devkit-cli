package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// DeployCommand defines the "deploy" command
var DeployCommand = &cli.Command{
	Name:  "deploy",
	Usage: "Deploy AVS components to specified network",
	Subcommands: []*cli.Command{
		{
			Name:  "contracts",
			Usage: "Deploy contracts to specified network",
			Subcommands: []*cli.Command{
				{
					Name:  "l1",
					Usage: "Deploy L1 contracts to specified network",
					Flags: append([]cli.Flag{
						&cli.StringFlag{
							Name:  "context",
							Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
						},
						&cli.BoolFlag{
							Name:  "skip-setup",
							Usage: "Skip AVS setup steps (metadata update, registrar setup, etc.) after contract deployment",
							Value: false,
						},
						&cli.BoolFlag{
							Name:  "use-zeus",
							Usage: "Use Zeus CLI to fetch l1(*) and l2(*) core addresses",
							Value: true,
						},
					}, common.GlobalFlags...),
					Action: StartDeployL1Action,
				},
				{
					Name:  "l2",
					Usage: "Deploy L2 contracts to specified network",
					Flags: append([]cli.Flag{
						&cli.StringFlag{
							Name:  "context",
							Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
						},
					}, common.GlobalFlags...),
					Action: StartDeployL2Action,
				},
			},
		},
	},
}
