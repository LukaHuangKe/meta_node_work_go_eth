package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

// 使用示例：
//
//	 # 查询区块
//	 ## 查询最新区块
//		go run main.go
//	 ## 查询指定区块
//		go run main.go -mode 0 -number 123456
//
//	# 发起转账
//	go run main.go -mode 1 -eth 0.1 -toAddress 0x6610ac67b16bb981B66A96174330c2225b3cCac4
func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found or cannot be loaded: %v", err)
	}

	// 从.env文件中获取 RPC URL
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		log.Fatal("ETH_RPC_URL is not set")
	}

	modeFlag := flag.Uint64("mode", 0, "exec mode (0: query, 1: transfer)")
	numberFlag := flag.Uint64("number", 0, "block number to query (0 means skip)")
	ethFlag := flag.String("eth", "", "eth to transfer")
	toAddressFlag := flag.String("toAddress", "", "toAddress to transfer")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	if *modeFlag == 1 {
		// 发起转账
		handlerTransfer(ctx, client, *ethFlag, *toAddressFlag)
	} else {
		// 查询区块
		handlerQueryBlock(ctx, client, numberFlag)
	}
}

func handlerQueryBlock(ctx context.Context, client *ethclient.Client, blockNum *uint64) {
	log.Printf("handling block %d", *blockNum)
	var queryBlockNum *big.Int
	title := "latest block"
	if *blockNum != 0 {
		// 必须要这样转换，不能用big.NewInt(int64(*blockNumberFlag))，不然可能会溢出
		queryBlockNum = big.NewInt(0).SetUint64(*blockNum)
		title = fmt.Sprintf("Block %d", *blockNum)
	}

	block, err := client.BlockByNumber(ctx, queryBlockNum)
	if err != nil {
		log.Fatalf("failed to get block %d: %v", *blockNum, err)
	}
	printBlockInfo(title, block)
}

func handlerTransfer(ctx context.Context, client *ethclient.Client, eth string, toAddress string) {
	log.Printf("handling transfer %s to %s", eth, toAddress)

	if len(eth) == 0 {
		log.Fatal("eth amount is required")
	}
	if len(toAddress) == 0 {
		log.Fatal("toAddress is required")
	}

	privKeyHex := os.Getenv("SENDER_PRIVATE_KEY")
	if privKeyHex == "" {
		log.Fatal("SENDER_PRIVATE_KEY is not set (required for send mode)")
	}

	// 解析私钥
	privKey, err := crypto.HexToECDSA(trim0x(privKeyHex))
	if err != nil {
		log.Fatalf("invalid private key: %v", err)
	}

	// 获取发送方地址
	publicKey := privKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)
	toAddr := common.HexToAddress(toAddress)

	// 获取链 ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("failed to get chain id: %v", err)
	}

	// 获取 nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	// 获取建议的 Gas 价格（使用 EIP-1559 动态费用）
	gasTipCap, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		log.Fatalf("failed to get gas tip cap: %v", err)
	}

	// 获取 base fee，计算 fee cap
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		log.Fatalf("failed to get header: %v", err)
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// 如果不支持 EIP-1559，使用传统 gas price
		gasPrice, err := client.SuggestGasPrice(ctx)
		if err != nil {
			log.Fatalf("failed to get gas price: %v", err)
		}
		baseFee = gasPrice
	}

	// fee cap = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	// 估算 Gas Limit（普通转账固定为 21000）
	gasLimit := uint64(21000)

	// 转换 ETH 金额为 Wei
	// amountEth * 1e18
	amountEth := cast.ToFloat64(eth)
	amountWei := new(big.Float).Mul(
		big.NewFloat(amountEth),
		big.NewFloat(1e18),
	)
	valueWei, _ := amountWei.Int(nil)

	// 检查余额是否足够
	balance, err := client.BalanceAt(ctx, fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance: %v", err)
	}

	// 计算总费用：value + gasFeeCap * gasLimit
	totalCost := new(big.Int).Add(
		valueWei,
		new(big.Int).Mul(gasFeeCap, big.NewInt(int64(gasLimit))),
	)

	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("insufficient balance: have %s wei, need %s wei", balance.String(), totalCost.String())
	}

	// 构造交易（EIP-1559 动态费用交易）
	txData := &types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &toAddr,
		Value:     valueWei,
		Data:      nil,
	}
	tx := types.NewTx(txData)

	// 签名交易
	signer := types.NewLondonSigner(chainID)
	signedTx, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		log.Fatalf("failed to sign transaction: %v", err)
	}

	// 发送交易
	if err := client.SendTransaction(ctx, signedTx); err != nil {
		log.Fatalf("failed to send transaction: %v", err)
	}

	// 输出交易信息
	fmt.Println("=== Transaction Sent ===")
	fmt.Printf("From       : %s\n", fromAddr.Hex())
	fmt.Printf("To         : %s\n", toAddr.Hex())
	fmt.Printf("Value      : %s ETH (%s Wei)\n", fmt.Sprintf("%.6f", amountEth), valueWei.String())
	fmt.Printf("Gas Limit  : %d\n", gasLimit)
	fmt.Printf("Gas Tip Cap: %s Wei\n", gasTipCap.String())
	fmt.Printf("Gas Fee Cap: %s Wei\n", gasFeeCap.String())
	fmt.Printf("Nonce      : %d\n", nonce)
	fmt.Printf("Tx Hash    : %s\n", signedTx.Hash().Hex())
	fmt.Println("\nTransaction is pending. Use --tx flag to query status:")
	fmt.Printf("  go run main.go --tx %s\n", signedTx.Hash().Hex())
}

// printBlockInfo 打印详细的区块信息（抄的02）
func printBlockInfo(title string, block *types.Block) {
	fmt.Println("======================================")
	fmt.Println(title)
	fmt.Println("======================================")
	//fmt.Printf("Block: %+v\n", block) //这行打印没啥意义

	// 基本信息
	fmt.Printf("Number       : %d\n", block.Number().Uint64())
	fmt.Printf("Hash         : %s\n", block.Hash().Hex())
	fmt.Printf("Parent Hash  : %s\n", block.ParentHash().Hex())

	// 时间信息
	blockTime := time.Unix(int64(block.Time()), 0)
	fmt.Printf("Time         : %s\n", blockTime.Format(time.RFC3339))
	fmt.Printf("Time (Local) : %s\n", blockTime.Local().Format("2006-01-02 15:04:05 MST"))

	// Gas 信息
	gasUsed := block.GasUsed()
	gasLimit := block.GasLimit()
	gasUsagePercent := float64(gasUsed) / float64(gasLimit) * 100
	fmt.Printf("Gas Used     : %d (%.2f%%)\n", gasUsed, gasUsagePercent)
	fmt.Printf("Gas Limit    : %d\n", gasLimit)

	// 交易信息
	txCount := len(block.Transactions())
	fmt.Printf("Tx Count     : %d\n", txCount)

	// 区块根信息（Merkle 树根）
	fmt.Printf("State Root   : %s\n", block.Root().Hex())
	fmt.Printf("Tx Root      : %s\n", block.TxHash().Hex())
	fmt.Printf("Receipt Root : %s\n", block.ReceiptHash().Hex())

	// 区块大小估算（简化版，实际大小还包括其他字段）
	if txCount > 0 {
		fmt.Printf("\nFirst Tx Hash: %s\n", block.Transactions()[0].Hash().Hex())
		if txCount > 1 {
			fmt.Printf("Last Tx Hash : %s\n", block.Transactions()[txCount-1].Hash().Hex())
		}
	}

	// 难度信息（PoW 相关，PoS 后基本固定）
	fmt.Printf("Difficulty   : %s\n", block.Difficulty().String())

	// 区块奖励相关信息
	coinbase := block.Coinbase()
	if coinbase != (common.Address{}) {
		fmt.Printf("Coinbase     : %s\n", coinbase.Hex())
	}

	fmt.Println("======================================")
	fmt.Println()
}

// trim0x 移除十六进制字符串前缀 "0x"
func trim0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s[2:]
	}
	return s
}
