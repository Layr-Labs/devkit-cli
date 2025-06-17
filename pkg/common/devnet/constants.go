package devnet

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:stable"
const CHAIN_ARGS = "--gas-limit 140000000 --base-fee 9400000"
const FUND_VALUE = "1000000000000000000"
const CONTEXT = "devnet"
const L1 = "l1"
const ANVIL_1_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// These are fallback EigenLayer deployment addresses when not specified in context
const ALLOCATION_MANAGER_ADDRESS = "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39"
const DELEGATION_MANAGER_ADDRESS = "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A"
