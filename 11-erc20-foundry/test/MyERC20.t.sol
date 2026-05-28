// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Test, console} from "forge-std/Test.sol";
import {MyERC20} from "../src/MyERC20.sol";

/**
 * @title MyERC20Test
 * @dev ERC20 代币合约的测试文件
 */
contract MyERC20Test is Test {
    MyERC20 public token;
    address public deployer;
    address public user1;
    address public user2;

    string constant TOKEN_NAME = "MyToken";
    string constant TOKEN_SYMBOL = "MTK";
    uint256 constant INITIAL_SUPPLY = 1000 * 10 ** 18;

    function setUp() public {
        // 设置测试账户
        deployer = address(this);
        user1 = address(0x1);
        user2 = address(0x2);

        // 部署 ERC20 合约
        token = new MyERC20(
            TOKEN_NAME,
            TOKEN_SYMBOL,
            INITIAL_SUPPLY,
            deployer
        );
    }

    function testInitialSupply() public view {
        assertEq(token.totalSupply(), INITIAL_SUPPLY);
        assertEq(token.balanceOf(deployer), INITIAL_SUPPLY);
    }

    function testTokenMetadata() public view {
        assertEq(token.name(), TOKEN_NAME);
        assertEq(token.symbol(), TOKEN_SYMBOL);
        assertEq(token.decimals(), 18);
    }

    function testTransfer() public {
        uint256 transferAmount = 100 * 10 ** 18;

        // 从 deployer 转账给 user1
        token.transfer(user1, transferAmount);

        assertEq(token.balanceOf(deployer), INITIAL_SUPPLY - transferAmount);
        assertEq(token.balanceOf(user1), transferAmount);
    }

    function testTransferFrom() public {
        uint256 approveAmount = 200 * 10 ** 18;
        uint256 transferAmount = 150 * 10 ** 18;

        // deployer 批准 user1 可以花费代币
        token.approve(user1, approveAmount);

        // 切换用户上下文
        vm.prank(user1);
        token.transferFrom(deployer, user2, transferAmount);

        assertEq(token.balanceOf(deployer), INITIAL_SUPPLY - transferAmount);
        assertEq(token.balanceOf(user2), transferAmount);
        assertEq(token.allowance(deployer, user1), approveAmount - transferAmount);
    }

    function testMint() public {
        uint256 mintAmount = 500 * 10 ** 18;

        token.mint(user1, mintAmount);

        assertEq(token.totalSupply(), INITIAL_SUPPLY + mintAmount);
        assertEq(token.balanceOf(user1), mintAmount);
    }

    function testTransferInsufficientBalance() public {
        uint256 excessAmount = INITIAL_SUPPLY + 1;

        vm.expectRevert();
        token.transfer(user1, excessAmount);
    }

    function testTransferFromInsufficientAllowance() public {
        uint256 approveAmount = 100 * 10 ** 18;
        uint256 transferAmount = 200 * 10 ** 18;

        token.approve(user1, approveAmount);

        vm.prank(user1);
        vm.expectRevert();
        token.transferFrom(deployer, user2, transferAmount);
    }
}