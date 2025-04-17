package common

import "github.com/urfave/cli/v2"

// GlobalFlags defines flags that apply to the entire application (global flags).
var GlobalFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:    "verbose",
		Aliases: []string{"v"},
		Usage:   "Enable verbose logging",
	},
}
