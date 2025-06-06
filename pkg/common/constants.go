package common

// Project structure constants
const (
	// ContractsDir is the subdirectory name for contract components
	ContractsDir = "contracts"

	// Makefile is the name of the makefile used for root level operations
	Makefile = "Makefile"

	// ContractsMakefile is the name of the makefile used for contract level operations
	ContractsMakefile = "Makefile"

	// DevkitConfigFile is the name of the config YAML used to configure devkit operations
	DevkitConfigFile = ".config.devkit.yml"

	// Filename for devkit project config
	BaseConfig = "config.yaml"

	// Filename for zeus config
	ZeusConfig = ".zeus"

	// Docker open timeout
	DockerOpenTimeoutSeconds = 10

	// Docker open retry interval in milliseconds
	DockerOpenRetryIntervalMilliseconds = 500

	// Default chainId for Anvil
	DefaultAnvilChainId = 31337

	// DevkitLatestContextVersion is the latest version of the context file
	DevkitLatestContextVersion = "0.0.5"
)
