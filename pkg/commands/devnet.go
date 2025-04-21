package commands

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/urfave/cli/v2"
)

// DevnetCommand defines the "devnet" command
var DevnetCommand = &cli.Command{
	Name:  "devnet",
	Usage: "Manage local AVS development network (Docker-based)",
	Subcommands: []*cli.Command{
		{
			Name:  "start",
			Usage: "Starts Docker containers and deploys local contracts",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "reset",
					Usage: "Wipe and restart the devnet from scratch",
				},
				&cli.StringFlag{
					Name:  "fork",
					Usage: "Fork from a specific chain (e.g. Base, OP)",
				},
				&cli.BoolFlag{
					Name:  "headless",
					Usage: "Run without showing logs or interactive TUI",
				},
				&cli.IntFlag{
					Name:  "port",
					Usage: "Specify a custom port for local devnet",
					Value: 8545,
				},
			},
			Action: func(cCtx *cli.Context) error {
				startTime := time.Now() // <-- start timing
				if cCtx.Bool("verbose") {
					log.Printf("Starting devnet...")
					if cCtx.Bool("reset") {
						log.Printf("Resetting devnet...")
					}
					if fork := cCtx.String("fork"); fork != "" {
						log.Printf("Forking from chain: %s", fork)
					}
					if cCtx.Bool("headless") {
						log.Printf("Running in headless mode")
					}
					log.Printf("Port: %d", cCtx.Int("port"))
				}
				cmd := exec.Command("docker", "compose", "-f", "contracts/anvil/docker-compose.yaml", "up", "-d")
				CHAIN_IMAGE_FROM_TOML := "" // TODO(supernova): Load image from eigen.toml.
				CHAIN_ARGS_FROM_TOML := ""  // TODO(supernova): Load args from eigen.toml
				chain_image := common.GetImageConfigOrDefault(CHAIN_IMAGE_FROM_TOML)
				chain_args := common.GetChainArgsConfigOrDefault(CHAIN_ARGS_FROM_TOML)
				port := cCtx.Int("port")
				rpc_url := fmt.Sprintf("http://localhost:%d", port)
				cmd.Env = append(os.Environ(),
					"FOUNDRY_IMAGE="+chain_image,
					"ANVIL_ARGS="+chain_args,
				)
				err := cmd.Run()
				if err != nil {
					log.Printf("Failed to start devnet %s", err)
				}
				// TODO(supernova): get addresses to fund from eigen.toml.
				common.FundWallets(common.FUND_VALUE, []string{
					"0x70997970c51812dc3a010c7d01b50e0d17dc79c8", // submit wallet
				}, "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80", rpc_url)
				elapsed := time.Since(startTime).Round(time.Second)
				log.Printf("Devnet started successfully in %s", elapsed)

				return nil
			},
		},
		{
			Name:  "stop",
			Usage: "Stops and removes all containers and resources",
			Action: func(cCtx *cli.Context) error {
				if cCtx.Bool("verbose") {
					log.Printf("Attempting to stop devnet containers...")
				}

				// Check if any devnet containers are running
				checkCmd := exec.Command("docker", "ps", "--filter", "name=devkit-devnet", "--format", "{{.Names}}")
				output, err := checkCmd.Output()
				if err != nil {
					log.Fatalf("Failed to check running containers: %v", err)
				}

				if len(output) == 0 {
					log.Printf("No running devkit devnet containers found. Nothing to stop.")
					return nil
				}

				// Stop and remove containers via docker compose
				stopCmd := exec.Command("docker", "compose", "-f", "contracts/anvil/docker-compose.yaml", "down")
				stopCmd.Stdout = os.Stdout
				stopCmd.Stderr = os.Stderr

				if err := stopCmd.Run(); err != nil {
					log.Fatalf("Failed to stop devnet containers: %v", err)
				}

				log.Printf("Devnet containers stopped and removed successfully.")

				return nil
			},
		},
	},
}
