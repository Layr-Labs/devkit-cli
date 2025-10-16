package common

// Project structure constants
const (
	// L1 and L2 config names
	L1 = "l1"
	L2 = "l2"

	// ContractsDir is the subdirectory name for contract components
	ContractsDir = "contracts"

	// Makefile is the name of the makefile used for root level operations
	Makefile = "Makefile"

	// ContractsMakefile is the name of the makefile used for contract level operations
	ContractsMakefile = "Makefile"

	// GlobalConfigFile is the name of the global YAML used to store global config details (eg, user_id)
	GlobalConfigFile = "config.yaml"

	// Filename for devkit project config
	BaseConfig = "config.yaml"

	// Filename for zeus config
	ZeusConfig = ".zeus"

	// Docker open timeout
	DockerOpenTimeoutSeconds = 10

	// Docker open retry interval in milliseconds
	DockerOpenRetryIntervalMilliseconds = 500

	// CrossChainRegistryOwnerAddress is the address of the owner of the cross chain registry
	CrossChainRegistryOwnerAddress = "0xb094Ba769b4976Dc37fC689A76675f31bc4923b0"

	// Curve type constants/enums for KeyRegistrar
	CURVE_TYPE_KEY_REGISTRAR_UNKNOWN = 0
	CURVE_TYPE_KEY_REGISTRAR_ECDSA   = 1
	CURVE_TYPE_KEY_REGISTRAR_BN254   = 2

	// These are fallback EigenLayer deployment addresses when not specified in context (assumes seploia)
	ALLOCATION_MANAGER_ADDRESS   = "0x42583067658071247ec8CE0A516A58f682002d07"
	DELEGATION_MANAGER_ADDRESS   = "0xD4A7E1Bd8015057293f0D0A557088c286942e84b"
	STRATEGY_MANAGER_ADDRESS     = "0x2E3D6c0744b10eb0A4e6F679F71554a39Ec47a5D"
	KEY_REGISTRAR_ADDRESS        = "0xA4dB30D08d8bbcA00D40600bee9F029984dB162a"
	CROSS_CHAIN_REGISTRY_ADDRESS = "0x287381B1570d9048c4B4C7EC94d21dDb8Aa1352a"
	MULTICHAIN_PROXY_ADMIN       = "0xC5dc0d145a21FDAD791Df8eDC7EbCB5330A3FdB5"
	EIGEN_CONTRACT_ADDRESS       = "0x3B78576F7D6837500bA3De27A60c7f594934027E"
	RELEASE_MANAGER_ADDRESS      = "0xd9Cb89F1993292dEC2F973934bC63B0f2A702776"

	// These are fallback EigenLayer Middleware deployment addresses when not specified in context
	SEPOLIA_BN254_TABLE_CALCULATOR_ADDRESS = "0xa19E3B00cf4aC46B5e6dc0Bbb0Fb0c86D0D65603"
	SEPOLIA_ECDSA_TABLE_CALCULATOR_ADDRESS = "0xaCB5DE6aa94a1908E6FA577C2ade65065333B450"

	MAINNET_BN254_TABLE_CALCULATOR_ADDRESS = "0x55F4b21681977F412B318eCB204cB933bD1dF57c"
	MAINNET_ECDSA_TABLE_CALCULATOR_ADDRESS = "0xA933CB4cbD0C4C208305917f56e0C3f51ad713Fa"
)
