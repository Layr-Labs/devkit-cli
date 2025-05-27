package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/commands/template"
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
		CallCommand,
		ReleaseCommand,
		template.Command,
	},
}
