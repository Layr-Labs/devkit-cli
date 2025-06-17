package commands

import (
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/urfave/cli/v2"

	"github.com/Layr-Labs/multichain-go/pkg/blsSigner"
	"github.com/Layr-Labs/multichain-go/pkg/chainManager"
	"github.com/Layr-Labs/multichain-go/pkg/logger"
	"github.com/Layr-Labs/multichain-go/pkg/operatorTableCalculator"
	"github.com/Layr-Labs/multichain-go/pkg/transport"
	"github.com/Layr-Labs/multichain-go/pkg/txSigner"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/robfig/cron/v3"
)

var TransportCommand = &cli.Command{
	Name:  "transport",
	Usage: "Transport Stake Root to L1",
	Subcommands: []*cli.Command{
		{
			Name:  "run",
			Usage: "Immediately transport stake root to L1",
			Flags: append([]cli.Flag{}, common.GlobalFlags...),
			Action: func(cCtx *cli.Context) error {
				// Invoke and return Transport
				return Transport(cCtx)
			},
		},
		{
			Name:  "schedule",
			Usage: "Schedule transport stake root to L1",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:  "cron-expr",
					Usage: "Specify a custom schedule to override config schedule",
					Value: "",
				},
			}, common.GlobalFlags...),
			Action: func(cCtx *cli.Context) error {
				// Extract context
				cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
				if err != nil {
					return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
				}
				envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
				if !ok {
					return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
				}

				// Extract cron-expr from flag or context
				schedule := cCtx.String("cron-expr")
				if schedule == "" {
					schedule = envCtx.Transporter.Schedule
				}

				// Invoke ScheduleTransport with configured schedule
				err = ScheduleTransport(cCtx, schedule)
				if err != nil {
					return fmt.Errorf("ScheduleTransport failed: %v", err)
				}

				// Keep process alive
				select {}
			},
		},
	},
}

func Transport(cCtx *cli.Context) error {
	// Get a raw zap logger to pass to operatorTableCalculator and transport
	rawLogger, err := logger.NewLogger(&logger.LoggerConfig{Debug: true})
	if err != nil {
		panic(err)
	}

	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)

	// Extract context
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
	}
	// Get the values from env/config
	crossChainRegistryAddress := ethcommon.HexToAddress(envCtx.EigenLayer.L1.CrossChainRegistry)
	rpcUrl, err := devnet.GetDevnetRPCUrlDefault(cfg, devnet.L1)
	if err != nil {
		rpcUrl = "http://localhost:8545"
	}
	chainId, err := devnet.GetDevnetChainIdOrDefault(cfg, devnet.L1, logger)
	if err != nil {
		chainId = common.DefaultAnvilChainId
	}

	cm := chainManager.NewChainManager()

	holeskyConfig := &chainManager.ChainConfig{
		ChainID: uint64(chainId),
		RPCUrl:  rpcUrl,
	}
	if err := cm.AddChain(holeskyConfig); err != nil {
		return fmt.Errorf("Failed to add chain: %v", err)
	}
	holeskyClient, err := cm.GetChainForId(holeskyConfig.ChainID)
	if err != nil {
		return fmt.Errorf("Failed to get chain for ID %d: %v", holeskyConfig.ChainID, err)
	}

	txSign, err := txSigner.NewPrivateKeySigner(envCtx.Transporter.PrivateKey)
	if err != nil {
		return fmt.Errorf("Failed to create private key signer: %v", err)
	}

	tableCalc, err := operatorTableCalculator.NewStakeTableRootCalculator(&operatorTableCalculator.Config{
		CrossChainRegistryAddress: crossChainRegistryAddress,
	}, holeskyClient.RPCClient, rawLogger)
	if err != nil {
		return fmt.Errorf("Failed to create StakeTableRootCalculator: %v", err)
	}

	block, err := holeskyClient.RPCClient.BlockByNumber(cCtx.Context, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return fmt.Errorf("Failed to get block by number: %v", err)
	}

	root, tree, dist, err := tableCalc.CalculateStakeTableRoot(cCtx.Context, block.NumberU64())
	if err != nil {
		return fmt.Errorf("Failed to calculate stake table root: %v", err)
	}

	scheme := bn254.NewScheme()
	genericPk, err := scheme.NewPrivateKeyFromHexString(envCtx.Transporter.BlsPrivateKey)
	if err != nil {
		return fmt.Errorf("Failed to create BLS private key: %v", err)
	}
	pk, err := bn254.NewPrivateKeyFromBytes(genericPk.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to convert BLS private key: %v", err)
	}

	inMemSigner, err := blsSigner.NewInMemoryBLSSigner(pk)
	if err != nil {
		return fmt.Errorf("Failed to create in-memory BLS signer: %v", err)
	}

	stakeTransport, err := transport.NewTransport(
		&transport.TransportConfig{
			L1CrossChainRegistryAddress: crossChainRegistryAddress,
		},
		holeskyClient.RPCClient,
		inMemSigner,
		txSign,
		cm,
		rawLogger,
	)
	if err != nil {
		return fmt.Errorf("Failed to create transport: %v", err)
	}

	referenceTimestamp := uint32(block.Time())

	err = stakeTransport.SignAndTransportGlobalTableRoot(
		root,
		referenceTimestamp,
		block.NumberU64(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to sign and transport global table root: %v", err)
	}
	logger.Info("Successfully signed and transported global table root, sleeping for 25 seconds")
	time.Sleep(25 * time.Second)

	opsets := dist.GetOperatorSets()
	if len(opsets) == 0 {
		return fmt.Errorf("No operator sets found, skipping AVS stake table transport")
	}
	for _, opset := range opsets {
		err = stakeTransport.SignAndTransportAvsStakeTable(
			referenceTimestamp,
			block.NumberU64(),
			opset,
			root,
			tree,
			dist,
			nil,
		)
		if err != nil {
			return fmt.Errorf("Failed to sign and transport AVS stake table for opset %v: %v", opset, err)
		}

		// log success
		logger.Info("Successfully signed and transported AVS stake table for opset %v", opset)
	}

	return nil
}

func ScheduleTransport(cCtx *cli.Context, cronExpr string) error {
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err := parser.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Call Transport() against cronExpr
	c := cron.New()
	_, err = c.AddFunc(cronExpr, func() {
		if err := Transport(cCtx); err != nil {
			log.Printf("Scheduled transport failed: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add transport function to scheduler: %w", err)
	}

	// Start the scheduled runner
	c.Start()
	log.Println("Transport scheduler started.")
	entries := c.Entries()
	if len(entries) > 0 {
		log.Printf("Next scheduled transport at: %s", entries[0].Next.Format(time.RFC3339))
	}

	// If the Context closes, stop the scheduler
	<-cCtx.Context.Done()
	c.Stop()
	log.Println("Transport scheduler stopped.")
	return nil
}
