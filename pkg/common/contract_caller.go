package common

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/devkit-cli/pkg/common/contracts"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	crosschainregistry "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/CrossChainRegistry"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/DelegationManager"
	keyregistrar "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/KeyRegistrar"
	releasemanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ReleaseManager"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ContractCaller provides a high-level interface for interacting with contracts
type ContractCaller struct {
	registry               *contracts.ContractRegistry
	ethclient              *ethclient.Client
	privateKey             *ecdsa.PrivateKey
	chainID                *big.Int
	logger                 iface.Logger
	allocationManagerAddr  common.Address
	delegationManagerAddr  common.Address
	strategyManagerAddr    common.Address
	keyRegistrarAddr       common.Address
	crossChainRegistryAddr common.Address
	releaseManagerAddr     common.Address
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, client *ethclient.Client, allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, keyRegistrarAddr common.Address, crossChainRegistryAddr common.Address, releaseManagerAddr common.Address, logger iface.Logger) (*ContractCaller, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Build contract registry with core EigenLayer contracts
	builder := contracts.NewRegistryBuilder(client)
	builder, err = builder.AddEigenLayerCore(allocationManagerAddr, delegationManagerAddr, strategyManagerAddr, keyRegistrarAddr, crossChainRegistryAddr, releaseManagerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to add EigenLayer core contracts: %w", err)
	}

	registry := builder.Build()

	return &ContractCaller{
		registry:               registry,
		ethclient:              client,
		privateKey:             privateKey,
		chainID:                chainID,
		logger:                 logger,
		allocationManagerAddr:  allocationManagerAddr,
		delegationManagerAddr:  delegationManagerAddr,
		strategyManagerAddr:    strategyManagerAddr,
		keyRegistrarAddr:       keyRegistrarAddr,
		crossChainRegistryAddr: crossChainRegistryAddr,
		releaseManagerAddr:     releaseManagerAddr,
	}, nil
}

func (cc *ContractCaller) buildTxOpts() (*bind.TransactOpts, error) {
	opts, err := bind.NewKeyedTransactorWithChainID(cc.privateKey, cc.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}
	return opts, nil
}

func (cc *ContractCaller) SendAndWaitForTransaction(
	ctx context.Context,
	txDescription string,
	fn func() (*types.Transaction, error),
) error {

	tx, err := fn()
	if err != nil {
		cc.logger.Error("%s failed during execution: %v", txDescription, err)
		return fmt.Errorf("%s execution: %w", txDescription, err)
	}

	receipt, err := bind.WaitMined(ctx, cc.ethclient, tx)
	if err != nil {
		cc.logger.Error("Waiting for %s transaction (hash: %s) failed: %v", txDescription, tx.Hash().Hex(), err)
		return fmt.Errorf("waiting for %s transaction (hash: %s): %w", txDescription, tx.Hash().Hex(), err)
	}
	if receipt.Status == 0 {
		cc.logger.Error("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
		return fmt.Errorf("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
	}
	return nil
}

func (cc *ContractCaller) UpdateAVSMetadata(ctx context.Context, avsAddress common.Address, metadataURI string) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "UpdateAVSMetadataURI", func() (*types.Transaction, error) {
		tx, err := allocationManager.UpdateAVSMetadataURI(opts, avsAddress, metadataURI)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for UpdateAVSMetadata: %s\n"+
					"avsAddress: %s\n"+
					"metadataURI: %s",
				tx.Hash().Hex(),
				avsAddress,
				metadataURI,
			)
		}
		return tx, err
	})

	return err
}

// SetAVSRegistrar sets the registrar address for an AVS
func (cc *ContractCaller) SetAVSRegistrar(ctx context.Context, avsAddress, registrarAddress common.Address) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "SetAVSRegistrar", func() (*types.Transaction, error) {
		tx, err := allocationManager.SetAVSRegistrar(opts, avsAddress, registrarAddress)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for SetAVSRegistrar: %s\n"+
					"avsAddress: %s\n"+
					"registrarAddress: %s",
				tx.Hash().Hex(),
				avsAddress,
				registrarAddress,
			)
		}
		return tx, err
	})

	return err
}

func (cc *ContractCaller) CreateOperatorSets(ctx context.Context, avsAddress common.Address, createSetParams []allocationmanager.IAllocationManagerTypesCreateSetParams) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "CreateOperatorSets", func() (*types.Transaction, error) {
		tx, err := allocationManager.CreateOperatorSets(opts, avsAddress, createSetParams)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for CreateOperatorSets: %s\n"+
					"avsAddress: %s\n"+
					"createSetParams: %v",
				tx.Hash().Hex(),
				avsAddress,
				createSetParams,
			)
		}
		return tx, err
	})

	return err
}

func (cc *ContractCaller) RegisterAsOperator(ctx context.Context, operatorAddress common.Address, allocationDelay uint32, metadataURI string) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	delegationManager, err := cc.registry.GetDelegationManager(cc.delegationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get DelegationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterAsOperator for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := delegationManager.RegisterAsOperator(opts, operatorAddress, allocationDelay, metadataURI)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for RegisterAsOperator: %s\n"+
					"operatorAddress: %s\n"+
					"allocationDelay: %d\n"+
					"metadataURI: %s",
				tx.Hash().Hex(),
				operatorAddress,
				allocationDelay,
				metadataURI,
			)
		}
		return tx, err
	})

	return err
}

func (cc *ContractCaller) RegisterForOperatorSets(ctx context.Context, operatorAddress, avsAddress common.Address, operatorSetIDs []uint32, payload []byte) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	params := allocationmanager.IAllocationManagerTypesRegisterParams{
		Avs:            avsAddress,
		OperatorSetIds: operatorSetIDs,
		Data:           payload,
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterForOperatorSets for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := allocationManager.RegisterForOperatorSets(opts, operatorAddress, params)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for RegisterForOperatorSets: %s\n"+
					"  operatorAddress: %s\n"+
					"  avsAddress: %s\n"+
					"  operatorSetIDs: %v\n"+
					"  payload: %v\n",
				tx.Hash().Hex(),
				operatorAddress.Hex(),
				avsAddress.Hex(),
				operatorSetIDs,
				"0x"+hex.EncodeToString(payload),
			)
		}
		return tx, err
	})
	return err
}

func (cc *ContractCaller) DepositIntoStrategy(ctx context.Context, strategyAddress common.Address, amount *big.Int) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	// Get or register the strategy contract
	strategy, err := cc.registry.GetStrategy(strategyAddress)
	if err != nil {
		// Strategy not registered, add it to registry
		err = cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        fmt.Sprintf("Strategy_%s", strategyAddress.Hex()[:8]),
			Type:        contracts.StrategyContract,
			Address:     strategyAddress,
			Description: fmt.Sprintf("Strategy contract at %s", strategyAddress.Hex()),
		})
		if err != nil {
			return fmt.Errorf("failed to register strategy contract: %w", err)
		}
		strategy, err = cc.registry.GetStrategy(strategyAddress)
		if err != nil {
			return fmt.Errorf("failed to get strategy contract: %w", err)
		}
	}

	underlyingToken, err := strategy.UnderlyingToken(nil)
	if err != nil {
		return fmt.Errorf("failed to get underlying token: %w", err)
	}

	cc.logger.Info("Depositing into strategy %s with amount %s underlying token %s", strategyAddress.Hex(), amount.String(), underlyingToken.Hex())

	// Get or register the ERC20 token contract
	erc20Contract, err := cc.registry.GetERC20(underlyingToken)
	if err != nil {
		// ERC20 not registered, add it to registry
		err = cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        fmt.Sprintf("Token_%s", underlyingToken.Hex()[:8]),
			Type:        contracts.ERC20Contract,
			Address:     underlyingToken,
			Description: fmt.Sprintf("ERC20 token at %s", underlyingToken.Hex()),
		})
		if err != nil {
			return fmt.Errorf("failed to register ERC20 contract: %w", err)
		}
		erc20Contract, err = cc.registry.GetERC20(underlyingToken)
		if err != nil {
			return fmt.Errorf("failed to get ERC20 contract: %w", err)
		}
	}

	// approve the strategy manager to spend the underlying tokens
	cc.logger.Info("Approving strategy manager %s to spend %s of token %s", cc.strategyManagerAddr.Hex(), amount.String(), underlyingToken.Hex())
	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("Approve strategy manager: token %s, amount %s", underlyingToken.Hex(), amount.String()), func() (*types.Transaction, error) {
		opts, err := cc.buildTxOpts()
		if err != nil {
			return nil, fmt.Errorf("failed to build transaction options for approval: %w", err)
		}
		return erc20Contract.Transact(opts, "approve", cc.strategyManagerAddr, amount)
	})
	if err != nil {
		return fmt.Errorf("failed to approve strategy manager: %w", err)
	}

	strategyManager, err := cc.registry.GetStrategyManager(cc.strategyManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get StrategyManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("DepositIntoStrategy : strategy %s, amount %s", strategyAddress.Hex(), amount.String()), func() (*types.Transaction, error) {
		tx, err := strategyManager.DepositIntoStrategy(opts, strategyAddress, underlyingToken, amount)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for DepositIntoStrategy: %s\n"+
					"strategyAddress: %s\n"+
					"underlyingTokenAddress: %d\n"+
					"amount: %s",
				tx.Hash().Hex(),
				strategyAddress,
				underlyingToken,
				amount,
			)
		}
		return tx, err
	})
	return err
}

func (cc *ContractCaller) DelegateToOperator(ctx context.Context, operatorAddress common.Address, signature DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry, approverSalt [32]byte) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	delegationManager, err := cc.registry.GetDelegationManager(cc.delegationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get DelegationManager: %w", err)
	}

	cc.logger.Info("DelegateToOperator parameters - Operator: %s, Signature: %s, Expiry: %s, ApproverSalt: %s",
		operatorAddress.Hex(),
		hex.EncodeToString(signature.Signature),
		signature.Expiry.String(),
		hex.EncodeToString(approverSalt[:]))

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("DelegateToOperator: operator %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := delegationManager.DelegateTo(opts, operatorAddress, signature, approverSalt)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for DelegateToOperator: %s\n"+
					"operatorAddress: %s\n"+
					"signature: %s\n"+
					"approverSalt: %s",
				tx.Hash().Hex(),
				operatorAddress,
				signature,
				approverSalt,
			)
		}
		return tx, err
	})
	return err
}

func (cc *ContractCaller) CreateApprovalSignature(ctx context.Context, stakerAddress common.Address, operatorAddress common.Address, approverAddress common.Address, approverPrivateKey string, approverSalt [32]byte, expiry *big.Int) (DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry, error) {
	delegationManager, err := cc.registry.GetDelegationManager(cc.delegationManagerAddr)
	if err != nil {
		return DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry{}, fmt.Errorf("failed to get DelegationManager: %w", err)
	}

	// calculateDelegationApprovalDigestHash
	delegationApprovalDigestHash, err := delegationManager.CalculateDelegationApprovalDigestHash(nil, stakerAddress, operatorAddress, approverAddress, approverSalt, expiry)
	if err != nil {
		return DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry{}, fmt.Errorf("failed to calculate delegation approval digest hash: %w", err)
	}

	// Convert private key from hex string to *ecdsa.PrivateKey
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(approverPrivateKey, "0x"))
	if err != nil {
		return DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry{}, fmt.Errorf("failed to parse private key: %w", err)
	}
	cc.logger.Info("Signing approval signature for staker %s, operator %s, approver %s, salt %s, expiry %s", stakerAddress.Hex(), operatorAddress.Hex(), approverAddress.Hex(), approverSalt, expiry.String())

	// sign the digest hash - convert [32]byte to []byte
	signature, err := crypto.Sign(delegationApprovalDigestHash[:], privateKey)
	if err != nil {
		return DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry{}, fmt.Errorf("failed to sign digest hash: %w", err)
	}

	// EigenLayer contracts use OpenZeppelin's SignatureChecker which expects recovery ID 27/28
	// crypto.Sign returns [R || S || V] where V is 0 or 1
	// OpenZeppelin's ECDSA library expects V to be 27 or 28
	if len(signature) == 65 {
		signature[64] += 27
		cc.logger.Debug("Adjusted signature for EigenLayer (V += 27): %s", hex.EncodeToString(signature))
	}

	// Create the signature with expiry structure
	signatureWithExpiry := DelegationManager.ISignatureUtilsMixinTypesSignatureWithExpiry{
		Signature: signature,
		Expiry:    expiry,
	}

	return signatureWithExpiry, nil
}

func (cc *ContractCaller) ModifyAllocations(ctx context.Context, operatorAddress common.Address, operatorPrivateKey string, strategies []common.Address, newMagnitudes []uint64, avsAddress common.Address, opSetId uint32, logger iface.Logger) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	operatorSet := allocationmanager.OperatorSet{Avs: avsAddress, Id: opSetId}
	allocations := []allocationmanager.IAllocationManagerTypesAllocateParams{
		{
			OperatorSet:   operatorSet,
			Strategies:    strategies,
			NewMagnitudes: newMagnitudes,
		},
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "ModifyAllocations", func() (*types.Transaction, error) {
		tx, err := allocationManager.ModifyAllocations(opts, operatorAddress, allocations)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for ModifyAllocations: %s\n"+
					"operatorAddress: %s\n"+
					"allocations: %s",
				tx.Hash().Hex(),
				operatorAddress,
				allocations,
			)
		}
		return tx, err
	})
	return err
}

func IsValidABI(v interface{}) error {
	b, err := json.Marshal(v) // serialize ABI field
	if err != nil {
		return fmt.Errorf("marshal ABI: %w", err)
	}
	_, err = abi.JSON(bytes.NewReader(b)) // parse it
	return err
}

// RegisterStrategiesFromConfig registers all strategy contracts found in the configuration
func (cc *ContractCaller) RegisterStrategiesFromConfig(cfg *OperatorSpec) error {
	for _, allocation := range cfg.Allocations {
		strategyAddress := common.HexToAddress(allocation.StrategyAddress)

		err := cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        allocation.Name,
			Type:        contracts.StrategyContract,
			Address:     strategyAddress,
			Description: fmt.Sprintf("Strategy contract for %s", allocation.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to register strategy %s (%s): %w", allocation.Name, allocation.StrategyAddress, err)
		}
	}
	return nil
}

// RegisterTokensFromStrategies registers all underlying token contracts from strategies
func (cc *ContractCaller) RegisterTokensFromStrategies(cfg *OperatorSpec) error {
	for _, allocation := range cfg.Allocations {
		strategyAddress := common.HexToAddress(allocation.StrategyAddress)

		// Get strategy contract
		strategy, err := cc.registry.GetStrategy(strategyAddress)
		if err != nil {
			return fmt.Errorf("failed to get strategy %s: %w", allocation.StrategyAddress, err)
		}

		// Get underlying token address
		underlyingTokenAddr, err := strategy.UnderlyingToken(nil)
		if err != nil {
			return fmt.Errorf("failed to get underlying token for strategy %s: %w", allocation.StrategyAddress, err)
		}

		// Register the token contract
		err = cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        fmt.Sprintf("Token_%s", allocation.Name),
			Type:        contracts.ERC20Contract,
			Address:     underlyingTokenAddr,
			Description: fmt.Sprintf("Underlying token for strategy %s", allocation.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to register token for strategy %s: %w", allocation.Name, err)
		}
	}
	return nil
}

func (cc *ContractCaller) ConfigureOpSetCurveType(ctx context.Context, avsAddress common.Address, opSetId uint32, curveType uint8) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	keyRegistrar, err := cc.registry.GetKeyRegistrar(cc.keyRegistrarAddr)
	if err != nil {
		return fmt.Errorf("failed to get KeyRegistrar: %w", err)
	}

	operatorSet := keyregistrar.OperatorSet{Avs: avsAddress, Id: opSetId}
	err = cc.SendAndWaitForTransaction(ctx, "ConfigureOpSetCurveType", func() (*types.Transaction, error) {
		tx, err := keyRegistrar.ConfigureOperatorSet(opts, operatorSet, curveType)
		return tx, err
	})
	return err
}

func (cc *ContractCaller) CreateGenerationReservation(ctx context.Context, opSetId uint32, operatorTableCalculator common.Address, avsAddress common.Address) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	crossChainRegistry, err := cc.registry.GetCrossChainRegistry(cc.crossChainRegistryAddr)
	if err != nil {
		return fmt.Errorf("failed to get CrossChainRegistry: %w", err)
	}
	cc.logger.Info("Creating generation reservation for operator set %d", opSetId)
	cc.logger.Info("Operator table calculator: %s", operatorTableCalculator.Hex())
	cc.logger.Info("AVS address: %s", avsAddress.Hex())

	operatorSet := crosschainregistry.OperatorSet{Avs: avsAddress, Id: opSetId}
	operatorSetConfig := crosschainregistry.ICrossChainRegistryTypesOperatorSetConfig{
		Owner:              avsAddress,
		MaxStalenessPeriod: 66666666,
	}

	// add 31337 to chainids
	chainIds := []*big.Int{big.NewInt(int64(cc.chainID.Int64()))}
	err = cc.SendAndWaitForTransaction(ctx, "CreateGenerationReservation", func() (*types.Transaction, error) {
		tx, err := crossChainRegistry.CreateGenerationReservation(opts, operatorSet, operatorTableCalculator, operatorSetConfig, chainIds)
		return tx, err
	})
	return err
}

func (cc *ContractCaller) WhitelistChainIdInCrossRegistry(ctx context.Context, operatorTableUpdater common.Address, chainId uint64) error {
	var (
		err      error
		nonce    uint64
		gasPrice *big.Int
		signedTx *types.Transaction
		receipt  *types.Receipt
	)

	chainIds := []*big.Int{big.NewInt(int64(chainId))}
	cc.logger.Info("Impersonating cross chain registry owner")
	ownerCrossChainRegistry := common.HexToAddress("0xDA29BB71669f46F2a779b4b62f03644A84eE3479")

	// Get RPC client from ethclient
	rpcClient := cc.ethclient.Client()

	// Fund the cross chain registry owner with 1 ETH if needed
	anvilKey := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(anvilKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse anvil private key: %w", err)
	}

	// Get the nonce for the sender
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	nonce, err = cc.ethclient.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err = cc.ethclient.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create the transaction
	tx := types.NewTransaction(
		nonce,
		ownerCrossChainRegistry,
		big.NewInt(1000000000000000000), // 1 ETH in wei
		21000,                           // Standard ETH transfer gas limit
		gasPrice,
		nil,
	)

	// Sign the transaction
	signedTx, err = types.SignTx(tx, types.NewEIP155Signer(cc.chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	if err = cc.ethclient.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err = bind.WaitMined(ctx, cc.ethclient, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed")
	}

	if err := ImpersonateAccount(rpcClient, ownerCrossChainRegistry); err != nil {
		return fmt.Errorf("failed to impersonate account: %w", err)
	}

	defer func() {
		if err := StopImpersonatingAccount(rpcClient, ownerCrossChainRegistry); err != nil {
			cc.logger.Error("failed to stop impersonating account: %w", err)
		}
	}()

	// Get gas price
	gasPrice, err = cc.ethclient.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Get the ABI from the metadata
	parsed, err := crosschainregistry.CrossChainRegistryMetaData.GetAbi()
	if err != nil {
		return fmt.Errorf("failed to get ABI: %w", err)
	}

	// Pack the function call data
	addChainIDsToWhitelistData, err := parsed.Pack("addChainIDsToWhitelist", chainIds, []common.Address{operatorTableUpdater})
	if err != nil {
		return fmt.Errorf("failed to pack addChainIDsToWhitelist call: %w", err)
	}

	// Send addChainIDsToWhitelist transaction from impersonated account using RPC
	var txHash common.Hash
	err = rpcClient.Call(&txHash, "eth_sendTransaction", map[string]interface{}{
		"from":     ownerCrossChainRegistry.Hex(),
		"to":       cc.crossChainRegistryAddr.Hex(),
		"gas":      "0x30d40", // 200000 in hex
		"gasPrice": fmt.Sprintf("0x%x", gasPrice),
		"value":    "0x0",
		"data":     fmt.Sprintf("0x%x", addChainIDsToWhitelistData),
	})
	if err != nil {
		cc.logger.Error("failed to send addChainIDsToWhitelist transaction: %w", err)
		return fmt.Errorf("failed to send addChainIDsToWhitelist transaction: %w", err)
	}

	// Force the tx to be mined
	err = rpcClient.Call(nil, "evm_mine")
	if err != nil {
		return fmt.Errorf("evm_mine call failed: %w", err)
	}

	// Wait for transaction receipt
	receipt, err = cc.ethclient.TransactionReceipt(ctx, txHash)
	if err != nil {
		cc.logger.Error("failed to get transaction receipt: %w", err)
		return fmt.Errorf("addChainIDsToWhitelist transaction failed: %w", err)
	}

	// Check for reverted tx and print receipt
	if receipt.Status == 0 {
		jsonBytes, err := json.MarshalIndent(receipt, "", "  ")
		if err != nil {
			cc.logger.Error("failed to marshal receipt: %v", err)
		} else {
			cc.logger.Error("addChainIDsToWhitelist transaction reverted: %s", string(jsonBytes))
		}
		return fmt.Errorf("addChainIDsToWhitelist transaction reverted")
	}

	cc.logger.Info("Successfully whitelisted chain ID %d in CrossChainRegistry (tx: %s)", chainId, txHash.Hex())

	return nil
}

func (cc *ContractCaller) RegisterKeyInKeyRegistrar(ctx context.Context, operatorAddress common.Address, avsAddress common.Address, opSetId uint32, keyData []byte, signature bn254.Signature) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	keyRegistrar, err := cc.registry.GetKeyRegistrar(cc.keyRegistrarAddr)
	if err != nil {
		return fmt.Errorf("failed to get KeyRegistrar: %w", err)
	}
	g1Point := &bn254.G1Point{
		G1Affine: signature.GetG1Point(),
	}
	g1Bytes, err := g1Point.ToPrecompileFormat()
	if err != nil {
		return fmt.Errorf("signature not in correct subgroup: %w", err)
	}
	operatorSet := keyregistrar.OperatorSet{Avs: avsAddress, Id: opSetId}
	err = cc.SendAndWaitForTransaction(ctx, "RegisterKeyInKeyRegistrar", func() (*types.Transaction, error) {
		tx, err := keyRegistrar.RegisterKey(opts, operatorAddress, operatorSet, keyData, g1Bytes)
		return tx, err
	})
	return err
}

// GetRegistry returns the contract registry for external access
func (cc *ContractCaller) GetRegistry() *contracts.ContractRegistry {
	return cc.registry
}

func (cc *ContractCaller) PublishRelease(ctx context.Context, avsAddress common.Address, artifacts []releasemanager.IReleaseManagerTypesArtifact, operatorSetId uint32, upgradeByTime int64) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}
	releaseManager, err := cc.registry.GetReleaseManager(cc.releaseManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get ReleaseManager: %w", err)
	}
	operatorSet := releasemanager.OperatorSet{Avs: avsAddress, Id: operatorSetId}
	release := releasemanager.IReleaseManagerTypesRelease{
		Artifacts:     artifacts,
		UpgradeByTime: uint32(upgradeByTime),
	}
	return cc.SendAndWaitForTransaction(ctx, "PublishRelease", func() (*types.Transaction, error) {
		tx, err := releaseManager.PublishRelease(opts, operatorSet, release)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for PublishRelease: %s\n"+
					"operatorSet: %s\n"+
					"release: %s",
				tx.Hash().Hex(),
				operatorSet,
				release,
			)
		}
		return tx, err
	})

}
