package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

// BuildCommand defines the "build" command
var BuildCommand = &cli.Command{
	Name:  "build",
	Usage: "Compiles AVS components (smart contracts via Foundry, Go binaries for operators/aggregators)",
	Flags: append([]cli.Flag{
		// TBD: Release flag will be implemented in future
		/*&cli.BoolFlag{
			Name:  "release",
			Usage: "Produce production-optimized artifacts",
		},*/
		&cli.StringFlag{
			Name:  "context",
			Usage: "Context : devnet , testnet or mainnet",
			Value: "devnet",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		var cfg *common.BaseConfig

		// First check if config is in context (for testing)
		if cfgValue := cCtx.Context.Value(ConfigContextKey); cfgValue != nil {
			cfg = cfgValue.(*common.BaseConfig)
		} else {

			context := cCtx.String("context")
			// Load from file if not in context
			var err error
			cfg, err = common.LoadBaseConfig(context)
			if err != nil {
				return err
			}
		}

		if common.IsVerboseEnabled(cCtx, cfg) {
			log.Printf("Project Name: %s", cfg.Config.Project.Name)
			log.Printf("Building AVS components...")
			if cCtx.Bool("release") {
				// log.Printf("Building in release mode with image tag: %s", cfg.Release.AVSLogicImageTag)
			}
		}

		// Execute make build with Makefile.Devkit
		cmd := exec.Command("make", "-f", common.DevkitMakefile, "build")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		// Build contracts if available
		if err := buildContractsIfAvailable(); err != nil {
			return err
		}

		log.Printf("Build completed successfully")
		return nil
	},
}

// buildContractsIfAvailable builds the contracts if the contracts directory exists
func buildContractsIfAvailable() error {
	contractsDir := common.ContractsDir
	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		return nil
	}

	makefilePath := filepath.Join(contractsDir, common.DevkitMakefile)
	if _, err := os.Stat(makefilePath); os.IsNotExist(err) {
		return fmt.Errorf("contracts directory exists but no %s found", common.DevkitMakefile)
	}

	cmd := exec.Command("make", "-f", common.DevkitMakefile, "build")
	cmd.Dir = contractsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
