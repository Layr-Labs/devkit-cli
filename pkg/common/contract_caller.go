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

	"github.com/Layr-Labs/devkit-cli/pkg/common/contracts"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ContractCaller provides a high-level interface for interacting with contracts
type ContractCaller struct {
	registry              *contracts.ContractRegistry
	ethclient             *ethclient.Client
	privateKey            *ecdsa.PrivateKey
	chainID               *big.Int
	logger                iface.Logger
	allocationManagerAddr common.Address
	delegationManagerAddr common.Address
	strategyManagerAddr   common.Address
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, client *ethclient.Client, allocationManagerAddr, delegationManagerAddr, strategyManagerAddr common.Address, logger iface.Logger) (*ContractCaller, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Build contract registry with core EigenLayer contracts
	registry := contracts.NewRegistryBuilder(client).
		AddEigenLayerCore(allocationManagerAddr, delegationManagerAddr, strategyManagerAddr).
		Build()

	return &ContractCaller{
		registry:              registry,
		ethclient:             client,
		privateKey:            privateKey,
		chainID:               chainID,
		logger:                logger,
		allocationManagerAddr: allocationManagerAddr,
		delegationManagerAddr: delegationManagerAddr,
		strategyManagerAddr:   strategyManagerAddr,
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
		cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        fmt.Sprintf("Strategy_%s", strategyAddress.Hex()[:8]),
			Type:        contracts.StrategyContract,
			Address:     strategyAddress,
			Description: fmt.Sprintf("Strategy contract at %s", strategyAddress.Hex()),
		})
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
		cc.registry.RegisterContract(contracts.ContractInfo{
			Name:        fmt.Sprintf("Token_%s", underlyingToken.Hex()[:8]),
			Type:        contracts.ERC20Contract,
			Address:     underlyingToken,
			Description: fmt.Sprintf("ERC20 token at %s", underlyingToken.Hex()),
		})
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

func (cc *ContractCaller) SetAllocationDelay(ctx context.Context, operatorAddress common.Address, delay uint32) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	allocationManager, err := cc.registry.GetAllocationManager(cc.allocationManagerAddr)
	if err != nil {
		return fmt.Errorf("failed to get AllocationManager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("SetAllocationDelay for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := allocationManager.SetAllocationDelay(opts, operatorAddress, delay)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for SetAllocationDelay: %s\n"+
					"operatorAddress: %s\n"+
					"delay: %d",
				tx.Hash().Hex(),
				operatorAddress,
				delay,
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

// GetRegistry returns the contract registry for external access
func (cc *ContractCaller) GetRegistry() *contracts.ContractRegistry {
	return cc.registry
}
