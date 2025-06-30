package contracts

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	// EigenLayer contract bindings
	allocationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/AllocationManager"
	crosschainregistry "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/CrossChainRegistry"
	delegationmanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/DelegationManager"
	istrategy "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IStrategy"
	keyregistrar "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/KeyRegistrar"
	releasemanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/ReleaseManager"
	strategymanager "github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/StrategyManager"
)

// ContractType represents different contract types
type ContractType string

const (
	AllocationManagerContract  ContractType = "AllocationManager"
	DelegationManagerContract  ContractType = "DelegationManager"
	StrategyManagerContract    ContractType = "StrategyManager"
	StrategyContract           ContractType = "Strategy"
	ERC20Contract              ContractType = "ERC20"
	KeyRegistrarContract       ContractType = "KeyRegistrar"
	CrossChainRegistryContract ContractType = "CrossChainRegistry"
	ReleaseManagerContract     ContractType = "ReleaseManager"
)

// ContractInfo holds metadata about a contract
type ContractInfo struct {
	Name        string
	Type        ContractType
	Address     common.Address
	Description string
}

// ContractRegistry manages contract instances and metadata
type ContractRegistry struct {
	client    *ethclient.Client
	contracts map[ContractType]map[common.Address]*ContractInstance
	metadata  map[common.Address]ContractInfo
}

// ContractInstance holds a contract instance with its metadata
type ContractInstance struct {
	Info     ContractInfo
	Instance interface{}
}

// NewContractRegistry creates a new contract registry
func NewContractRegistry(client *ethclient.Client) *ContractRegistry {
	return &ContractRegistry{
		client:    client,
		contracts: make(map[ContractType]map[common.Address]*ContractInstance),
		metadata:  make(map[common.Address]ContractInfo),
	}
}

// RegisterContract registers a contract with the registry
func (cr *ContractRegistry) RegisterContract(info ContractInfo) error {
	if cr.contracts[info.Type] == nil {
		cr.contracts[info.Type] = make(map[common.Address]*ContractInstance)
	}

	// Create the contract instance based on type
	instance, err := cr.createContractInstance(info)
	if err != nil {
		return fmt.Errorf("failed to create contract instance for %s: %w", info.Name, err)
	}

	cr.contracts[info.Type][info.Address] = &ContractInstance{
		Info:     info,
		Instance: instance,
	}
	cr.metadata[info.Address] = info

	return nil
}

// GetContract retrieves a contract instance by type and address
func (cr *ContractRegistry) GetContract(contractType ContractType, address common.Address) (*ContractInstance, error) {
	if cr.contracts[contractType] == nil {
		return nil, fmt.Errorf("no contracts of type %s registered", contractType)
	}

	instance, exists := cr.contracts[contractType][address]
	if !exists {
		return nil, fmt.Errorf("contract of type %s at address %s not found", contractType, address.Hex())
	}

	return instance, nil
}

// GetAllocationManager returns an AllocationManager instance
func (cr *ContractRegistry) GetAllocationManager(address common.Address) (*allocationmanager.AllocationManager, error) {
	instance, err := cr.GetContract(AllocationManagerContract, address)
	if err != nil {
		return nil, err
	}

	am, ok := instance.Instance.(*allocationmanager.AllocationManager)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not an AllocationManager", address.Hex())
	}

	return am, nil
}

// GetDelegationManager returns a DelegationManager instance
func (cr *ContractRegistry) GetDelegationManager(address common.Address) (*delegationmanager.DelegationManager, error) {
	instance, err := cr.GetContract(DelegationManagerContract, address)
	if err != nil {
		return nil, err
	}

	dm, ok := instance.Instance.(*delegationmanager.DelegationManager)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a DelegationManager", address.Hex())
	}

	return dm, nil
}

// GetStrategyManager returns a StrategyManager instance
func (cr *ContractRegistry) GetStrategyManager(address common.Address) (*strategymanager.StrategyManager, error) {
	instance, err := cr.GetContract(StrategyManagerContract, address)
	if err != nil {
		return nil, err
	}

	sm, ok := instance.Instance.(*strategymanager.StrategyManager)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a StrategyManager", address.Hex())
	}

	return sm, nil
}

// GetStrategy returns an IStrategy instance
func (cr *ContractRegistry) GetStrategy(address common.Address) (*istrategy.IStrategy, error) {
	instance, err := cr.GetContract(StrategyContract, address)
	if err != nil {
		return nil, err
	}

	strategy, ok := instance.Instance.(*istrategy.IStrategy)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a Strategy", address.Hex())
	}

	return strategy, nil
}

// GetKeyRegistrar returns a KeyRegistrar instance
func (cr *ContractRegistry) GetKeyRegistrar(address common.Address) (*keyregistrar.KeyRegistrar, error) {
	instance, err := cr.GetContract(KeyRegistrarContract, address)
	if err != nil {
		return nil, err
	}

	keyRegistrar, ok := instance.Instance.(*keyregistrar.KeyRegistrar)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a KeyRegistrar", address.Hex())
	}

	return keyRegistrar, nil
}

// GetCrossChainRegistry returns a CrossChainRegistry instance
func (cr *ContractRegistry) GetCrossChainRegistry(address common.Address) (*crosschainregistry.CrossChainRegistry, error) {
	instance, err := cr.GetContract(CrossChainRegistryContract, address)
	if err != nil {
		return nil, err
	}

	crossChainRegistry, ok := instance.Instance.(*crosschainregistry.CrossChainRegistry)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a CrossChainRegistry", address.Hex())
	}

	return crossChainRegistry, nil
}

// GetERC20 returns an ERC20 bound contract instance
func (cr *ContractRegistry) GetERC20(address common.Address) (*bind.BoundContract, error) {
	instance, err := cr.GetContract(ERC20Contract, address)
	if err != nil {
		return nil, err
	}

	erc20, ok := instance.Instance.(*bind.BoundContract)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not an ERC20", address.Hex())
	}

	return erc20, nil
}

// GetReleaseManager returns a ReleaseManager instance
func (cr *ContractRegistry) GetReleaseManager(address common.Address) (*releasemanager.ReleaseManager, error) {
	instance, err := cr.GetContract(ReleaseManagerContract, address)
	if err != nil {
		return nil, err
	}
	releaseManager, ok := instance.Instance.(*releasemanager.ReleaseManager)
	if !ok {
		return nil, fmt.Errorf("contract at %s is not a ReleaseManager", address.Hex())
	}
	return releaseManager, nil
}

// ListContracts returns all registered contracts of a specific type
func (cr *ContractRegistry) ListContracts(contractType ContractType) []ContractInfo {
	var contracts []ContractInfo

	if typeMap, exists := cr.contracts[contractType]; exists {
		for _, instance := range typeMap {
			contracts = append(contracts, instance.Info)
		}
	}

	return contracts
}

// createContractInstance creates the appropriate contract instance based on type
func (cr *ContractRegistry) createContractInstance(info ContractInfo) (interface{}, error) {
	switch info.Type {
	case AllocationManagerContract:
		return allocationmanager.NewAllocationManager(info.Address, cr.client)
	case DelegationManagerContract:
		return delegationmanager.NewDelegationManager(info.Address, cr.client)
	case StrategyManagerContract:
		return strategymanager.NewStrategyManager(info.Address, cr.client)
	case StrategyContract:
		return istrategy.NewIStrategy(info.Address, cr.client)
	case ERC20Contract:
		return NewERC20Contract(info.Address, cr.client)
	case KeyRegistrarContract:
		return keyregistrar.NewKeyRegistrar(info.Address, cr.client)
	case CrossChainRegistryContract:
		return crosschainregistry.NewCrossChainRegistry(info.Address, cr.client)
	case ReleaseManagerContract:
		return releasemanager.NewReleaseManager(info.Address, cr.client)
	default:
		return nil, fmt.Errorf("unsupported contract type: %s", info.Type)
	}
}

// RegistryBuilder helps build a registry with predefined contracts
type RegistryBuilder struct {
	registry *ContractRegistry
}

// NewRegistryBuilder creates a new registry builder
func NewRegistryBuilder(client *ethclient.Client) *RegistryBuilder {
	return &RegistryBuilder{
		registry: NewContractRegistry(client),
	}
}

// AddEigenLayerCore adds the core EigenLayer contracts
func (rb *RegistryBuilder) AddEigenLayerCore(
	allocationManagerAddr, delegationManagerAddr, strategyManagerAddr common.Address, keystoreRegistrarAddr common.Address, crossChainRegistryAddr common.Address, releaseManagerAddr common.Address,
) (*RegistryBuilder, error) {
	// Register AllocationManager
	err := rb.registry.RegisterContract(ContractInfo{
		Name:        "AllocationManager",
		Type:        AllocationManagerContract,
		Address:     allocationManagerAddr,
		Description: "EigenLayer AllocationManager contract",
	})
	if err != nil {
		return nil, err
	}

	// Register DelegationManager
	err = rb.registry.RegisterContract(ContractInfo{
		Name:        "DelegationManager",
		Type:        DelegationManagerContract,
		Address:     delegationManagerAddr,
		Description: "EigenLayer DelegationManager contract",
	})
	if err != nil {
		return nil, err
	}

	// Register StrategyManager
	err = rb.registry.RegisterContract(ContractInfo{
		Name:        "StrategyManager",
		Type:        StrategyManagerContract,
		Address:     strategyManagerAddr,
		Description: "EigenLayer StrategyManager contract",
	})
	if err != nil {
		return nil, err
	}

	err = rb.registry.RegisterContract(ContractInfo{
		Name:        "KeyRegistrar",
		Type:        KeyRegistrarContract,
		Address:     keystoreRegistrarAddr,
		Description: "EigenLayer KeyRegistrar contract",
	})
	if err != nil {
		return nil, err
	}

	err = rb.registry.RegisterContract(ContractInfo{
		Name:        "CrossChainRegistry",
		Type:        CrossChainRegistryContract,
		Address:     crossChainRegistryAddr,
		Description: "EigenLayer CrossChainRegistry contract",
	})
	if err != nil {
		return nil, err
	}
	err = rb.registry.RegisterContract(ContractInfo{
		Name:        "ReleaseManager",
		Type:        ReleaseManagerContract,
		Address:     releaseManagerAddr,
		Description: "EigenLayer ReleaseManager contract",
	})
	if err != nil {
		return nil, err
	}
	return rb, nil
}

// AddStrategy adds a strategy contract
func (rb *RegistryBuilder) AddStrategy(address common.Address, name string) (*RegistryBuilder, error) {
	err := rb.registry.RegisterContract(ContractInfo{
		Name:        name,
		Type:        StrategyContract,
		Address:     address,
		Description: fmt.Sprintf("Strategy contract: %s", name),
	})
	if err != nil {
		return nil, err
	}
	return rb, nil
}

// AddERC20 adds an ERC20 token contract
func (rb *RegistryBuilder) AddERC20(address common.Address, symbol string) (*RegistryBuilder, error) {
	err := rb.registry.RegisterContract(ContractInfo{
		Name:        symbol,
		Type:        ERC20Contract,
		Address:     address,
		Description: fmt.Sprintf("ERC20 token: %s", symbol),
	})
	if err != nil {
		return nil, err
	}
	return rb, nil
}

func (rb *RegistryBuilder) AddReleaseManager(address common.Address) (*RegistryBuilder, error) {
	err := rb.registry.RegisterContract(ContractInfo{
		Name:    "ReleaseManager",
		Type:    ReleaseManagerContract,
		Address: address,
	})
	if err != nil {
		return nil, err
	}
	return rb, nil
}

// Build returns the constructed registry
func (rb *RegistryBuilder) Build() *ContractRegistry {
	return rb.registry
}
