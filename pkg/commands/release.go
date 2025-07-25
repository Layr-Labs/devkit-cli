package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/urfave/cli/v2"
)

// OperatorSetRelease represents the data for each operator set
type OperatorSetRelease struct {
	Digest      string `json:"digest"`
	Registry    string `json:"registry"`
	RuntimeSpec string `json:"runtimeSpec,omitempty"` // YAML content of the runtime spec
}

// ReleaseCommand defines the "release" command
var ReleaseCommand = &cli.Command{
	Name:  "release",
	Usage: "Manage AVS releases and artifacts",
	Subcommands: []*cli.Command{
		{
			Name:  "publish",
			Usage: "Publish a new AVS release",
			Flags: append(common.GlobalFlags, []cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
				},
				&cli.Int64Flag{
					Name:     "upgrade-by-time",
					Usage:    "Unix timestamp by which the upgrade must be completed",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "registry",
					Usage: "Registry to use for the release. If not provided, will use registry from context",
				},
			}...),
			Action: publishReleaseAction,
		},
		{
			Name:  "uri",
			Usage: "Set release metadata URI for an operator set",
			Flags: append(common.GlobalFlags, []cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
				},
				&cli.StringFlag{
					Name:     "metadata-uri",
					Usage:    "Metadata URI to set for the release",
					Required: true,
				},
				&cli.UintFlag{
					Name:     "operator-set-id",
					Usage:    "Operator set ID",
					Required: true,
				},
				&cli.StringFlag{
					Name:  "avs-address",
					Usage: "AVS address (if not provided, will use from context)",
				},
			}...),
			Action: setReleaseMetadataURIAction,
		},
	},
}
