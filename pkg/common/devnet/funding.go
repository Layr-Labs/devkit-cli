package devnet

import (
	devkitcommon "devkit-cli/pkg/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"math/big"
	"os"
	"os/exec"
	"time"
)

// FundWallets sends ETH to a list of addresses using `cast send`
// Only funds wallets with balance < 10 ether.
func FundWalletsDevnet(cfg *devkitcommon.EigenConfig, rpcURL string) {
	var client *ethclient.Client
	var err error

	// Retry with exponential backoff up to 5 times
	for retries := 0; retries < 5; retries++ {
		client, err = ethclient.Dial(rpcURL)
		if err == nil {
			break
		}
		log.Printf("⚠️  Waiting for devnet to be ready (%d/5)...", retries+1)
		time.Sleep(time.Duration(1<<retries) * time.Second) // 1s, 2s, 4s, 8s, 16s
	}

	if err != nil {
		log.Printf("❌ Could not connect to devnet RPC after retries: %v", err)
		return
	}
	defer client.Close()

	// All operator keys from [operator]
	for _, key := range cfg.Operator.Keys {
		privateKey, _ := crypto.HexToECDSA(key)
		fundIfNeeded(client, crypto.PubkeyToAddress(privateKey.PublicKey), key, rpcURL)
	}

	// All submit wallets from operator sets
	for _, set := range cfg.OperatorSets {
		privateKey, _ := crypto.HexToECDSA(set.SubmitWallet)
		addr := crypto.PubkeyToAddress(privateKey.PublicKey)
		fundIfNeeded(client, addr, cfg.Operator.Keys[0], rpcURL) // fund from index 0 key
	}
}

func fundIfNeeded(client *ethclient.Client, to common.Address, fromKey string, rpcURL string) {
	balanceCmd := exec.Command("cast", "balance",
		to.String(),
		"--rpc-url", rpcURL,
	)
	balanceOutput, err := balanceCmd.Output()
	if err != nil {
		log.Printf("❌ Failed to get balance for %s: %v", to.String(), err)
		return
	}
	threshold := new(big.Int)
	threshold.SetString(FUND_VALUE, 10)
	balance := new(big.Int)
	balance.SetString(string(balanceOutput), 10)
	if balance.Cmp(threshold) >= 0 {
		log.Printf("✅ %s already has sufficient balance (%s wei)", to, balance.String())
		return
	}

	log.Printf("Funding %s with %s from %s", to, FUND_VALUE, fromKey)
	cmd := exec.Command("cast", "send",
		to.String(),
		"--value", FUND_VALUE,
		"--rpc-url", rpcURL,
		"--private-key", fromKey,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("❌ Failed to fund %s: %v", to, err)
	} else {
		log.Printf("✅ Funded %s", to)
	}
}
