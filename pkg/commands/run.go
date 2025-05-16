package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

// RunCommand defines the "run" command
var RunCommand = &cli.Command{
	Name:  "run",
	Usage: "Submits tasks to the local devnet, triggers off-chain execution, and aggregates results",
	Flags: append([]cli.Flag{}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		if cCtx.Bool("verbose") {
			log.Printf("Running AVS tasks...")
		}

		err := common.CallMakefileTarget(cCtx.Context, filepath.Join("contracts"), common.Makefile, "run")
		if err != nil {
			return fmt.Errorf("failed to call make run in Makefile.Devkit %w", err)
		}

		log.Printf("Task execution completed successfully")
		return nil
	},
}
