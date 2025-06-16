package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/commands/config"
	"github.com/Layr-Labs/devkit-cli/pkg/commands/context"
	"github.com/Layr-Labs/devkit-cli/pkg/commands/template"
	"github.com/urfave/cli/v2"
)

var AVSCommand = &cli.Command{
	Name:  "avs",
	Usage: "Manage EigenLayer AVS (Autonomous Verifiable Services) projects",
	Subcommands: []*cli.Command{
		CreateCommand,
		config.Command,
		context.Command,
		BuildCommand,
		DevnetCommand,
		TransportCommand,
		RunCommand,
		CallCommand,
		ReleaseCommand,
		template.Command,
	},
}
