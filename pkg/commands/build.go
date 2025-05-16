package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/testutils"
	"fmt"
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
			Usage: "devnet ,testnet or mainnet",
			Value: "devnet",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		log, _ := common.GetLogger()

		var cfg *common.ConfigWithContextConfig

		// First check if config is in context (for testing)
		if cfgValue := cCtx.Context.Value(testutils.ConfigContextKey); cfgValue != nil {
			cfg = cfgValue.(*common.ConfigWithContextConfig)
		} else {

			context := cCtx.String("context")
			// Load from file if not in context
			var err error
			cfg, err = common.LoadConfigWithContextConfig(context)
			if err != nil {
				return err
			}
		}

		if common.IsVerboseEnabled(cCtx, cfg) {
			log.Info("Project Name: %s", cfg.Config.Project.Name)
			log.Info("Building AVS components...")

		}

		// All scripts contained here
		scriptsDir := filepath.Join(".devkit", "scripts")

		// Execute build via .devkit scripts
		cmd := exec.CommandContext(cCtx.Context, "bash", "-c", filepath.Join(scriptsDir, "build"))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		// Build contracts if available
		if err := buildContractsIfAvailable(cCtx); err != nil {
			return err
		}

		log.Info("Build completed successfully")
		return nil
	},
}

// buildContractsIfAvailable builds the contracts if the contracts directory exists
func buildContractsIfAvailable(cCtx *cli.Context) error {
	log, _ := common.GetLogger()
	contractsDir := common.ContractsDir
	if _, err := os.Stat(contractsDir); os.IsNotExist(err) {
		return nil
	}

	configPath := filepath.Join(contractsDir, common.ContractsMakefile)
	log.Info(configPath)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("contracts directory exists but no %s found", common.ContractsMakefile)
	}

	cmd := exec.CommandContext(cCtx.Context, "make", "-f", common.ContractsMakefile, "build")
	cmd.Dir = contractsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if cmdErr := cmd.Run(); cmdErr != nil {
		return fmt.Errorf("build failed %w", cmdErr)
	}

	return nil
}
