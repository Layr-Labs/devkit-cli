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

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	delegationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/DelegationManager"
	istrategy "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IStrategy"
	strategymanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/StrategyManager"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TODO: Should we break this out into it's own package?
type ContractCaller struct {
	allocationManager      *allocationmanager.AllocationManager
	delegationManager      *delegationmanager.DelegationManager
	strategyManager        *strategymanager.StrategyManager
	ethclient              *ethclient.Client
	privateKey             *ecdsa.PrivateKey
	chainID                *big.Int
	logger                 iface.Logger
	strategyManagerAddress common.Address
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, client *ethclient.Client, allocationManagerAddr, delegationManagerAddr, strategyManagerAddr common.Address, logger iface.Logger) (*ContractCaller, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	allocationManager, err := allocationmanager.NewAllocationManager(allocationManagerAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create AllocationManager: %w", err)
	}

	delegationManager, err := delegationmanager.NewDelegationManager(delegationManagerAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create DelegationManager: %w", err)
	}

	strategyManager, err := strategymanager.NewStrategyManager(strategyManagerAddr, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create StrategyManager: %w", err)
	}

	return &ContractCaller{
		allocationManager:      allocationManager,
		delegationManager:      delegationManager,
		strategyManager:        strategyManager,
		ethclient:              client,
		privateKey:             privateKey,
		chainID:                chainID,
		logger:                 logger,
		strategyManagerAddress: strategyManagerAddr,
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

	err = cc.SendAndWaitForTransaction(ctx, "UpdateAVSMetadataURI", func() (*types.Transaction, error) {
		tx, err := cc.allocationManager.UpdateAVSMetadataURI(opts, avsAddress, metadataURI)
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

	err = cc.SendAndWaitForTransaction(ctx, "SetAVSRegistrar", func() (*types.Transaction, error) {
		tx, err := cc.allocationManager.SetAVSRegistrar(opts, avsAddress, registrarAddress)
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

// CreateOperatorSets creates operator sets for an AVS
func (cc *ContractCaller) CreateOperatorSets(ctx context.Context, avsAddress common.Address, sets []allocationmanager.IAllocationManagerTypesCreateSetParams) error {
	opts, err := cc.buildTxOpts()
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "CreateOperatorSets", func() (*types.Transaction, error) {
		tx, err := cc.allocationManager.CreateOperatorSets(opts, avsAddress, sets)
		if err == nil && tx != nil {
			cc.logger.Debug(
				"Transaction hash for CreateOperatorSets: %s\n"+
					"avsAddress: %s\n"+
					"IAllocationManagerTypesCreateSetParams[]: %s",
				tx.Hash().Hex(),
				avsAddress,
				sets,
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

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterAsOperator for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := cc.delegationManager.RegisterAsOperator(opts, operatorAddress, allocationDelay, metadataURI)
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

	params := allocationmanager.IAllocationManagerTypesRegisterParams{
		Avs:            avsAddress,
		OperatorSetIds: operatorSetIDs,
		Data:           payload,
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterForOperatorSets for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		tx, err := cc.allocationManager.RegisterForOperatorSets(opts, operatorAddress, params)
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

	istrategyContract, err := istrategy.NewIStrategy(strategyAddress, cc.ethclient)
	if err != nil {
		return fmt.Errorf("failed to create IStrategy contract: %w", err)
	}
	underlyingToken, err := istrategyContract.UnderlyingToken(nil)
	if err != nil {
		return fmt.Errorf("failed to get underlying token: %w", err)
	}

	cc.logger.Info("Depositing into strategy %s with amount %s underlying token %s", strategyAddress.Hex(), amount.String(), underlyingToken.Hex())

	// Create manual ERC20 bindings for approve function
	erc20ABI := `[{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return fmt.Errorf("failed to parse ERC20 ABI: %w", err)
	}
	erc20Contract := bind.NewBoundContract(underlyingToken, parsedABI, cc.ethclient, cc.ethclient, cc.ethclient)

	// approve the strategy manager to spend the underlying tokens
	cc.logger.Info("Approving strategy manager %s to spend %s of token %s", cc.strategyManagerAddress.Hex(), amount.String(), underlyingToken.Hex())
	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("Approve strategy manager: token %s, amount %s", underlyingToken.Hex(), amount.String()), func() (*types.Transaction, error) {
		opts, err := cc.buildTxOpts()
		if err != nil {
			return nil, fmt.Errorf("failed to build transaction options for approval: %w", err)
		}
		return erc20Contract.Transact(opts, "approve", cc.strategyManagerAddress, amount)
	})
	if err != nil {
		return fmt.Errorf("failed to approve strategy manager: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("DepositIntoStrategy : strategy %s, amount %s", strategyAddress.Hex(), amount.String()), func() (*types.Transaction, error) {
		tx, err := cc.strategyManager.DepositIntoStrategy(opts, strategyAddress, underlyingToken, amount)
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

	if err != nil {
		return fmt.Errorf("failed to create AllocationManager contract: %w", err)
	}

	err = cc.SendAndWaitForTransaction(ctx, "ModifyAllocations", func() (*types.Transaction, error) {
		tx, err := cc.allocationManager.ModifyAllocations(opts, operatorAddress, allocations)
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
