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
			Name:  "run",
			Usage: "Immediately transport stake root to L1",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
				},
			}, common.GlobalFlags...),
			Action: Transport,
		},
		{
			Name:  "verify",
			Usage: "Verify that the context active_stake_roots match onchain state",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
				},
			}, common.GlobalFlags...),
			Action: VerifyActiveStakeTableRoots,
		},
		{
			Name:  "schedule",
			Usage: "Schedule transport stake root to L1",
			Flags: append([]cli.Flag{
				&cli.StringFlag{
					Name:  "context",
					Usage: "Select the context to use in this command (devnet, testnet or mainnet)",
				},
				&cli.StringFlag{
					Name:  "cron-expr",
					Usage: "Specify a custom schedule to override config schedule",
					Value: "",
				},
			}, common.GlobalFlags...),
			Action: func(cCtx *cli.Context) error {
				// Extract vars
				contextName := cCtx.String("context")

				// Load config according to provided contextName
				var err error
				var cfg *common.ConfigWithContextConfig
				if contextName == "" {
					cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
				} else {
					cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
				}
				if err != nil {
					return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
				}

				// Extract context details
				envCtx, ok := cfg.Context[contextName]
				if !ok {
					return fmt.Errorf("context '%s' not found in configuration", contextName)
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
	logger := common.LoggerFromContext(cCtx)

	// Construct and collate all roots
	roots := make(map[uint64][32]byte)

	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return fmt.Errorf("failed to load configurations for whitelist chain id in cross registry: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	// Debug logging to check what's loaded
	logger.Info("Transporter config loaded - Private key present: %v, BLS key present: %v",
		envCtx.Transporter.PrivateKey != "",
		envCtx.Transporter.BlsPrivateKey != "")

	// Get the values from env/config
	crossChainRegistryAddress := ethcommon.HexToAddress(envCtx.EigenLayer.L1.CrossChainRegistry)

	// Unpack chain config from context
	l1Config, ok := envCtx.Chains[common.L1]
	if !ok {
		return fmt.Errorf("L1 chain config not found in context ('%s')", contextName)
	}
	l2Config, ok := envCtx.Chains[common.L2]
	if !ok {
		return fmt.Errorf("L2 chain config not found in context ('%s')", contextName)
	}

	// Unpack chain details from chain configs
	l1RpcUrl := l1Config.RPCURL
	l2RpcUrl := l2Config.RPCURL
	l1ChainId := l1Config.ChainID
	l2ChainId := l2Config.ChainID

	// Attempt to advance blocks
	if contextName == devnet.DEVNET_CONTEXT {
		err = devnet.AdvanceBlocks(cCtx, l1RpcUrl, 100)
		if err != nil {
			return fmt.Errorf("failed to advance blocks: %v", err)
		}
	} else {
		// Wait for one block to be mined
		time.Sleep(12 * time.Second)
	}

	cm := chainManager.NewChainManager()

	l1ChainManagerConfig := &chainManager.ChainConfig{
		ChainID: uint64(l1ChainId),
		RPCUrl:  l1RpcUrl,
	}
	l2ChainManagerConfig := &chainManager.ChainConfig{
		ChainID: uint64(l2ChainId),
		RPCUrl:  l2RpcUrl,
	}
	if err := cm.AddChain(l1ChainManagerConfig); err != nil {
		return fmt.Errorf("failed to add l1 chain: %v", err)
	}
	if err := cm.AddChain(l2ChainManagerConfig); err != nil {
		return fmt.Errorf("failed to add l2 chain: %v", err)
	}

	l1Client, err := cm.GetChainForId(l1ChainManagerConfig.ChainID)
	if err != nil {
		return fmt.Errorf("failed to get l1 chain for ID %d: %v", l1Config.ChainID, err)
	}

	// Check if private key is empty
	if envCtx.Transporter.PrivateKey == "" {
		return fmt.Errorf("Transporter private key is empty. Please check config/contexts/devnet.yaml")
	}

	txSign, err := txSigner.NewPrivateKeySigner(envCtx.Transporter.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create private key signer: %v", err)
	}

	tableCalc, err := operatorTableCalculator.NewStakeTableRootCalculator(&operatorTableCalculator.Config{
		CrossChainRegistryAddress: crossChainRegistryAddress,
	}, l1Client.RPCClient, rawLogger)
	if err != nil {
		return fmt.Errorf("failed to create StakeTableRootCalculator: %v", err)
	}

	// Sync chains so that timestamps match on both anvil instances (for devnet)
	if contextName == devnet.DEVNET_CONTEXT {
		logger.Info("Syncing chains...")
		err = devnet.SyncL1L2Timestamps(cCtx, l1RpcUrl, l2RpcUrl)
		if err != nil {
			return fmt.Errorf("failed to sync chains: %v", err)
		}
	}

	l1Block, err := l1Client.RPCClient.BlockByNumber(cCtx.Context, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return fmt.Errorf("failed to get block by number for l1: %v", err)
	}
	referenceTimestamp := uint32(l1Block.Time())
	logger.Info(" - Chains in sync (at ts: %d)", uint32(referenceTimestamp))

	root, tree, dist, err := tableCalc.CalculateStakeTableRoot(cCtx.Context, l1Block.NumberU64())
	if err != nil {
		return fmt.Errorf("failed to calculate stake table root: %v", err)
	}

	// Check if BLS private key is empty
	if envCtx.Transporter.BlsPrivateKey == "" {
		return fmt.Errorf("Transporter BLS private key is empty. Please check config/contexts/devnet.yaml")
	}

	scheme := bn254.NewScheme()
	genericPk, err := scheme.NewPrivateKeyFromHexString(envCtx.Transporter.BlsPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to create BLS private key: %v", err)
	}
	pk, err := bn254.NewPrivateKeyFromBytes(genericPk.Bytes())
	if err != nil {
		return fmt.Errorf("failed to convert BLS private key: %v", err)
	}

	inMemSigner, err := blsSigner.NewInMemoryBLSSigner(pk)
	if err != nil {
		return fmt.Errorf("failed to create in-memory BLS signer: %v", err)
	}

	stakeTransport, err := transport.NewTransport(
		&transport.TransportConfig{
			L1CrossChainRegistryAddress: crossChainRegistryAddress,
		},
		l1Client.RPCClient,
		inMemSigner,
		txSign,
		cm,
		rawLogger,
	)
	if err != nil {
		return fmt.Errorf("failed to create transport: %v", err)
	}

	// Provide chainIds to ignore for Devnets
	var ignoreChainIds = []*big.Int{}
	if contextName == devnet.DEVNET_CONTEXT {
		ignoreChainIds = []*big.Int{new(big.Int).SetUint64(11155111), new(big.Int).SetUint64(84532)}
	}

	// Transport globalTableRoot
	err = stakeTransport.SignAndTransportGlobalTableRoot(
		cCtx.Context,
		root,
		referenceTimestamp,
		l1Block.NumberU64(),
		ignoreChainIds,
	)
	if err != nil {
		return fmt.Errorf("failed to sign and transport global table root: %v", err)
	}

	// Collect the provided roots
	roots[l1ChainManagerConfig.ChainID] = root
	roots[l2ChainManagerConfig.ChainID] = root
	// Write the roots to context (each time we process one)
	err = WriteStakeTableRootsToContext(cCtx, roots)
	if err != nil {
		return fmt.Errorf("failed to write active_stake_roots: %w", err)
	}

	// Sleep before transporting AVSStakeTable
	logger.Info("Successfully signed and transported global table root, sleeping for 25 seconds")
	time.Sleep(25 * time.Second)

	// Fetch OperatorSets for AVSStakeTable transport
	opsets := dist.GetOperatorSets()
	if len(opsets) == 0 {
		return fmt.Errorf("no operator sets found, skipping AVS stake table transport")
	}

	for _, opset := range opsets {
		err = stakeTransport.SignAndTransportAvsStakeTable(
			cCtx.Context,
			referenceTimestamp,
			l1Block.NumberU64(),
			opset,
			root,
			tree,
			dist,
			ignoreChainIds,
		)
		if err != nil {
			return fmt.Errorf("failed to sign and transport AVS stake table for opset %v: %v", opset, err)
		}

		// log success
		logger.Info("Successfully signed and transported AVS stake table for opset %v", opset)
	}

	return nil
}

// Record StakeTableRoots in the context for later retrieval
func WriteStakeTableRootsToContext(cCtx *cli.Context, roots map[uint64][32]byte) error {
	// Get flag selected contextName
	contextName := cCtx.String("context")

	// Check for context
	var yamlPath string
	var rootNode, contextNode *yaml.Node
	var err error
	if contextName == "" {
		yamlPath, rootNode, contextNode, _, err = common.LoadDefaultContext()
	} else {
		yamlPath, rootNode, contextNode, _, err = common.LoadContext(contextName)
	}
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Navigate context to arrive at context.transporter.active_stake_roots
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
	logger := common.LoggerFromContext(cCtx)

	// Discover and collate all roots
	roots := make(map[uint64][32]byte)

	// Extract vars
	contextName := cCtx.String("context")

	// Load config according to provided contextName
	var err error
	var cfg *common.ConfigWithContextConfig
	if contextName == "" {
		cfg, contextName, err = common.LoadDefaultConfigWithContextConfig()
	} else {
		cfg, contextName, err = common.LoadConfigWithContextConfig(contextName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load configurations for getting onchain stake table roots: %w", err)
	}

	// Extract context details
	envCtx, ok := cfg.Context[contextName]
	if !ok {
		return nil, fmt.Errorf("context '%s' not found in configuration", contextName)
	}

	// Get the values from env/config
	crossChainRegistryAddress := ethcommon.HexToAddress(envCtx.EigenLayer.L1.CrossChainRegistry)

	// Unpack chain config from context
	l1Config, ok := envCtx.Chains[common.L1]
	if !ok {
		return nil, fmt.Errorf("L1 chain config not found in context ('%s')", contextName)
	}
	l2Config, ok := envCtx.Chains[common.L2]
	if !ok {
		return nil, fmt.Errorf("L2 chain config not found in context ('%s')", contextName)
	}

	// Unpack chain details from chain configs
	l1RpcUrl := l1Config.RPCURL
	l2RpcUrl := l2Config.RPCURL
	l1ChainId := l1Config.ChainID
	l2ChainId := l2Config.ChainID

	// Get a new chainManager
	cm := chainManager.NewChainManager()

	// Configure L1 chain
	l1ChainManagerConfig := &chainManager.ChainConfig{
		ChainID: uint64(l1ChainId),
		RPCUrl:  l1RpcUrl,
	}

	// Configure L2 chain
	l2ChainManagerConfig := &chainManager.ChainConfig{
		ChainID: uint64(l2ChainId),
		RPCUrl:  l2RpcUrl,
	}

	if err := cm.AddChain(l1ChainManagerConfig); err != nil {
		return nil, fmt.Errorf("failed to add l1 chain: %v", err)
	}
	if err := cm.AddChain(l2ChainManagerConfig); err != nil {
		return nil, fmt.Errorf("failed to add l2 chain: %v", err)
	}

	l1Client, err := cm.GetChainForId(l1ChainManagerConfig.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain for ID %d: %v", l1ChainManagerConfig.ChainID, err)
	}

	// Construct registry caller
	ccRegistryCaller, err := ICrossChainRegistry.NewICrossChainRegistryCaller(crossChainRegistryAddress, l1Client.RPCClient)
	if err != nil {
		return nil, fmt.Errorf("failed to get CrossChainRegistryCaller for %s: %v", crossChainRegistryAddress, err)
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
		// Ignore 11155111 and 84532 from chainIds if checking devnet
		if contextName == devnet.DEVNET_CONTEXT && (chainId.Uint64() == 11155111 || chainId.Uint64() == 84532) {
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

	// Print discovered roots
	logger.Info("Successfully collected StakeTableRoots...")
	for k, v := range roots {
		logger.Info(" - ChainId: %d, Root: %x", k, v)
	}

	return roots, nil
}

// Verify the context stored ActiveStakeRoots match onchain state
func VerifyActiveStakeTableRoots(cCtx *cli.Context) error {
	// Get logger
	logger := common.LoggerFromContext(cCtx)

	// Get flag selected contextName
	contextName := cCtx.String("context")

	// Check for context
	var contextNode *yaml.Node
	var err error
	if contextName == "" {
		_, _, contextNode, _, err = common.LoadDefaultContext()
	} else {
		_, _, contextNode, _, err = common.LoadContext(contextName)
	}
	if err != nil {
		return fmt.Errorf("context loading failed: %w", err)
	}

	// Navigate context to arrive at context.transporter.active_stake_roots
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
