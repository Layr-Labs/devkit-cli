package common

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	delegationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/DelegationManager"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TODO: Should we break this out into it's own package?
type ContractCaller struct {
	allocationManager *allocationmanager.AllocationManager
	delegationManager *delegationmanager.DelegationManager
	ethclient         *ethclient.Client
	privateKey        *ecdsa.PrivateKey
	chainID           *big.Int
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, client *ethclient.Client, allocationManagerAddr, delegationManagerAddr common.Address) (*ContractCaller, error) {
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

	return &ContractCaller{
		allocationManager: allocationManager,
		delegationManager: delegationManager,
		ethclient:         client,
		privateKey:        privateKey,
		chainID:           chainID,
	}, nil
}

func (cc *ContractCaller) buildTxOpts(ctx context.Context) (*bind.TransactOpts, error) {
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
	log, _ := GetLogger()

	tx, err := fn()
	if err != nil {
		log.Error("%s failed during execution: %v", txDescription, err)
		return fmt.Errorf("%s execution: %w", txDescription, err)
	}

	receipt, err := bind.WaitMined(ctx, cc.ethclient, tx)
	if err != nil {
		log.Error("Waiting for %s transaction (hash: %s) failed: %v", txDescription, tx.Hash().Hex(), err)
		return fmt.Errorf("waiting for %s transaction (hash: %s): %w", txDescription, tx.Hash().Hex(), err)
	}
	if receipt.Status == 0 {
		log.Error("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
		return fmt.Errorf("%s transaction (hash: %s) reverted", txDescription, tx.Hash().Hex())
	}
	return nil
}

func (cc *ContractCaller) UpdateAVSMetadata(ctx context.Context, avsAddress common.Address, metadataURI string) error {
	opts, err := cc.buildTxOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	return cc.SendAndWaitForTransaction(ctx, "UpdateAVSMetadataURI", func() (*types.Transaction, error) {
		return cc.allocationManager.UpdateAVSMetadataURI(opts, avsAddress, metadataURI)
	})
}

// SetAVSRegistrar sets the registrar address for an AVS
func (cc *ContractCaller) SetAVSRegistrar(ctx context.Context, avsAddress, registrarAddress common.Address) error {
	opts, err := cc.buildTxOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	return cc.SendAndWaitForTransaction(ctx, "SetAVSRegistrar", func() (*types.Transaction, error) {
		return cc.allocationManager.SetAVSRegistrar(opts, avsAddress, registrarAddress)
	})
}

// CreateOperatorSets creates operator sets for an AVS
func (cc *ContractCaller) CreateOperatorSets(ctx context.Context, avsAddress common.Address, sets []allocationmanager.IAllocationManagerTypesCreateSetParams) error {
	opts, err := cc.buildTxOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	return cc.SendAndWaitForTransaction(ctx, "CreateOperatorSets", func() (*types.Transaction, error) {
		return cc.allocationManager.CreateOperatorSets(opts, avsAddress, sets)
	})
}

func (cc *ContractCaller) RegisterAsOperator(ctx context.Context, operatorAddress common.Address, allocationDelay uint32, metadataURI string) error {
	opts, err := cc.buildTxOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	return cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterAsOperator for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		return cc.delegationManager.RegisterAsOperator(opts, operatorAddress, allocationDelay, metadataURI)
	})
}

func (cc *ContractCaller) RegisterForOperatorSets(ctx context.Context, operatorAddress, avsAddress common.Address, operatorSetIDs []uint32, payload []byte) error {
	opts, err := cc.buildTxOpts(ctx)
	if err != nil {
		return fmt.Errorf("failed to build transaction options: %w", err)
	}

	params := allocationmanager.IAllocationManagerTypesRegisterParams{
		Avs:            avsAddress,
		OperatorSetIds: operatorSetIDs,
		Data:           payload,
	}

	return cc.SendAndWaitForTransaction(ctx, fmt.Sprintf("RegisterForOperatorSets for %s", operatorAddress.Hex()), func() (*types.Transaction, error) {
		return cc.allocationManager.RegisterForOperatorSets(opts, operatorAddress, params)
	})
}
