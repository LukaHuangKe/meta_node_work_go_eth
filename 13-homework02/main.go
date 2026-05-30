package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

// 运行：go run .（点表示当前目录下的所有 Go 文件）
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

	// 获取Counter合约地址
	counterAddress := os.Getenv("COUNTER_CONTRACT_ADDRESS")
	if counterAddress == "" {
		log.Fatal("COUNTER_CONTRACT_ADDRESS is not set")
	}

	privateKey := os.Getenv("PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("PRIVATE_KEY is not set")
	}

	// 连接到Ethereum节点
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	// 构建合约对象
	counter, err := NewCounter(common.HexToAddress(counterAddress), client)
	if err != nil {
		fmt.Println("NewCounter error : ", err)
	}

	// 应该是0
	getNumber(counter)

	// 调用下set方法
	setNumber(ctx, client, counter, counterAddress, privateKey, 200)

	// 应该是100
	time.Sleep(10 * time.Second)
	getNumber(counter)
}

func setNumber(ctx context.Context, client *ethclient.Client, counter *Counter, counterAddress string, privateKey string, num int64) {
	bigNum := big.NewInt(num)
	// 获取当前区块链的ChainID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		fmt.Println("获取ChainID失败:", err)
		return
	}

	// 解析私钥
	privateKeyECDSA, err := crypto.HexToECDSA(trim0x(privateKey))
	if err != nil {
		fmt.Println("crypto.HexToECDSA error ,", err)
		return
	}
	// 获取发送方地址
	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	fromAddr := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 获得合约地址
	contractAddr := common.HexToAddress(counterAddress)

	// 计算Gas Tap
	gasTipCap, _ := client.SuggestGasTipCap(context.Background())
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

	// gasFeeCap（Gas 价格上限） = base fee * 2 + tip cap（简单策略）
	gasFeeCap := new(big.Int).Add(
		new(big.Int).Mul(baseFee, big.NewInt(2)),
		gasTipCap,
	)

	fmt.Printf("baseFee: %v, gasTipCap: %v, gasFeeCap: %v\n", baseFee, gasTipCap, gasFeeCap)

	// 计算GasLimit
	parsedABI, err := CounterMetaData.GetAbi()
	if err != nil {
		log.Fatalf("failed to get CounterMetaData ABI: %v", err)
	}
	callData, err := parsedABI.Pack("setNumber", bigNum)
	if err != nil {
		log.Fatalf("failed to pack setNumber data: %v", err)
	}
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddr,
		To:   &contractAddr,
		Data: callData,
	})
	if err != nil {
		log.Fatalf("failed to estimate gas: %v", err)
	}
	fmt.Printf("gasLimit: %v\n", gasLimit)
	// 增加 20% 的缓冲，避免 Gas 不足
	gasLimit = gasLimit * 120 / 100

	// 获取nonce（可选）
	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		log.Fatalf("failed to get nonce: %v", err)
	}

	//构建参数对象
	opts, err := bind.NewKeyedTransactorWithChainID(privateKeyECDSA, chainID)
	if err != nil {
		fmt.Println("bind.NewKeyedTransactorWithChainID error ,", err)
		return
	}
	//设置参数
	opts.GasFeeCap = gasFeeCap
	opts.GasLimit = gasLimit
	opts.GasTipCap = gasTipCap
	opts.Nonce = big.NewInt(0).SetUint64(nonce) //这个不一定要设置

	//调用合约transfer方法
	tx, err := counter.SetNumber(opts, bigNum)
	if err != nil {
		fmt.Println("ounter.SetNumber error ,", err)
		return
	}

	fmt.Println("setNumber tx : ", tx.Hash().Hex())
}

func getNumber(counter *Counter) {
	// 调用合约方法获取当前计数
	number, err := counter.Number(nil)
	if err != nil {
		fmt.Println("counter.Number error : ", err)
	}
	fmt.Println("number is : ", number)
}

// trim0x 移除十六进制字符串前缀 "0x"
func trim0x(s string) string {
	if len(s) >= 2 && s[0:2] == "0x" {
		return s[2:]
	}
	return s
}
