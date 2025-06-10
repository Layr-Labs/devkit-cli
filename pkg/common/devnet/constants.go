package devnet

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:stable"
const CHAIN_ARGS = "--gas-limit 140000000 --base-fee 9400000"
const FUND_VALUE = "10000000000000000000"
const DEVNET_CONTEXT = "devnet"
const L1 = "l1"
const ANVIL_1_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

// Ref https://github.com/Layr-Labs/eigenlayer-contracts/blob/c08c9e849c27910f36f3ab746f3663a18838067f/src/contracts/core/AllocationManagerStorage.sol#L63
const ALLOCATION_DELAY_INFO_SLOT = 155

// These are fallback EigenLayer deployment addresses when not specified in context
const ALLOCATION_MANAGER_ADDRESS = "0x78469728304326CBc65f8f95FA756B0B73164462"
const DELEGATION_MANAGER_ADDRESS = "0xA44151489861Fe9e3055d95adC98FbD462B948e7"
const STRATEGY_MANAGER_ADDRESS = "0xdfB5f6CE42aAA7830E94ECFCcAd411beF4d4D5b6"
