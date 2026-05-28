# ERC20 Token with OpenZeppelin

这是一个使用 Foundry 和 OpenZeppelin 部署 ERC20 代币合约的示例项目。

## 项目结构

```
11-erc20-foundry/
├── src/
│   └── MyERC20.sol          # ERC20 代币合约
├── script/
│   └── DeployERC20.s.sol    # 部署脚本
├── test/                     # 测试文件
├── lib/                      # 依赖库
│   ├── forge-std/           # Foundry 标准库
│   └── openzeppelin-contracts/  # OpenZeppelin 合约库
└── foundry.toml             # Foundry 配置文件
```

## 合约说明

### MyERC20.sol

一个基于 OpenZeppelin 的 ERC20 代币合约，支持：
- 自定义代币名称和符号
- 设置初始供应量
- 将初始代币铸造给指定地址
- 可铸造新代币的功能

**构造函数参数：**
- `name`: 代币名称（例如：MyToken）
- `symbol`: 代币符号（例如：MTK）
- `initialSupply`: 初始供应量（wei 单位，18 位小数）
- `recipient`: 初始代币接收地址

## 使用方法

### 1. 编译合约

```bash
forge build
```

编译后，Foundry 会在 `out/` 目录下自动生成包含 ABI 的 JSON 文件。例如 `MyERC20.sol` 会生成 `out/MyERC20.sol/MyERC20.json`。

### 2. 提取 ABI

编译完成后，可以使用以下方法提取合约的 ABI：

#### 方法一：使用 jq 提取（推荐）

```bash
# 查看 ABI
jq '.abi' out/MyERC20.sol/MyERC20.json

# 保存 ABI 到单独文件
jq '.abi' out/MyERC20.sol/MyERC20.json > MyERC20.abi.json
```

**注意：** 如果系统没有安装 `jq`，可以通过以下方式安装：
- macOS: `brew install jq`
- Ubuntu/Debian: `sudo apt-get install jq`
- 或者使用下面的手动提取方法

#### 方法二：手动提取

直接打开 `out/MyERC20.sol/MyERC20.json` 文件，复制其中的 `abi` 字段内容。

**注意：** 生成的 JSON 文件包含：
- `abi`: 合约的 ABI（应用二进制接口）
- `bytecode`: 完整字节码
- `deployedBytecode`: 部署后的字节码
- 其他编译元数据

### 4. 运行测试（如果有）

```bash
forge test
```

### 5. 部署到本地链

首先启动本地 Anvil 节点：

```bash
anvil
```

然后在另一个终端运行部署脚本：

```bash
# 使用默认私钥部署
forge script script/DeployERC20.s.sol:DeployERC20Script --rpc-url http://localhost:8545 --broadcast --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# 或者使用环境变量中的私钥
forge script script/DeployERC20.s.sol:DeployERC20Script --rpc-url http://localhost:8545 --broadcast --private-key $PRIVATE_KEY
```

### 6. 部署到测试网（Sepolia）

```bash
# 需要先设置 RPC URL 和私钥
forge script script/DeployERC20.s.sol:DeployERC20Script \
  --rpc-url $SEPOLIA_RPC_URL \
  --broadcast \
  --verify \
  --etherscan-api-key $ETHERSCAN_API_KEY \
  --private-key $PRIVATE_KEY
```

### 7. 仅模拟部署（不实际部署）

```bash
forge script script/DeployERC20.s.sol:DeployERC20Script --rpc-url http://localhost:8545
```

## 环境变量（可选）

可以创建 `.env` 文件来存储配置：

```bash
# .env
PRIVATE_KEY=your_private_key_here
SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/your_key
ETHERSCAN_API_KEY=your_etherscan_api_key
```

然后在脚本中使用：
```solidity
address deployer = vm.envAddress("DEPLOYER_ADDRESS")
```

## 代币参数

默认配置在 `DeployERC20.s.sol` 中：
- **名称**: MyToken
- **符号**: MTK
- **初始供应量**: 1000 * 10^18 (1000 个代币)
- **初始接收者**: 部署者地址

可以在部署脚本中修改这些参数。

## 验证合约（部署到测试网/主网后）

```bash
forge verify-contract <CONTRACT_ADDRESS> \
  src/MyERC20.sol:MyERC20 \
  --etherscan-api-key $ETHERSCAN_API_KEY \
  --constructor-args $(cast abi-encode "constructor(string,string,uint256,address)" "MyToken" "MTK" "1000000000000000000000" <DEPLOYER_ADDRESS>)
```

## 与合约交互

部署后，可以使用 `cast` 命令与合约交互：

```bash
# 查询代币名称
cast call <CONTRACT_ADDRESS> "name()(string)" --rpc-url http://localhost:8545

# 查询代币符号
cast call <CONTRACT_ADDRESS> "symbol()(string)" --rpc-url http://localhost:8545

# 查询总供应量
cast call <CONTRACT_ADDRESS> "totalSupply()(uint256)" --rpc-url http://localhost:8545

# 查询账户余额
cast call <CONTRACT_ADDRESS> "balanceOf(address)(uint256)" <ADDRESS> --rpc-url http://localhost:8545

# 转账代币
cast send <CONTRACT_ADDRESS> "transfer(address,uint256)" <TO_ADDRESS> <AMOUNT> \
  --rpc-url http://localhost:8545 \
  --private-key $PRIVATE_KEY
```

## 依赖

- [Foundry](https://book.getfoundry.sh/) - 以太坊开发工具链
- [OpenZeppelin Contracts](https://github.com/OpenZeppelin/openzeppelin-contracts) - 安全智能合约库

## 参考资源

- [Foundry 文档](https://book.getfoundry.sh/)
- [OpenZeppelin ERC20 文档](https://docs.openzeppelin.com/contracts/5.x/erc20)
- [ERC-20 标准](https://eips.ethereum.org/EIPS/eip-20)
