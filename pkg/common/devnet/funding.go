package devnet

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"strings"

	devkitcommon "github.com/Layr-Labs/devkit-cli/pkg/common"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// FundWallets sends ETH to a list of addresses using `cast send`
// Only funds wallets with balance < 10 ether.
func FundWalletsDevnet(cfg *devkitcommon.ConfigWithContextConfig, rpcURL string) error {

	if os.Getenv("SKIP_DEVNET_FUNDING") == "true" {
		log.Println("🔧 Skipping devnet wallet funding (test mode)")
		return nil
	}

	// All operator keys from [operator]
	// We only intend to fund for devnet, so hardcoding to `CONTEXT` is fine
	for _, key := range cfg.Context[CONTEXT].Operators {
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

	log.Printf("💸 Funding %s with %s from private key", to, FUND_VALUE)
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
