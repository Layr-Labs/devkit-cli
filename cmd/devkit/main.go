package main

import (
	"log"
	"os"

	"devkit-cli/pkg/commands"
	"devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "devkit",
		Usage: "EigenLayer Development Kit",
		Flags: common.GlobalFlags,
		Commands: []*cli.Command{
			commands.AVSCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
