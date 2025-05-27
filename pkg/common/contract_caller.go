package common

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
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
	logger            iface.Logger
}

func NewContractCaller(privateKeyHex string, chainID *big.Int, client *ethclient.Client, allocationManagerAddr, delegationManagerAddr common.Address, logger iface.Logger) (*ContractCaller, error) {
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
		logger:            logger,
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
