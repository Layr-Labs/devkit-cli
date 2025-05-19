package commands

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/testutils"
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

		log.Info("Build completed successfully")
		return nil
	},
}
