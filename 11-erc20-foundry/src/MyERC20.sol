// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {ERC20} from "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title MyERC20
 * @dev 一个简单的 ERC20 代币合约，使用 OpenZeppelin 实现
 */
contract MyERC20 is ERC20 {
    /**
     * @dev 构造函数
     * @param name 代币名称
     * @param symbol 代币符号
     * @param initialSupply 初始供应量（wei 单位）
     * @param recipient 初始代币接收地址
     */
    constructor(
        string memory name,
        string memory symbol,
        uint256 initialSupply,
        address recipient
    ) ERC20(name, symbol) {
        // 将初始供应量的代币铸造给指定地址
        _mint(recipient, initialSupply);
    }

    /**
     * @dev 允许合约所有者铸造新代币
     * @param to 接收代币的地址
     * @param amount 铸造的代币数量
     */
    function mint(address to, uint256 amount) public {
        _mint(to, amount);
    }
}