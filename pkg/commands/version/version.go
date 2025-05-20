package version

import (
	"fmt"
	"github.com/Layr-Labs/devkit-cli/internal/version"
	"github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/urfave/cli/v2"
)

// RunCommand defines the "run" command
var VersionCommand = &cli.Command{
	Name:  "version",
	Usage: "Print the version of the devkit",
	Flags: append([]cli.Flag{}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		// Invoke and return AVSRun
		return VersionRun(cCtx)
	},
}

func VersionRun(cCtx *cli.Context) error {
	v := version.GetVersion()
	commit := version.GetCommit()

	fmt.Printf("Version: %s\nCommit: %s\n", v, commit)

	return nil
}
