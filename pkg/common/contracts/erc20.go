package contracts

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Standard ERC20 ABI
const ERC20ABI = `[
	{
		"constant": true,
		"inputs": [],
		"name": "name",
		"outputs": [{"name": "", "type": "string"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "symbol",
		"outputs": [{"name": "", "type": "string"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "decimals",
		"outputs": [{"name": "", "type": "uint8"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [],
		"name": "totalSupply",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_from", "type": "address"},
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transferFrom",
		"outputs": [{"name": "", "type": "bool"}],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_spender", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "approve",
		"outputs": [{"name": "", "type": "bool"}],
		"payable": false,
		"stateMutability": "nonpayable",
		"type": "function"
	},
	{
		"constant": true,
		"inputs": [
			{"name": "_owner", "type": "address"},
			{"name": "_spender", "type": "address"}
		],
		"name": "allowance",
		"outputs": [{"name": "", "type": "uint256"}],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "from", "type": "address"},
			{"indexed": true, "name": "to", "type": "address"},
			{"indexed": false, "name": "value", "type": "uint256"}
		],
		"name": "Transfer",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "owner", "type": "address"},
			{"indexed": true, "name": "spender", "type": "address"},
			{"indexed": false, "name": "value", "type": "uint256"}
		],
		"name": "Approval",
		"type": "event"
	}
]`

// NewERC20Contract creates a new ERC20 contract instance
func NewERC20Contract(address common.Address, client *ethclient.Client) (*bind.BoundContract, error) {
	parsedABI, err := abi.JSON(strings.NewReader(ERC20ABI))
	if err != nil {
		return nil, err
	}

	return bind.NewBoundContract(address, parsedABI, client, client, client), nil
}

// Common ERC20 token addresses on mainnet
var WellKnownTokens = map[string]common.Address{
	"WETH":   common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
	"USDC":   common.HexToAddress("0xA0b86a33E6417C1C8b6A82a8AA8877f1Df4b7F3f"),
	"USDT":   common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"),
	"DAI":    common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"),
	"stETH":  common.HexToAddress("0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"),
	"bEIGEN": common.HexToAddress("0x83E9115d334D248Ce39a6f36144aEaB5b3456e75"),
}

// GetTokenAddress returns the address for some known token symbols
func GetTokenAddress(symbol string) (common.Address, bool) {
	addr, exists := WellKnownTokens[symbol]
	return addr, exists
}

// GetERC20ABI returns the parsed ERC20 ABI
func GetERC20ABI() (abi.ABI, error) {
	return abi.JSON(strings.NewReader(ERC20ABI))
}

// PackTransferCall creates the call data for an ERC20 transfer
func PackTransferCall(to common.Address, amount *big.Int) ([]byte, error) {
	parsedABI, err := GetERC20ABI()
	if err != nil {
		return nil, err
	}
	return parsedABI.Pack("transfer", to, amount)
}
