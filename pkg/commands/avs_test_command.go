package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"
)

// TestCommand defines the "test" command
var TestCommand = &cli.Command{
	Name:  "test",
	Usage: "Submits tasks to the local devnet, triggers off-chain execution, and aggregates results",
	Flags: append([]cli.Flag{}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		if cCtx.Bool("verbose") {
			log.Printf("Testing AVS tasks...")
		}

		err := common.CallDevkitMakeTarget(cCtx.Context, "test")
		if err != nil {
			return fmt.Errorf("failed to call make run in Makefile.Devkit %w", err)
		}

		log.Printf("Task execution completed successfully")
		return nil
	},
}
