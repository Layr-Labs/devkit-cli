package devnet

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	devkitcommon "github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/contracts"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// TokenFunding represents a token transfer configuration
type TokenFunding struct {
	TokenName     string         `json:"token_name"`
	HolderAddress common.Address `json:"holder_address"`
	Amount        *big.Int       `json:"amount"`
}

// Common Sepolia token holders with large balances - mapped by token address
var DefaultTokenHolders = map[common.Address]TokenFunding{
	common.HexToAddress(ST_ETH_TOKEN_ADDRESS): { // stETH token address
		TokenName:     "stETH",
		HolderAddress: common.HexToAddress("0xC8088abD2FdaF4819230EB0FdA2D9766FDF9F409"),                                    // Large stETH holder
		Amount:        new(big.Int).Mul(big.NewInt(STRATEGY_TOKEN_FUNDING_AMOUNT_BY_LARGE_HOLDER_IN_ETH), big.NewInt(1e18)), // 1000 tokens
	},
	common.HexToAddress(B_EIGEN_TOKEN_ADDRESS): { // bEIGEN token address
		TokenName:     "bEIGEN",
		HolderAddress: common.HexToAddress("0x5f8C207382426D3f7F248E6321Cf93B34e66d6b9"),                                    // Large EIGEN holder that calls unwrap() to get bEIGEN
		Amount:        new(big.Int).Mul(big.NewInt(STRATEGY_TOKEN_FUNDING_AMOUNT_BY_LARGE_HOLDER_IN_ETH), big.NewInt(1e18)), // 1000 tokens
	},
}

// FundStakerWithTokens funds staker with strategy tokens using impersonation
func FundStakerWithTokens(ctx context.Context, ethClient *ethclient.Client, rpcClient *rpc.Client, stakerAddress common.Address, tokenFunding TokenFunding, tokenAddress common.Address, rpcURL string) error {
	if tokenFunding.TokenName == "bEIGEN" {
		// For bEIGEN, we need to call unwrap() on the EIGEN contract first
		// to convert EIGEN tokens to bEIGEN tokens

		// Parse EIGEN unwrap ABI
		eigenABI, err := abi.JSON(strings.NewReader(contracts.EIGEN_CONTRACT_ABI))
		if err != nil {
			return fmt.Errorf("failed to parse EIGEN unwrap ABI: %w", err)
		}

		// Start impersonating the token holder for unwrap call
		if err := devkitcommon.ImpersonateAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
			return fmt.Errorf("failed to impersonate token holder for unwrap: %w", err)
		}

		// Get gas price
		gasPrice, err := ethClient.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("failed to get gas price for unwrap: %w", err)
		}

		// Encode unwrap function call
		unwrapData, err := eigenABI.Pack("unwrap", tokenFunding.Amount)
		if err != nil {
			return fmt.Errorf("failed to pack unwrap call: %w", err)
		}
		// eth balance of holder address
		balance, err := ethClient.BalanceAt(ctx, tokenFunding.HolderAddress, nil)
		if err != nil {
			return fmt.Errorf("failed to get balance of holder address: %w", err)
		}

		// if holder balance < 0.1 ether, fund it
		fundValue, _ := strconv.ParseInt(FUND_VALUE, 10, 64)
		if balance.Cmp(big.NewInt(fundValue)) < 0 {
			err = fundIfNeeded(ethClient, tokenFunding.HolderAddress, ANVIL_2_KEY)
			if err != nil {
				return fmt.Errorf("failed to fund holder address: %w", err)
			}
		}

		// Send unwrap transaction from impersonated account using RPC for impersonated accounts
		var unwrapTxHash common.Hash
		err = rpcClient.Call(&unwrapTxHash, "eth_sendTransaction", map[string]interface{}{
			"from":     tokenFunding.HolderAddress.Hex(),
			"to":       EIGEN_CONTRACT_ADDRESS,
			"gas":      "0x30d40", // 200000 in hex
			"gasPrice": fmt.Sprintf("0x%x", gasPrice),
			"value":    "0x0",
			"data":     fmt.Sprintf("0x%x", unwrapData),
		})
		if err != nil {
			return fmt.Errorf("failed to send unwrap transaction: %w", err)
		}

		// Wait for unwrap transaction receipt
		unwrapReceipt, err := waitForTransaction(ctx, ethClient, unwrapTxHash)
		if err != nil {
			return fmt.Errorf("unwrap transaction failed: %w", err)
		}
		log.Printf("EIGEN to bEIGEN unwrap transaction receipt: %v", unwrapReceipt.TxHash)

		if unwrapReceipt.Status == 0 {
			return fmt.Errorf("EIGEN to bEIGEN unwrap transaction reverted")
		}

		// Stop impersonating for unwrap (we'll impersonate again for transfer)
		if err := devkitcommon.StopImpersonatingAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
			log.Printf("⚠️  Failed to stop impersonating after unwrap %s: %v", tokenFunding.HolderAddress.Hex(), err)
		}
	} else if tokenFunding.TokenName == "stETH" {
		// Get config
		anvil1Key := ANVIL_2_KEY
		anvil1Key = strings.TrimPrefix(anvil1Key, "0x")
		privateKey, err := crypto.HexToECDSA(anvil1Key)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		anvil1Address := crypto.PubkeyToAddress(privateKey.PublicKey)

		// Start impersonating the token holder
		if err := devkitcommon.ImpersonateAccount(rpcClient, anvil1Address); err != nil {
			return fmt.Errorf("failed to impersonate token holder: %w", err)
		}

		// stake eth to get stETH , call submit(address _referral) with referral as 0 address with ETH value to get stETh back
		// create call from abi
		stethABI, err := abi.JSON(strings.NewReader(contracts.ST_ETH_CONTRACT_ABI))
		if err != nil {
			return fmt.Errorf("failed to parse stETH contract ABI: %w", err)
		}

		submitData, err := stethABI.Pack("submit", common.Address{})
		if err != nil {
			return fmt.Errorf("failed to pack submit call: %w", err)
		}

		// Get gas price
		gasPrice, err := ethClient.SuggestGasPrice(ctx)
		if err != nil {
			return fmt.Errorf("failed to get gas price for unwrap: %w", err)
		}

		// Send submit transaction from impersonated account using RPC
		var submitTxHash common.Hash
		err = rpcClient.Call(&submitTxHash, "eth_sendTransaction", map[string]interface{}{
			"from":     anvil1Address.Hex(),
			"to":       ST_ETH_TOKEN_ADDRESS,
			"gas":      "0x30d40", // 200000 in hex
			"gasPrice": fmt.Sprintf("0x%x", gasPrice),
			"value":    fmt.Sprintf("0x%x", tokenFunding.Amount),
			"data":     fmt.Sprintf("0x%x", submitData),
		})
		if err != nil {
			return fmt.Errorf("failed to send submit transaction: %w", err)
		}

		// Wait for submit transaction receipt
		submitReceipt, err := waitForTransaction(ctx, ethClient, submitTxHash)
		if err != nil {
			return fmt.Errorf("submit transaction failed: %w", err)
		}

		log.Printf("stETH transaction receipt: %v", submitReceipt.TxHash)

		if submitReceipt.Status == 0 {
			return fmt.Errorf("stETH transaction reverted")
		}

		// transfer stETH to staker
		transferData, err := contracts.PackTransferCall(stakerAddress, tokenFunding.Amount)
		if err != nil {
			return fmt.Errorf("failed to pack transfer call: %w", err)
		}

		// Send transfer transaction from impersonated account using RPC
		var transferTxHash common.Hash
		err = rpcClient.Call(&transferTxHash, "eth_sendTransaction", map[string]interface{}{
			"from":     anvil1Address.Hex(),
			"to":       ST_ETH_TOKEN_ADDRESS,
			"gas":      "0x30d40", // 200000 in hex
			"gasPrice": fmt.Sprintf("0x%x", gasPrice),
			"value":    "0x0",
			"data":     fmt.Sprintf("0x%x", transferData),
		})

		if err != nil {
			return fmt.Errorf("failed to send transfer transaction: %w", err)
		}

		// Wait for transfer transaction receipt
		transferReceipt, err := waitForTransaction(ctx, ethClient, transferTxHash)
		if err != nil {
			return fmt.Errorf("transfer transaction failed: %w", err)
		}

		if transferReceipt.Status == 0 {
			return fmt.Errorf("stETH transfer transaction reverted")
		}

		// Stop impersonating for transfer
		if err := devkitcommon.StopImpersonatingAccount(rpcClient, anvil1Address); err != nil {
			log.Printf("⚠️  Failed to stop impersonating after transfer %s: %v", anvil1Address.Hex(), err)
		}
	}

	return nil
}

// FundStakersWithStrategyTokens funds all stakers with the specified strategy tokens
func FundStakersWithStrategyTokens(cfg *devkitcommon.ConfigWithContextConfig, rpcURL string, tokenAddresses []string) error {
	if os.Getenv("SKIP_TOKEN_FUNDING") == "true" {
		log.Println("🔧 Skipping token funding (test mode)")
		return nil
	}

	// Connect to RPC
	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer rpcClient.Close()

	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to connect to ETH client: %w", err)
	}
	defer ethClient.Close()

	ctx := context.Background()

	// Fund each staker with each requested token
	for _, staker := range cfg.Context[DEVNET_CONTEXT].Stakers {
		stakerAddr := common.HexToAddress(staker.StakerAddress)

		for _, tokenAddressStr := range tokenAddresses {
			tokenAddress := common.HexToAddress(tokenAddressStr)
			tokenFunding, exists := DefaultTokenHolders[tokenAddress]

			if !exists {
				log.Printf("Unknown token address: %s, skipping", tokenAddress.Hex())
				continue
			}

			err := FundStakerWithTokens(ctx, ethClient, rpcClient, stakerAddr, tokenFunding, tokenAddress, rpcURL)
			if err != nil {
				log.Printf("❌ Failed to fund %s with %s (%s): %v", stakerAddr.Hex(), tokenFunding.TokenName, tokenAddressStr, err)
				continue
			}
		}
	}

	return nil
}

// waitForTransaction waits for a transaction to be mined
func waitForTransaction(ctx context.Context, client *ethclient.Client, txHash common.Hash) (*types.Receipt, error) {
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}

		// If error is "not found", continue waiting
		if err.Error() == "not found" {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				// Small delay before retrying
				continue
			}
		}

		return nil, err
	}
}

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
	for _, operator := range cfg.Context[DEVNET_CONTEXT].Operators {
		var privateKey *ecdsa.PrivateKey

		// Check if ECDSA keystore is configured
		if len(operator.Keystores) > 0 && operator.Keystores[0].ECDSAKeystorePath != "" && operator.Keystores[0].ECDSAKeystorePassword != "" {
			// Load from keystore
			keystoreData, err := os.ReadFile(operator.Keystores[0].ECDSAKeystorePath)
			if err != nil {
				log.Fatalf("failed to read ECDSA keystore file %s: %v", operator.Keystores[0].ECDSAKeystorePath, err)
			}

			key, err := keystore.DecryptKey(keystoreData, operator.Keystores[0].ECDSAKeystorePassword)
			if err != nil {
				log.Fatalf("failed to decrypt ECDSA keystore: %v", err)
			}

			privateKey = key.PrivateKey
		} else if operator.ECDSAKey != "" {
			// Fall back to plaintext key
			cleanedKey := strings.TrimPrefix(operator.ECDSAKey, "0x")
			var err error
			privateKey, err = crypto.HexToECDSA(cleanedKey)
			if err != nil {
				log.Fatalf("invalid private key %q: %v", operator.ECDSAKey, err)
			}
		} else {
			log.Fatalf("no ECDSA key configuration found for operator %s", operator.Address)
		}
		err = fundIfNeeded(ethClient, crypto.PubkeyToAddress(privateKey.PublicKey), ANVIL_2_KEY)
		if err != nil {
			return err
		}
	}

	// Fund transporter
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.Context[DEVNET_CONTEXT].Transporter.PrivateKey, "0x"))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	err = fundIfNeeded(ethClient, crypto.PubkeyToAddress(privateKey.PublicKey), ANVIL_2_KEY)
	if err != nil {
		return err
	}

	return nil
}

func fundIfNeeded(ethClient *ethclient.Client, to common.Address, fromKey string) error {
	balance, err := ethClient.BalanceAt(context.Background(), to, nil)
	if err != nil {
		log.Printf(" Please check if your L1 and L2 fork rpc url is up")
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

// GetUnderlyingTokenAddressesFromStrategies extracts all unique underlying token addresses from strategy contracts
func GetUnderlyingTokenAddressesFromStrategies(cfg *devkitcommon.ConfigWithContextConfig, rpcURL string, logger iface.Logger) ([]string, error) {
	// Connect to ETH client
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ETH client: %w", err)
	}
	defer ethClient.Close()

	// Get EigenLayer contract addresses from config
	context := cfg.Context[DEVNET_CONTEXT]
	eigenLayer := context.EigenLayer
	if eigenLayer == nil {
		return nil, fmt.Errorf("EigenLayer configuration not found")
	}

	// Create a ContractCaller with proper registry
	contractCaller, err := devkitcommon.NewContractCaller(
		context.DeployerPrivateKey,
		big.NewInt(1), // Chain ID doesn't matter for read operations
		ethClient,
		common.HexToAddress(eigenLayer.L1.AllocationManager),
		common.HexToAddress(eigenLayer.L1.DelegationManager),
		common.HexToAddress(eigenLayer.L1.StrategyManager),
		common.HexToAddress(eigenLayer.L1.KeyRegistrar),
		common.HexToAddress(""),
		common.HexToAddress(""),
		common.HexToAddress(""),
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create contract caller: %w", err)
	}

	uniqueTokenAddresses := make(map[string]bool)
	var tokenAddresses []string

	// Register and process strategies for all operators
	for _, operator := range context.Operators {
		// Register strategies from this operator's allocations
		err := contractCaller.RegisterStrategiesFromConfig(&operator)
		if err != nil {
			log.Printf("⚠️  Failed to register strategies for operator %s: %v", operator.Address, err)
			continue
		}

		// Get underlying tokens for each allocation
		for _, allocation := range operator.Allocations {
			strategyAddress := common.HexToAddress(allocation.StrategyAddress)

			strategy, err := contractCaller.GetRegistry().GetStrategy(strategyAddress)
			if err != nil {
				log.Printf("⚠️  Failed to get strategy contract %s: %v", allocation.StrategyAddress, err)
				continue
			}

			// Call underlyingToken() on the strategy contract using the binding
			underlyingTokenAddr, err := strategy.UnderlyingToken(nil)
			if err != nil {
				log.Printf("⚠️  Failed to call underlyingToken() on strategy %s: %v", allocation.StrategyAddress, err)
				continue
			}

			// Add to unique set
			tokenAddrStr := underlyingTokenAddr.Hex()
			if !uniqueTokenAddresses[tokenAddrStr] {
				uniqueTokenAddresses[tokenAddrStr] = true
				tokenAddresses = append(tokenAddresses, tokenAddrStr)
				log.Printf("📋 Found underlying token %s for strategy %s (%s)", tokenAddrStr, allocation.Name, allocation.StrategyAddress)
			}
		}
	}

	return tokenAddresses, nil
}
