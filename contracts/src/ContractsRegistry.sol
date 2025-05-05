pragma solidity ^0.8.27;


contract ContractsRegistry {

    mapping(string => address) public nameToAddress;

    function registerContract(string memory contractName,address contractAddress) public {
        nameToAddress[contractName] = contractAddress;
    }


}