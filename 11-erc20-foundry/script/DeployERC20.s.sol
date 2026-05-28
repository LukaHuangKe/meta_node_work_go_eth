// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Script, console} from "forge-std/Script.sol";
import {MyERC20} from "../src/MyERC20.sol";

/**
 * @title DeployERC20
 * @dev 部署 ERC20 代币合约的脚本
 */
contract DeployERC20Script is Script {
    // 代币名称
    string constant TOKEN_NAME = "MyToken";
    // 代币符号
    string constant TOKEN_SYMBOL = "MTK";
    // 初始供应量（1000 个代币，18 位小数）
    uint256 constant INITIAL_SUPPLY = 1000 * 10 ** 18;

    function setUp() public {}

    function run() public returns (address) {
        // 获取部署者地址作为初始代币接收者
        address deployer = msg.sender;

        // 如果要部署到本地链，可以使用以下方式获取地址
        // address deployer = vm.envAddress("DEPLOYER_ADDRESS");

        console.log("Deploying ERC20 token...");
        console.log("Token Name:", TOKEN_NAME);
        console.log("Token Symbol:", TOKEN_SYMBOL);
        console.log("Initial Supply:", INITIAL_SUPPLY);
        console.log("Initial Recipient:", deployer);

        vm.startBroadcast();

        // 部署 ERC20 合约
        MyERC20 token = new MyERC20(
            TOKEN_NAME,
            TOKEN_SYMBOL,
            INITIAL_SUPPLY,
            deployer
        );

        vm.stopBroadcast();

        console.log("ERC20 Token deployed at:", address(token));
        console.log("Deployer balance:", token.balanceOf(deployer));

        return address(token);
    }
}