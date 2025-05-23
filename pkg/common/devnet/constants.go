package devnet

import (
	"fmt"
	"os"
	"runtime"
)

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:stable"
const CHAIN_ARGS = "--chain-id 31337"
const FUND_VALUE = "10000000000000000000"
const CONTEXT = "devnet"
const L1 = "l1"

// @TODO: Add core eigenlayer deployment addresses to context
const ALLOCATION_MANAGER_ADDRESS = "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39"
const DELEGATION_MANAGER_ADDRESS = "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A"

// GetDefaultRPCURL returns the default RPC URL with platform-aware host
func GetDefaultRPCURL() string {
	// Use same logic as GetDockerHost but inline to avoid circular import
	host := "localhost"
	if dockersHost := os.Getenv("DOCKERS_HOST"); dockersHost != "" {
		host = dockersHost
	} else if runtime.GOOS != "linux" {
		host = "host.docker.internal"
	}
	return fmt.Sprintf("http://%s:8545", host)
}

// Legacy constant for backward compatibility
const RPC_URL = "http://localhost:8545"
