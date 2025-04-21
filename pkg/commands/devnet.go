package commands

import (
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"os/exec"
	"time"
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
				cmd.Env = append(os.Environ(), "FOUNDRY_IMAGE=ghcr.io/foundry-rs/foundry:latest") //TODO(supernova): Get this value from  eigen.toml .
				cmd.Env = append(os.Environ(), "ANVIL_ARGS=--block-time 3")
				cmd.Run()
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
