package commands

import (
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ICrossChainRegistry"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IOperatorTableUpdater"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"

	"github.com/Layr-Labs/multichain-go/pkg/blsSigner"
	"github.com/Layr-Labs/multichain-go/pkg/chainManager"
	"github.com/Layr-Labs/multichain-go/pkg/logger"
	"github.com/Layr-Labs/multichain-go/pkg/operatorTableCalculator"
	"github.com/Layr-Labs/multichain-go/pkg/transport"
	"github.com/Layr-Labs/multichain-go/pkg/txSigner"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/robfig/cron/v3"
)

var TransportCommand = &cli.Command{
	Name:  "transport",
	Usage: "Transport Stake Root to L1",
	Subcommands: []*cli.Command{
		{
			Name:   "run",
			Usage:  "Immediately transport stake root to L1",
			Flags:  append([]cli.Flag{}, common.GlobalFlags...),
			Action: Transport,
		},
		{
			Name:   "verify",
			Usage:  "Verify that the context active_stake_roots match onchain state",
			Flags:  append([]cli.Flag{}, common.GlobalFlags...),
			Action: VerifyActiveStakeTableRoots,
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

	// Construct and collate all roots
	roots := make(map[uint64][32]byte)

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
		[]*big.Int{new(big.Int).SetUint64(17000)},
	)
	if err != nil {
		return fmt.Errorf("Failed to sign and transport global table root: %v", err)
	}

	// Collect the provided roots
	roots[holeskyConfig.ChainID] = root

	// Write the roots to context (each time we process one)
	err = WriteStakeTableRootsToContext(roots)
	if err != nil {
		return fmt.Errorf("failed to write active_stake_roots: %w", err)
	}

	// Sleep before transporting AVSStakeTable
	logger.Info("Successfully signed and transported global table root, sleeping for 25 seconds")
	time.Sleep(25 * time.Second)

	// Fetch OperatorSets for AVSStakeTable transport
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
			[]*big.Int{new(big.Int).SetUint64(17000)},
		)
		if err != nil {
			return fmt.Errorf("Failed to sign and transport AVS stake table for opset %v: %v", opset, err)
		}

		// log success
		logger.Info("Successfully signed and transported AVS stake table for opset %v", opset)
	}

	return nil
}

// Record StakeTableRoots in the context for later retrieval
func WriteStakeTableRootsToContext(roots map[uint64][32]byte) error {
	// Load and navigate context to arrive at context.transporter.active_stake_roots
	yamlPath, rootNode, contextNode, err := common.LoadContext("devnet") // @TODO: use selected context name
	if err != nil {
		return err
	}
	transporterNode := common.GetChildByKey(contextNode, "transporter")
	if transporterNode == nil {
		return fmt.Errorf("'transporter' section missing in context")
	}
	activeRootsNode := common.GetChildByKey(transporterNode, "active_stake_roots")
	if activeRootsNode == nil {
		activeRootsNode = &yaml.Node{
			Kind:    yaml.SequenceNode,
			Tag:     "!!seq",
			Content: []*yaml.Node{},
		}
		// insert key-value into transporter
		transporterNode.Content = append(transporterNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "active_stake_roots"},
			activeRootsNode,
		)
	} else if activeRootsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("'active_stake_roots' exists but is not a list")
	}

	// Force block style on activeRootsNode to prevent collapse
	activeRootsNode.Style = 0

	// Construct index of the context stored roots
	indexByChainID := make(map[uint64]int)
	for idx, node := range activeRootsNode.Content {
		if node.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i < len(node.Content)-1; i += 2 {
			if node.Content[i].Value == "chain_id" {
				cid, err := strconv.ParseUint(node.Content[i+1].Value, 10, 64)
				if err == nil {
					indexByChainID[cid] = idx
				}
			}
		}
	}

	// Append roots to the context
	for chainID, root := range roots {
		hexRoot := fmt.Sprintf("0x%x", root)

		// Check for entry for this chainId
		if idx, ok := indexByChainID[chainID]; ok {
			// Update stake_root field in existing node
			entry := activeRootsNode.Content[idx]
			found := false
			for i := 0; i < len(entry.Content)-1; i += 2 {
				if entry.Content[i].Value == "stake_root" {
					entry.Content[i+1].Value = hexRoot
					found = true
					break
				}
			}
			// If stake_root missing, insert it
			if !found {
				entry.Content = append(entry.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "stake_root"},
					&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: hexRoot},
				)
			}
		} else {
			// Append new entry
			entryNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Tag:   "!!map",
				Style: 0,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "chain_id", Style: 0},
					{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.FormatUint(chainID, 10), Style: 0},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "stake_root", Style: 0},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: hexRoot, Style: 0},
				},
			}
			activeRootsNode.Content = append(activeRootsNode.Content, entryNode)
		}
	}

	// Write the context back to disk
	err = common.WriteYAML(yamlPath, rootNode)
	if err != nil {
		return fmt.Errorf("failed to write updated context to disk: %w", err)
	}

	return nil
}

// Get all stake table roots from appropriate OperatorTableUpdaters
func GetOnchainStakeTableRoots(cCtx *cli.Context) (map[uint64][32]byte, error) {
	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)

	// Discover and collate all roots
	roots := make(map[uint64][32]byte)

	// Extract context
	cfg, err := common.LoadConfigWithContextConfig(devnet.DEVNET_CONTEXT)
	if err != nil {
		return nil, fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
	}
	envCtx, ok := cfg.Context[devnet.DEVNET_CONTEXT]
	if !ok {
		return nil, fmt.Errorf("context '%s' not found in configuration", devnet.DEVNET_CONTEXT)
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

	// Get a new chainManager
	cm := chainManager.NewChainManager()

	// Configure L1 chain
	holeskyConfig := &chainManager.ChainConfig{
		ChainID: uint64(chainId),
		RPCUrl:  rpcUrl,
	}
	if err := cm.AddChain(holeskyConfig); err != nil {
		return nil, fmt.Errorf("Failed to add chain: %v", err)
	}
	holeskyClient, err := cm.GetChainForId(holeskyConfig.ChainID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get chain for ID %d: %v", holeskyConfig.ChainID, err)
	}

	// Construct registry caller
	ccRegistryCaller, err := ICrossChainRegistry.NewICrossChainRegistryCaller(crossChainRegistryAddress, holeskyClient.RPCClient)
	if err != nil {
		return nil, fmt.Errorf("Failed to get CrossChainRegistryCaller for %s: %v", crossChainRegistryAddress, err)
	}

	// Get chains from contract
	chainIds, addresses, err := ccRegistryCaller.GetSupportedChains(&bind.CallOpts{})
	if err != nil {
		return nil, fmt.Errorf("failed to get supported chains: %w", err)
	}
	if len(chainIds) == 0 {
		return nil, fmt.Errorf("no supported chains found in cross-chain registry")
	}

	// Iterate and collect all roots for all chainIds
	for i, chainId := range chainIds {
		// Ignore 17000 from chainIds
		if chainId.Uint64() == 17000 {
			continue
		}

		// Use provided OperatorTableUpdaterTransactor address
		addr := addresses[i]
		chain, err := cm.GetChainForId(chainId.Uint64())
		if err != nil {
			return nil, fmt.Errorf("failed to get chain for ID %d: %w", chainId, err)
		}

		// Get the OperatorTableUpdaterTransactor at the provided chains address
		transactor, err := IOperatorTableUpdater.NewIOperatorTableUpdater(addr, chain.RPCClient)
		if err != nil {
			return nil, fmt.Errorf("failed to bind NewIOperatorTableUpdaterTransactor: %w", err)
		}

		// Collect the current root from provided chainId
		root, err := transactor.GetCurrentGlobalTableRoot(&bind.CallOpts{})
		if err != nil {
			return nil, fmt.Errorf("failed to get stake root: %w", err)
		}

		// Collect the provided root
		roots[chainId.Uint64()] = root
	}

	return roots, nil
}

// Verify the context stored ActiveStakeRoots match onchain state
func VerifyActiveStakeTableRoots(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx.Context)

	// Read expected roots from context
	_, _, contextNode, err := common.LoadContext("devnet") // @TODO: make dynamic
	if err != nil {
		return fmt.Errorf("failed to load context YAML: %w", err)
	}

	transporterNode := common.GetChildByKey(contextNode, "transporter")
	if transporterNode == nil {
		return fmt.Errorf("missing 'transporter' section in context")
	}

	activeRootsNode := common.GetChildByKey(transporterNode, "active_stake_roots")
	if activeRootsNode == nil || activeRootsNode.Kind != yaml.SequenceNode {
		return fmt.Errorf("'active_stake_roots' is missing or not a list")
	}

	expectedMap := make(map[uint64][32]byte)
	for _, entry := range activeRootsNode.Content {
		if entry.Kind != yaml.MappingNode {
			return fmt.Errorf("malformed entry in 'active_stake_roots'; expected map")
		}

		var chainID uint64
		var rootBytes [32]byte
		var foundCID, foundRoot bool

		for i := 0; i < len(entry.Content); i += 2 {
			key := entry.Content[i].Value
			val := entry.Content[i+1].Value

			switch key {
			case "chain_id":
				cid, err := strconv.ParseUint(val, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid chain_id: %w", err)
				}
				chainID = cid
				foundCID = true
			case "stake_root":
				b, err := hexutil.Decode(val)
				if err != nil {
					return fmt.Errorf("invalid stake_root hex: %w", err)
				}
				if len(b) != 32 {
					return fmt.Errorf("stake_root must be 32 bytes, got %d", len(b))
				}
				copy(rootBytes[:], b)
				foundRoot = true
			}
		}

		if !foundCID || !foundRoot {
			return fmt.Errorf("entry missing required fields 'chain_id' or 'stake_root'")
		}

		expectedMap[chainID] = rootBytes
	}

	// Fetch actual roots
	actualMap, err := GetOnchainStakeTableRoots(cCtx)
	if err != nil {
		return fmt.Errorf("failed to get onchain roots: %w", err)
	}

	// Compare expectations to actual (use actual as map source to allow user to move chainId if req)
	for id, actual := range actualMap {
		expected, ok := expectedMap[id]
		if !ok {
			return fmt.Errorf("missing onchain root for chainId %d", id)
		}
		if expected != actual {
			return fmt.Errorf("root mismatch for chainId %d:\nexpected: %x\ngot:      %x", id, expected, actual)
		}
	}

	logger.Info("Root matches onchain state.")
	return nil
}

// Schedule transport using the default parser and transportFunc
func ScheduleTransport(cCtx *cli.Context, cronExpr string) error {
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	// Run the scheduler with transport func
	return ScheduleTransportWithParserAndFunc(cCtx, cronExpr, parser, func() {
		if err := Transport(cCtx); err != nil {
			log.Printf("Scheduled transport failed: %v", err)
		}
	})
}

// Schedule transport using custom parser and transportFunc
func ScheduleTransportWithParserAndFunc(cCtx *cli.Context, cronExpr string, parser cron.Parser, transportFunc func()) error {
	// Validate cron expression
	c := cron.New(cron.WithParser(parser))
	_, err := parser.Parse(cronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Call Transport() against cronExpr
	_, err = c.AddFunc(cronExpr, transportFunc)
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
