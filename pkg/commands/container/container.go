package container

import (
	"github.com/urfave/cli/v2"
)

// Command is the top-level `devkit container` command.
var Command = &cli.Command{
	Name:  "container",
	Usage: "Manage TEE container projects and deployments",
	Subcommands: []*cli.Command{
		CreateCommand,
		InitCommand,
		BuildCommand,
		UpCommand,
		DeployCommand,
	},
}


