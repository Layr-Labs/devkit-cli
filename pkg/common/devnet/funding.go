package devnet

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	devkitcommon "github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// FundWallets sends ETH to a list of addresses
// Only funds wallets with balance < 0.3 ether.
func FundWalletsDevnet(cfg *devkitcommon.ConfigWithContextConfig, rpcURL string) error {
	if os.Getenv("SKIP_DEVNET_FUNDING") == "true" {
		log.Println("🔧 Skipping devnet wallet funding (test mode)")
		return nil
	}

	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to ETH client: %w", err)
	}
	defer ethClient.Close()

	// All operator keys from [operator]
	// We only intend to fund for devnet, so hardcoding to `CONTEXT` is fine
	for _, key := range cfg.Context[CONTEXT].Operators {
		pvtKey := strings.TrimPrefix(key.ECDSAKey, "0x")
		privateKey, err := crypto.HexToECDSA(pvtKey)
		if err != nil {
			log.Fatalf("invalid private key %q: %v", key.ECDSAKey, err)
		}
		err = fundIfNeeded(ethClient, crypto.PubkeyToAddress(privateKey.PublicKey), ANVIL_1_KEY)
		if err != nil {
			return err
		}
	}

	return nil
}

func fundIfNeeded(ethClient *ethclient.Client, to common.Address, fromKey string) error {
	balance, err := ethClient.BalanceAt(context.Background(), to, nil)
	if err != nil {
		log.Printf(" Please check if your mainnet fork rpc url is up")
		return fmt.Errorf("failed to get balance for account %s %v", to.String(), err)
	}
	threshold := new(big.Int)
	threshold.SetString("300000000000000000", 10) // 0.3 ETH in wei

	if balance.Cmp(threshold) >= 0 {
		log.Printf("✅ %s already has sufficient balance (%s wei)", to, balance.String())
		return nil
	}

	value, _ := new(big.Int).SetString(FUND_VALUE, 10) // 1 ETH in wei
	gasPrice, err := ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Get the nonce for the sender
	fromKey = strings.TrimPrefix(fromKey, "0x")
	privateKey, err := crypto.HexToECDSA(fromKey)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}
	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Check sender's balance
	senderBalance, err := ethClient.BalanceAt(context.Background(), fromAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get sender balance: %w", err)
	}

	// Calculate total cost (value + gas)
	gasLimit := uint64(21000)
	totalCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	totalCost.Add(totalCost, value)

	if senderBalance.Cmp(totalCost) < 0 {
		return fmt.Errorf("funder has insufficient balance: has %s wei, needs %s wei", senderBalance.String(), totalCost.String())
	}

	nonce, err := ethClient.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	tx := types.NewTransaction(
		nonce,
		to,
		value,
		gasLimit,
		gasPrice,
		nil, // data
	)

	// Get chain ID
	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Sign the transaction with the latest signer
	signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	err = ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Printf("Failed to send eth funding transaction: %v", err)
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	log.Printf("Transaction sent, waiting for confirmation...")

	// Wait for transaction to be mined using bind.WaitMined
	receipt, err := bind.WaitMined(context.Background(), ethClient, signedTx)
	if err != nil {
		return fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("transaction failed")
	}

	log.Printf("✅ Funded %s (tx: %s)", to, signedTx.Hash().Hex())
	return nil
}
