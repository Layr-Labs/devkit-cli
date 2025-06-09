package devnet

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"

	devkitcommon "github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/contracts"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"

	"context"

	"github.com/ethereum/go-ethereum/accounts/abi"
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

// EIGEN contract ABI for unwrap function
const eigenUnwrapABI = `[{"constant":false,"inputs":[{"name":"amount","type":"uint256"}],"name":"unwrap","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`

// EIGEN contract address
const eigenContractAddress = "0xec53bF9167f50cDEB3Ae105f56099aaaB9061F83"

// Common mainnet token holders with large balances - mapped by token address
var DefaultTokenHolders = map[common.Address]TokenFunding{
	common.HexToAddress("0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"): { // stETH token address
		TokenName:     "stETH",
		HolderAddress: common.HexToAddress("0x176F3DAb24a159341c0509bB36B833E7fdd0a132"), // Large stETH holder
		Amount:        new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)),              // 1000 tokens
	},
	common.HexToAddress("0x83E9115d334D248Ce39a6f36144aEaB5b3456e75"): { // bEIGEN token address
		TokenName:     "bEIGEN",
		HolderAddress: common.HexToAddress("0x564a1Bd9cFe0969d2A3880fcF9e228E9E1b29856"), // Large EIGEN holder that calls unwrap() to get bEIGEN
		Amount:        new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e18)),              // 1000 tokens
	},
}

// ImpersonateAccount enables impersonation of an account on Anvil
func ImpersonateAccount(client *rpc.Client, address common.Address) error {
	var result interface{}
	err := client.Call(&result, "anvil_impersonateAccount", address.Hex())
	if err != nil {
		return fmt.Errorf("failed to impersonate account %s: %w", address.Hex(), err)
	}
	return nil
}

// StopImpersonatingAccount disables impersonation of an account on Anvil
func StopImpersonatingAccount(client *rpc.Client, address common.Address) error {
	var result interface{}
	err := client.Call(&result, "anvil_stopImpersonatingAccount", address.Hex())
	if err != nil {
		return fmt.Errorf("failed to stop impersonating account %s: %w", address.Hex(), err)
	}
	return nil
}

// FundStakerWithTokens funds staker with strategy tokens using impersonation
func FundStakerWithTokens(ctx context.Context, ethClient *ethclient.Client, rpcClient *rpc.Client, stakerAddress common.Address, tokenFunding TokenFunding, tokenAddress common.Address, rpcURL string) error {
	if tokenFunding.TokenName == "bEIGEN" {
		// For bEIGEN, we need to call unwrap() on the EIGEN contract first
		// to convert EIGEN tokens to bEIGEN tokens

		// Parse EIGEN unwrap ABI
		eigenABI, err := abi.JSON(strings.NewReader(eigenUnwrapABI))
		if err != nil {
			return fmt.Errorf("failed to parse EIGEN unwrap ABI: %w", err)
		}

		// Start impersonating the token holder for unwrap call
		if err := ImpersonateAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
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
		if balance.Cmp(big.NewInt(100000000000000000)) < 0 {
			err = fundIfNeeded(tokenFunding.HolderAddress, ANVIL_1_KEY, rpcURL)
			if err != nil {
				return fmt.Errorf("failed to fund holder address: %w", err)
			}
		}

		// Send unwrap transaction from impersonated account using RPC for impersonated accounts
		var unwrapTxHash common.Hash
		err = rpcClient.Call(&unwrapTxHash, "eth_sendTransaction", map[string]interface{}{
			"from":     tokenFunding.HolderAddress.Hex(),
			"to":       eigenContractAddress,
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
		log.Printf("EIGEN to bEIGEN unwrap transaction eceipt: %v", unwrapReceipt.TxHash)

		if unwrapReceipt.Status == 0 {
			return fmt.Errorf("EIGEN to bEIGEN unwrap transaction reverted")
		}

		// Stop impersonating for unwrap (we'll impersonate again for transfer)
		if err := StopImpersonatingAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
			log.Printf("⚠️  Failed to stop impersonating after unwrap %s: %v", tokenFunding.HolderAddress.Hex(), err)
		}
	}

	// Start impersonating the token holder
	if err := ImpersonateAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
		return fmt.Errorf("failed to impersonate token holder: %w", err)
	}

	defer func() {
		if err := StopImpersonatingAccount(rpcClient, tokenFunding.HolderAddress); err != nil {
			log.Printf("⚠️  Failed to stop impersonating %s: %v", tokenFunding.HolderAddress.Hex(), err)
		}
	}()

	// Get gas price
	gasPrice, err := ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Encode transfer function call using the registry's ERC20 contract
	transferData, err := contracts.PackTransferCall(stakerAddress, tokenFunding.Amount)
	if err != nil {
		return fmt.Errorf("failed to pack transfer call: %w", err)
	}

	// Send token transfer transaction from impersonated account using RPC
	var txHash common.Hash
	err = rpcClient.Call(&txHash, "eth_sendTransaction", map[string]interface{}{
		"from":     tokenFunding.HolderAddress.Hex(),
		"to":       tokenAddress.Hex(),
		"gas":      "0x186a0", // 100000 in hex
		"gasPrice": fmt.Sprintf("0x%x", gasPrice),
		"value":    "0x0",
		"data":     fmt.Sprintf("0x%x", transferData),
	})
	if err != nil {
		return fmt.Errorf("failed to send token transfer transaction: %w", err)
	}

	// Wait for transaction receipt
	receipt, err := waitForTransaction(ctx, ethClient, txHash)
	if err != nil {
		return fmt.Errorf("token transfer transaction failed: %w", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("token transfer transaction reverted")
	}

	log.Printf("✅ Successfully funded %s with %s %s (tx: %s)",
		stakerAddress.Hex(),
		tokenFunding.Amount.String(),
		tokenAddress,
		txHash.Hex())

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

// FundWallets sends ETH to a list of addresses using `cast send`
// Only funds wallets with balance < 10 ether.
func FundWalletsDevnet(cfg *devkitcommon.ConfigWithContextConfig, rpcURL string) error {

	if os.Getenv("SKIP_DEVNET_FUNDING") == "true" {
		log.Println("🔧 Skipping devnet wallet funding (test mode)")
		return nil
	}

	// All operator keys from [operator]
	// We only intend to fund for devnet, so hardcoding to `CONTEXT` is fine
	for _, key := range cfg.Context[DEVNET_CONTEXT].Operators {
		cleanedKey := strings.TrimPrefix(key.ECDSAKey, "0x")
		privateKey, err := crypto.HexToECDSA(cleanedKey)
		if err != nil {
			log.Fatalf("invalid private key %q: %v", key.ECDSAKey, err)
		}
		err = fundIfNeeded(crypto.PubkeyToAddress(privateKey.PublicKey), key.ECDSAKey, rpcURL)
		if err != nil {
			return err
		}
	}
	return nil
}

func fundIfNeeded(to common.Address, fromKey string, rpcURL string) error {
	balanceCmd := exec.Command("cast", "balance", to.String(), "--rpc-url", rpcURL)
	balanceCmd.Env = append(os.Environ(), "FOUNDRY_DISABLE_NIGHTLY_WARNING=1")
	output, err := balanceCmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(output), "Error: error sending request for url") {
			log.Printf(" Please check if your mainnet fork rpc url is up")
		}
		return fmt.Errorf("failed to get balance for account%s", to.String())
	}
	threshold := new(big.Int)
	threshold.SetString(FUND_VALUE, 10)

	balanceStr := strings.TrimSpace(string(output))
	balance := new(big.Int)
	if _, ok := balance.SetString(balanceStr, 10); !ok {
		return fmt.Errorf("failed to parse balance from cast output: %s", balanceStr)
	}
	balance.SetString(string(output), 10)
	if balance.Cmp(threshold) >= 0 {
		log.Printf("✅ %s already has sufficient balance (%s wei)", to, balance.String())
		return nil
	}

	cmd := exec.Command("cast", "send",
		to.String(),
		"--value", FUND_VALUE,
		"--rpc-url", rpcURL,
		"--private-key", fromKey,
	)

	_, err = cmd.CombinedOutput()

	if err != nil {
		log.Printf("❌ Failed to fund %s: %v", to, err)
		return err
	} else {
		log.Printf("✅ Funded %s", to)
	}
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
		common.HexToAddress(eigenLayer.AllocationManager),
		common.HexToAddress(eigenLayer.DelegationManager),
		common.HexToAddress(eigenLayer.StrategyManager),
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
