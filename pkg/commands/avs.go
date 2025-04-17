package commands

import (
	"github.com/urfave/cli/v2"
)

// AVSCommand defines the top-level "avs" command
var AVSCommand = &cli.Command{
	Name:  "avs",
	Usage: "Manage EigenLayer AVS (Actively Validated Services) projects",
	Subcommands: []*cli.Command{
		CreateCommand,
		ConfigCommand,
		BuildCommand,
		DevnetCommand,
		RunCommand,
		ReleaseCommand,
	},
}
