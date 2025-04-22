package commands

import (
	"devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// AVSCommand defines the top-level "avs" command
var AVSCommand = &cli.Command{
	Name:  "avs",
	Usage: "Manage EigenLayer AVS (Autonomous Verifiable Services) projects",
	Flags: common.GlobalFlags,
	Subcommands: []*cli.Command{
		CreateCommand,
		ConfigCommand,
		BuildCommand,
		DevnetCommand,
		RunCommand,
		ReleaseCommand,
	},
}
