package commands

import (
	"devkit-cli/pkg/common"
	"devkit-cli/pkg/common/config"
	"log"
	"os"
	"os/exec"

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
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		var cfg *config.EigenConfig

		// First check if config is in context (for testing)
		if cfgValue := cCtx.Context.Value(ConfigContextKey); cfgValue != nil {
			cfg = cfgValue.(*config.EigenConfig)
		} else {
			// Load from file if not in context
			var err error
			cfg, err = config.LoadEigenConfig()
			if err != nil {
				return err
			}
		}

		if common.IsVerboseEnabled(cCtx, cfg) {
			log.Printf("Project Name: %s", cfg.Project.Name)
			log.Printf("Building AVS components...")
			if cCtx.Bool("release") {
				log.Printf("Building in release mode with image tag: %s", cfg.Release.AVSLogicImageTag)
			}
		}

		// Execute make build with Makefile.Devkit
		cmd := exec.Command("make", "-f", "Makefile.Devkit", "build")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}

		log.Printf("Build completed successfully")
		return nil
	},
}
