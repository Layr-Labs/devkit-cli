package commands

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/devnet"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
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

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/robfig/cron/v3"
)

type G1Point struct{ X, Y *big.Int }
type BN254OperatorInfo struct {
	Pubkey  G1Point
	Weights []*big.Int
}

type Receipt struct {
	Status          hexutil.Uint64 `json:"status"`
	TransactionHash ethcommon.Hash `json:"transactionHash"`
}

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
			Action: func(cCtx *cli.Context) error {
				// Initial transport will take ownership and set a locally controller generator
				return Transport(cCtx, true)
			},
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

func Transport(cCtx *cli.Context, initialRun bool) error {
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
		return fmt.Errorf("transporter private key is empty. Please check config/contexts/devnet.yaml")
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
		return fmt.Errorf("transporter BLS private key is empty. Please check config/contexts/devnet.yaml")
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

	// On initial devnet Transport we take ownership of contracts and configure generator to use context keys
	if contextName == devnet.DEVNET_CONTEXT && initialRun {
		// Transfer ownership to our context configured PrivateKey
		transferOwnership(logger, l1Config.RPCURL, crossChainRegistryAddress, envCtx.Transporter.PrivateKey)

		// Construct registry caller
		ccRegistryCaller, err := ICrossChainRegistry.NewICrossChainRegistryCaller(crossChainRegistryAddress, l1Client.RPCClient)
		if err != nil {
			return fmt.Errorf("failed to get CrossChainRegistryCaller for %s: %v", crossChainRegistryAddress, err)
		}

		// Get chains from contract
		chainIds, addresses, err := ccRegistryCaller.GetSupportedChains(&bind.CallOpts{})
		if err != nil {
			return fmt.Errorf("failed to get supported chains: %w", err)
		}
		if len(chainIds) == 0 {
			return fmt.Errorf("no supported chains found in cross-chain registry")
		}

		// Iterate and collect all roots for all chainIds
		for i, chainId := range chainIds {
			// Ignore non devnet chainIds if checking devnet
			if contextName == devnet.DEVNET_CONTEXT && !(int(chainId.Uint64()) == l1ChainId || int(chainId.Uint64()) == l2ChainId) {
				continue
			}

			// Use provided OperatorTableUpdaterTransactor address
			tableUpdaterAddr := addresses[i]

			// Update owner on OperatorTableUpdaterTransactor address
			rpcURL := l1Config.RPCURL
			if chainId.Uint64() == uint64(l2ChainId) {
				rpcURL = l2Config.RPCURL
			}
			transferOwnership(logger, rpcURL, tableUpdaterAddr, envCtx.Transporter.PrivateKey)

			// Read the current generator (avs,id) from OperatorTableUpdater
			gen, err := getGenerator(cCtx.Context, logger, cm, chainId, tableUpdaterAddr)
			if err != nil {
				return fmt.Errorf("getGenerator chain %d at %s: %w", chainId.Uint64(), tableUpdaterAddr.Hex(), err)
			}

			// Move to a new unconfigered operatorSet
			if gen.Id == 1 {
				gen.Id = 2
			} else {
				gen.Id = 1
			}

			// Connect to an ethClient to construct contractCaller
			client, err := ethclient.Dial(l1RpcUrl)
			if err != nil {
				return fmt.Errorf("failed to connect to L1 RPC: %w", err)
			}

			// Construct contractCaller with KeyRegistrar
			contractCaller, err := common.NewContractCaller(
				envCtx.Transporter.PrivateKey,
				big.NewInt(int64(l1ChainId)),
				client,
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(envCtx.EigenLayer.L1.KeyRegistrar),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				ethcommon.HexToAddress(""),
				logger,
			)
			if err != nil {
				return fmt.Errorf("failed to create contract caller: %w", err)
			}

			// Derive BN254 keys from the hex string (no keystore files needed)
			blsHex := strings.TrimPrefix(envCtx.Transporter.BlsPrivateKey, "0x")

			// Extract key details
			scheme := bn254.NewScheme()
			skGeneric, err := scheme.NewPrivateKeyFromHexString(blsHex)
			if err != nil {
				return fmt.Errorf("parse BLS hex: %w", err)
			}
			blsPriv, err := bn254.NewPrivateKeyFromBytes(skGeneric.Bytes())
			if err != nil {
				return fmt.Errorf("convert BLS key: %w", err)
			}
			blsPub := blsPriv.Public()

			// Encode keyData for KeyRegistrar from the PUBLIC key
			keyData, err := contractCaller.EncodeBN254KeyData(blsPub)
			if err != nil {
				return fmt.Errorf("encode key data: %w", err)
			}

			// Configure the curve type
			if err := configureCurveTypeAsAVS(
				cCtx.Context,
				logger,
				l1RpcUrl, // KeyRegistrar is on L1
				ethcommon.HexToAddress(envCtx.EigenLayer.L1.KeyRegistrar),
				gen.Avs,
				uint32(gen.Id),
				common.CURVE_TYPE_KEY_REGISTRAR_BN254,
			); err != nil {
				return fmt.Errorf("configure curve type as AVS: %w", err)
			}

			// EOA/operator address you want to register for this OperatorSet
			opEOA := mustKey(logger, envCtx.Transporter.PrivateKey)
			operatorAddress := crypto.PubkeyToAddress(opEOA.PublicKey)

			// Build the message hash per registrar rules and sign with BLS private key
			msgHash, err := contractCaller.GetOperatorRegistrationMessageHash(
				cCtx.Context,
				operatorAddress,
				gen.Avs,
				uint32(gen.Id),
				keyData,
			)
			if err != nil {
				return fmt.Errorf("registration hash: %w", err)
			}

			// Sign the message hash with BLS key
			sig, err := blsPriv.SignSolidityCompatible(msgHash)
			if err != nil {
				return fmt.Errorf("BLS sign: %w", err)
			}
			bn254Signature := bn254.Signature(*sig)

			// Register in KeyRegistrar
			if err := contractCaller.RegisterKeyInKeyRegistrar(
				cCtx.Context,
				operatorAddress,
				gen.Avs,
				uint32(gen.Id),
				keyData,
				bn254Signature,
			); err != nil {
				return fmt.Errorf("register key in key registrar: %w", err)
			}

			// Get the certificateVerifier Addr on this chain
			certificateVerifierAddr := readBN254CertificateVerifier(cCtx.Context, logger, rpcURL, tableUpdaterAddr)

			// Update generator using the transporter BLS key
			if err := updateGeneratorFromContext(cCtx.Context, logger, cm, chainId, tableUpdaterAddr, certificateVerifierAddr, txSign, envCtx.Transporter.BlsPrivateKey, gen); err != nil {
				return fmt.Errorf("updateGenerator chain %d at %s: %w", chainId.Uint64(), tableUpdaterAddr.Hex(), err)
			}
		}
	}

	// Perform global table transport
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
		// Ignore non devnet chainIds if checking devnet
		if contextName == devnet.DEVNET_CONTEXT && !(int(chainId.Uint64()) == l1ChainId || int(chainId.Uint64()) == l2ChainId) {
			continue
		}

		// Use provided OperatorTableUpdaterTransactor address
		tableUpdaterAddr := addresses[i]
		chain, err := cm.GetChainForId(chainId.Uint64())
		if err != nil {
			return nil, fmt.Errorf("failed to get chain for ID %d: %w", chainId, err)
		}

		// Get the OperatorTableUpdaterTransactor at the provided chains address
		transactor, err := IOperatorTableUpdater.NewIOperatorTableUpdater(tableUpdaterAddr, chain.RPCClient)
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
		if err := Transport(cCtx, false); err != nil {
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

// Impersonate the current owner and call *.transferOwnership(newOwner).
func transferOwnership(logger iface.Logger, rpcURL string, proxy ethcommon.Address, privateKey string) {
	ctx := context.Background()
	c, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		logger.Error("failed to connect to rpc: %w", err)
	}

	// Transporter private key - used to derive the new owner address
	priv := mustKey(logger, privateKey)
	newOwner := crypto.PubkeyToAddress(priv.PublicKey)

	// ABI with owner() and transferOwnership(address)
	ownableABI := mustABI(logger, `[
	  {"inputs":[],"name":"owner","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"},
	  {"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"}
	]`)

	// Read current owner
	currOwner := readOwner(ctx, logger, c, ownableABI, proxy)
	logger.Info("Current owner: %s", currOwner.Hex())

	// Impersonate the current owner and fund it
	impersonate(ctx, logger, c, currOwner)
	defer stopImpersonate(ctx, c, currOwner)

	// Pack transferOwnership(newOwner)
	calldata, err := ownableABI.Pack("transferOwnership", newOwner)
	if err != nil {
		logger.Error("failed to pack callData %w", err)
	}

	// Send tx via eth_sendTransaction from the impersonated owner to the proxy
	tx := map[string]any{
		"from":  currOwner.Hex(),
		"to":    proxy.Hex(),
		"data":  hexutil.Encode(calldata),
		"value": "0x0",
	}
	var txHash ethcommon.Hash
	if err := c.CallContext(ctx, &txHash, "eth_sendTransaction", tx); err != nil {
		logger.Error("failed to send tx: %w", err)
	}

	// Await for tx receipt
	mustWaitReceipt(ctx, logger, c, txHash)
	logger.Info("TransferOwnership tx: %s", txHash.Hex())

	// Verify
	newOwnerRead := readOwner(ctx, logger, c, ownableABI, proxy)
	logger.Info("New owner: %s", newOwnerRead.Hex())
}

// Impersonate the AVS and call KeyRegistrar.configureOperatorSet(opSet, curveType)
func configureCurveTypeAsAVS(
	ctx context.Context,
	logger iface.Logger,
	rpcURL string,
	keyRegistrar ethcommon.Address,
	avs ethcommon.Address,
	opSetId uint32,
	curveType uint8,
) error {
	// Connect to provided RPC
	c, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		return fmt.Errorf("rpc dial: %w", err)
	}

	// Build minimal ABI
	krABI := mustABI(logger, `[
      {"inputs":[{"components":[{"internalType":"address","name":"avs","type":"address"},{"internalType":"uint32","name":"id","type":"uint32"}],"internalType":"struct OperatorSet","name":"opSet","type":"tuple"}],"name":"getOperatorSetCurveType","outputs":[{"internalType":"uint8","name":"","type":"uint8"}],"stateMutability":"view","type":"function"},
      {"inputs":[{"components":[{"internalType":"address","name":"avs","type":"address"},{"internalType":"uint32","name":"id","type":"uint32"}],"internalType":"struct OperatorSet","name":"opSet","type":"tuple"},{"internalType":"uint8","name":"curveType","type":"uint8"}],"name":"configureOperatorSet","outputs":[],"stateMutability":"nonpayable","type":"function"}
    ]`)

	// Tuple type to match (address avs, uint32 id)
	type opSetT struct {
		Avs ethcommon.Address
		Id  uint32
	}
	opSet := opSetT{Avs: avs, Id: opSetId}

	// Read current curve type; skip if already set
	calldataGet, _ := krABI.Pack("getOperatorSetCurveType", opSet)
	var out string
	if err := c.CallContext(ctx, &out, "eth_call",
		map[string]any{"to": keyRegistrar.Hex(), "data": hexutil.Encode(calldataGet)},
		"latest",
	); err != nil {
		return fmt.Errorf("getOperatorSetCurveType call: %w", err)
	}
	decoded, err := krABI.Unpack("getOperatorSetCurveType", ethcommon.FromHex(out))
	if err != nil {
		return fmt.Errorf("unpack getOperatorSetCurveType: %w", err)
	}
	if ct, ok := decoded[0].(uint8); ok && ct == curveType {
		logger.Info("Operator set %d already configured with curveType, skipping", opSetId)
		return nil
	}

	// Impersonate the current owner and fund it
	impersonate(ctx, logger, c, avs)
	defer stopImpersonate(ctx, c, avs)

	// Send configureOperatorSet from the AVS
	calldataCfg, err := krABI.Pack("configureOperatorSet", opSet, curveType)
	if err != nil {
		return fmt.Errorf("pack configureOperatorSet: %w", err)
	}

	// Construct tx to send from the AVS
	tx := map[string]any{
		"from":  avs.Hex(),
		"to":    keyRegistrar.Hex(),
		"data":  hexutil.Encode(calldataCfg),
		"value": "0x0",
	}
	var txHash ethcommon.Hash
	if err := c.CallContext(ctx, &txHash, "eth_sendTransaction", tx); err != nil {
		return fmt.Errorf("send configureOperatorSet: %w", err)
	}

	// Await receipt
	mustWaitReceipt(ctx, logger, c, txHash)
	logger.Info("ConfigureOperatorSet tx sent by AVS: %s", txHash.Hex())
	return nil
}

// Read BN254CertificateVerifier from TableUpdater
func readBN254CertificateVerifier(ctx context.Context, logger iface.Logger, rpcURL string, addr ethcommon.Address) ethcommon.Address {
	// Connect to provided RPC
	c, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		logger.Error("failed to connect to rpc: %w", err)
	}

	// Minimal ABI: bn254CertificateVerifier() -> address bn254CertificateVerifier
	certificateVerifierAbi := mustABI(logger, `[
		{"inputs":[],"name":"bn254CertificateVerifier","outputs":[{"type":"address"}],"stateMutability":"view","type":"function"}
	]`)

	data, _ := certificateVerifierAbi.Pack("bn254CertificateVerifier")
	call := map[string]any{"to": addr.Hex(), "data": hexutil.Encode(data)}
	var out string
	if err := c.CallContext(ctx, &out, "eth_call", call, "latest"); err != nil {
		logger.Error("failed to call contract: %w", err)
	}
	b := ethcommon.FromHex(out)
	return ethcommon.BytesToAddress(b[len(b)-20:])
}

// Call calculateOperatorInfoLeaf via a bound contract
func calcOperatorInfoLeaf(
	ctx context.Context,
	logger iface.Logger,
	backend bind.ContractCaller,
	addr ethcommon.Address,
	info BN254OperatorInfo,
) ([32]byte, error) {
	abiCalc := mustABI(logger, `[
		{"inputs":[{"components":[{"components":[{"internalType":"uint256","name":"X","type":"uint256"},{"internalType":"uint256","name":"Y","type":"uint256"}],"internalType":"struct BN254.G1Point","name":"pubkey","type":"tuple"},{"internalType":"uint256[]","name":"weights","type":"uint256[]"}],"internalType":"struct IOperatorTableCalculatorTypes.BN254OperatorInfo","name":"operatorInfo","type":"tuple"}],"name":"calculateOperatorInfoLeaf","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"pure","type":"function"}
	]`)

	c := bind.NewBoundContract(addr, abiCalc, backend, nil, nil)

	var outs []any
	if err := c.Call(&bind.CallOpts{Context: ctx}, &outs, "calculateOperatorInfoLeaf", info); err != nil {
		return [32]byte{}, fmt.Errorf("calcOperatorInfoLeaf call: %w", err)
	}
	if len(outs) != 1 {
		return [32]byte{}, fmt.Errorf("unexpected outputs len: %d", len(outs))
	}

	var leaf [32]byte
	switch v := outs[0].(type) {
	case [32]uint8:
		for i := 0; i < 32; i++ {
			leaf[i] = byte(v[i])
		}
	case []byte:
		if len(v) != 32 {
			return [32]byte{}, fmt.Errorf("bytes32 wrong length: %d", len(v))
		}
		copy(leaf[:], v)
	case ethcommon.Hash:
		leaf = v
	default:
		return [32]byte{}, fmt.Errorf("unexpected return type %T", v)
	}

	return leaf, nil
}

// Read OperatorTableUpdater.getGenerator() as a typed struct
func getGenerator(
	ctx context.Context,
	logger iface.Logger,
	cm chainManager.IChainManager,
	chainId *big.Int,
	tableUpdaterAddr ethcommon.Address,
) (IOperatorTableUpdater.OperatorSet, error) {
	chain, err := cm.GetChainForId(chainId.Uint64())
	if err != nil {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("get chain %d: %w", chainId.Uint64(), err)
	}

	// Minimal ABI: getGenerator() -> (address avs, uint32 id)
	abiGet := mustABI(logger, `[
		{"inputs":[],"name":"getGenerator","outputs":[{"components":[{"internalType":"address","name":"avs","type":"address"},{"internalType":"uint32","name":"id","type":"uint32"}],"internalType":"struct OperatorSet","name":"","type":"tuple"}],"stateMutability":"view","type":"function"}
	]`)

	// Pack calldata
	data, err := abiGet.Pack("getGenerator")
	if err != nil {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("pack getGenerator: %w", err)
	}

	// Do the eth_call
	raw, err := chain.RPCClient.CallContract(ctx, ethereum.CallMsg{
		To:   &tableUpdaterAddr,
		Data: data,
	}, nil)
	if err != nil {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("eth_call getGenerator: %w", err)
	}

	// Use Outputs.Unpack, then cast
	vals, err := abiGet.Methods["getGenerator"].Outputs.Unpack(raw)
	if err != nil {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("unpack getGenerator: %w", err)
	}
	if len(vals) != 1 {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("unexpected outputs len: %d", len(vals))
	}

	// The decoder created an anonymous struct with the right fields
	out, ok := vals[0].(struct {
		Avs ethcommon.Address `json:"avs"`
		Id  uint32            `json:"id"`
	})
	if !ok {
		return IOperatorTableUpdater.OperatorSet{}, fmt.Errorf("unexpected type %T", vals[0])
	}

	return IOperatorTableUpdater.OperatorSet{Avs: out.Avs, Id: out.Id}, nil
}

// Update the generator onchain to verify against context provided BLS key
func updateGeneratorFromContext(
	ctx context.Context,
	logger iface.Logger,
	cm chainManager.IChainManager,
	chainId *big.Int,
	tableUpdaterAddr ethcommon.Address,
	certificateVerifierAddr ethcommon.Address,
	txSign txSigner.ITransactionSigner,
	blsHex string,
	gen IOperatorTableUpdater.OperatorSet,
) error {
	chain, err := cm.GetChainForId(chainId.Uint64())
	if err != nil {
		return fmt.Errorf("get chain %d: %w", chainId.Uint64(), err)
	}

	updaterTx, err := IOperatorTableUpdater.NewIOperatorTableUpdater(tableUpdaterAddr, chain.RPCClient)
	if err != nil {
		return fmt.Errorf("bind updater tx: %w", err)
	}

	// Derive BLS pubkey
	scheme := bn254.NewScheme()
	skGeneric, err := scheme.NewPrivateKeyFromHexString(strings.TrimPrefix(blsHex, "0x"))
	if err != nil {
		return fmt.Errorf("parse bls: %w", err)
	}
	sk, err := bn254.NewPrivateKeyFromBytes(skGeneric.Bytes())
	if err != nil {
		return fmt.Errorf("bls convert: %w", err)
	}
	signer, err := blsSigner.NewInMemoryBLSSigner(sk)
	if err != nil {
		return fmt.Errorf("bls signer: %w", err)
	}
	pub, err := signer.GetPublicKey()
	if err != nil {
		return fmt.Errorf("pubkey: %w", err)
	}
	g1 := bn254.NewZeroG1Point().AddPublicKey(pub)
	g1b, err := g1.ToPrecompileFormat()
	if err != nil {
		return fmt.Errorf("g1 bytes: %w", err)
	}
	pkG1 := G1Point{
		X: new(big.Int).SetBytes(g1b[0:32]),
		Y: new(big.Int).SetBytes(g1b[32:64]),
	}

	// One-operator info
	info := BN254OperatorInfo{Pubkey: pkG1, Weights: []*big.Int{big.NewInt(1)}}

	// Calculate the root leaf
	root, err := calcOperatorInfoLeaf(ctx, logger, chain.RPCClient, certificateVerifierAddr, info)
	if err != nil {
		return fmt.Errorf("calc operatorInfo leaf: %w", err)
	}

	genInfo := IOperatorTableUpdater.IOperatorTableCalculatorTypesBN254OperatorSetInfo{
		OperatorInfoTreeRoot: root,
		NumOperators:         new(big.Int).SetUint64(1),
		AggregatePubkey:      IOperatorTableUpdater.BN254G1Point{X: pkG1.X, Y: pkG1.Y},
		TotalWeights:         []*big.Int{big.NewInt(1)},
	}

	auth, err := txSign.GetTransactOpts(ctx, chainId)
	if err != nil {
		return fmt.Errorf("opts: %w", err)
	}

	tx, err := updaterTx.UpdateGenerator(auth, gen, genInfo)
	if err != nil {
		return fmt.Errorf("updateGenerator tx: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, chain.RPCClient, tx)
	if err != nil {
		return fmt.Errorf("wait mined: %w", err)
	}
	if receipt.Status != 1 {
		return fmt.Errorf("updateGenerator reverted: %s", tx.Hash().Hex())
	}
	return nil
}

func readOwner(ctx context.Context, logger iface.Logger, c *rpc.Client, ab abi.ABI, proxy ethcommon.Address) ethcommon.Address {
	data, _ := ab.Pack("owner")
	call := map[string]any{"to": proxy.Hex(), "data": hexutil.Encode(data)}
	var out string
	if err := c.CallContext(ctx, &out, "eth_call", call, "latest"); err != nil {
		logger.Error("failed to call contract: %w", err)
	}
	b := ethcommon.FromHex(out)
	return ethcommon.BytesToAddress(b[len(b)-20:])
}

func impersonate(ctx context.Context, logger iface.Logger, c *rpc.Client, who ethcommon.Address) {
	var ok bool
	if err := c.CallContext(ctx, &ok, "anvil_impersonateAccount", who.Hex()); err != nil {
		logger.Error("failed to impersonate: %w", err)
	}
	// Fund so it can pay gas
	_ = c.CallContext(ctx, &ok, "anvil_setBalance", who.Hex(), "0x56BC75E2D63100000") // 100 ETH
}

func stopImpersonate(ctx context.Context, c *rpc.Client, who ethcommon.Address) {
	var ok bool
	_ = c.CallContext(ctx, &ok, "anvil_stopImpersonatingAccount", who.Hex())
}

func mustABI(logger iface.Logger, s string) abi.ABI {
	a, err := abi.JSON(strings.NewReader(s))
	if err != nil {
		logger.Error("invalid abi: %w", err)
	}
	return a
}

func mustKey(logger iface.Logger, hex string) *ecdsa.PrivateKey {
	if strings.HasPrefix(hex, "0x") || strings.HasPrefix(hex, "0X") {
		hex = hex[2:]
	}
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		logger.Error("invalid key: %w", err)
	}
	return key
}

func mustWaitReceipt(ctx context.Context, logger iface.Logger, c *rpc.Client, h ethcommon.Hash) {
	var receipt Receipt
	for {
		_ = c.CallContext(ctx, &receipt, "eth_getTransactionReceipt", h)
		if receipt.TransactionHash != (ethcommon.Hash{}) {
			break
		}
		time.Sleep(150 * time.Millisecond)
	}
	if receipt.Status != 1 {
		// Get reason
		var trace map[string]any
		_ = c.CallContext(ctx, &trace, "debug_traceTransaction", h.Hex(), map[string]any{"disableStack": true})
		logger.Error("tx reverted. trace: %+v", trace)
	}
}
