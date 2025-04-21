package common

import (
	"log"
	"os/exec"
)

// FundWallets sends ETH to a list of addresses using `cast send`
// Requires `cast` to be installed and available in the system's PATH.
func FundWallets(value string, to []string, fromKey string, rpcURL string) {
	for _, addr := range to {
		cmd := exec.Command("cast", "send",
			addr,
			"--value", value,
			"--rpc-url", rpcURL,
			"--private-key", fromKey,
		)

		if err := cmd.Run(); err != nil {
			log.Printf("❌ Failed to fund %s: %v", addr, err)
		} else {
			log.Printf("✅ Funded %s", addr)
		}
	}
}
