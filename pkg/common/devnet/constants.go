package devnet

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:stable"
const L2_CHAIN_ARGS = "--chain-id 8453 --gas-limit 140000000 --base-fee 9400000"
const L1_CHAIN_ARGS = "--chain-id 31337"
const FUND_VALUE = "10000000000000000000"
const CONTEXT = "devnet"
const L1 = "l1"
const L2 = "l2"

// These are fallback EigenLayer deployment addresses when not specified in context
const ALLOCATION_MANAGER_ADDRESS = "0x948a420b8CC1d6BFd0B6087C2E7c344a2CD0bc39"
const DELEGATION_MANAGER_ADDRESS = "0x39053D51B77DC0d36036Fc1fCc8Cb819df8Ef37A"
