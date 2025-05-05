// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.27;

import {Script, console} from "forge-std/Script.sol";
import {ContractsRegistry} from "../src/ContractsRegistry.sol";
import "forge-std/Test.sol";

contract WriteToContractsRegistry is Script,Test {
    ContractsRegistry public registry;
    address public constant CONTRACTS_REGISTRY = 0x5FbDB2315678afecb367f032d93F642f64180aa3; // always at this address since we deploy it first every time using  anvil 0 index key
    string public outputPath;
    function setUp() public {

        registry = ContractsRegistry(CONTRACTS_REGISTRY);

    }

    function run(string memory outputFileName) public {

        outputPath = string(bytes(string.concat("script/output/", outputFileName)));
        string memory output_data = vm.readFile(outputPath);
        uint deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        address allocationManager = stdJson.readAddress(output_data, ".addresses.allocationManager");
        vm.startBroadcast(deployerPrivateKey);
        registry.registerContract("allocationManager",allocationManager);

       

       
        vm.stopBroadcast();
    }
}
