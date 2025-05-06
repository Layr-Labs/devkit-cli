package devnet

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:nightly-1ae64e38a1c69bda45343947875f7c86bad00038"
const CHAIN_ARGS = "--block-time 3 --base-fee 0 --gas-price 0"
const FUND_VALUE = "10000000000000000000"
const RPC_URL = "http://localhost:8545"
const DEVNET_ENV_KEY = "devnet"
const CONTRACTS_REGISTRY="0x5FbDB2315678afecb367f032d93F642f64180aa3"
const CONTRACTS_REGISTRY_ABI = `[{"inputs":[{"internalType":"string","name":"","type":"string"}],"name":"nameToAddress","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"}]`
