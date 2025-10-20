package devnet

// Foundry Image Date : 21 April 2025
const FOUNDRY_IMAGE = "ghcr.io/foundry-rs/foundry:stable"
const L1_CHAIN_ARGS = "--gas-limit 140000000 --base-fee 0 --gas-price 1000000 --no-rate-limit"
const L2_CHAIN_ARGS = "--gas-limit 140000000 --base-fee 0 --gas-price 1000000 --no-rate-limit"

const FUND_VALUE = "1000000000000000000"
const DEVNET_CONTEXT = "devnet"
const ANVIL_1_KEY = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
const ANVIL_2_KEY = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"

// Ref https://github.com/Layr-Labs/eigenlayer-contracts/blob/c08c9e849c27910f36f3ab746f3663a18838067f/src/contracts/core/AllocationManagerStorage.sol#L63
const ALLOCATION_DELAY_INFO_SLOT = 155

// Curve type constants for KeyRegistrar
const CURVE_TYPE_KEY_REGISTRAR_UNKNOWN = 0
const CURVE_TYPE_KEY_REGISTRAR_ECDSA = 1
const CURVE_TYPE_KEY_REGISTRAR_BN254 = 2

const EIGEN_CONTRACT_ADDRESS = "0x3B78576F7D6837500bA3De27A60c7f594934027E"

const ST_ETH_TOKEN_ADDRESS = "0x00c71b0fCadE911B2feeE9912DE4Fe19eB04ca56"
const B_EIGEN_TOKEN_ADDRESS = "0x275cCf9Be51f4a6C94aBa6114cdf2a4c45B9cb27"
const STRATEGY_TOKEN_FUNDING_AMOUNT_BY_LARGE_HOLDER_IN_ETH = 1000

const DEFAULT_L1_FORK_URL = "https://rpc.sepolia.ethpandaops.io"
const DEFAULT_L2_FORK_URL = "https://base-sepolia.gateway.tenderly.co"

const L1_CONTAINER_NAME_PREFIX = "devkit-devnet-l1-"
const L2_CONTAINER_NAME_PREFIX = "devkit-devnet-l2-"

const L1_CONTAINER_TYPE = "l1"
const L2_CONTAINER_TYPE = "l2"

const DEFAULT_L1_ANVIL_CHAINID = 31337
const DEFAULT_L2_ANVIL_CHAINID = 31338

const DEFAULT_L1_ANVIL_RPCURL = "http://localhost:8545"
const DEFAULT_L2_ANVIL_RPCURL = "http://localhost:9545"
