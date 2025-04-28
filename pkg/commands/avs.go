package commands

import (
	"github.com/urfave/cli/v2"
)

var AVSCommand = &cli.Command{
	Name:  "avs",
	Usage: "Manage EigenLayer AVS (Autonomous Verifiable Services) projects",
	Subcommands: []*cli.Command{
		CreateCommand,
		ConfigCommand,
		BuildCommand,
		DevnetCommand,
		RunCommand,
		ReleaseCommand,
	},
}
