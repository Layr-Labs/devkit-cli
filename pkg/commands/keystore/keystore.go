package keystore

import (
	"github.com/urfave/cli/v2"
)

var KeystoreCommand = &cli.Command{
	Name:  "keystore",
	Usage: "Manage keystore operations",
	Subcommands: []*cli.Command{
		CreateCommand,
		ReadCommand,
	},
}
