// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.27;

import {Script, console} from "forge-std/Script.sol";
import {ContractsRegistry} from "../src/ContractsRegistry.sol";

contract DeployContractsRegistry is Script {
    ContractsRegistry public registry;

    function setUp() public {}

    function run() public {
        uint deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        registry = new ContractsRegistry();

        console.log("Contracts registry deployed to address:");
        console.log(address(registry));
        vm.stopBroadcast();
    }
}
