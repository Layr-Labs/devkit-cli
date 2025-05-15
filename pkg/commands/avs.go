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

func MergeCommands(cmds ...*cli.Command) []*cli.Command {
	merged := make([]*cli.Command, 0)
	for _, cmd := range cmds {
		merged = append(merged, cmd)
	}
	return merged
}
