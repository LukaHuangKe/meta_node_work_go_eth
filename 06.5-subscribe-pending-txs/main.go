package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
)

// 08-subscribe-pending-txs-ethclient.go
// 使用 go-ethereum 库订阅 newPendingTransactions
// 这是更可靠的方式，因为 go-ethereum 处理了所有底层细节

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found or cannot be loaded: %v", err)
	}

	rpcURL := os.Getenv("ETH_WS_URL")
	if rpcURL == "" {
		// 尝试从 ETH_RPC_URL 推导 (替换 http -> ws, https -> wss)
		httpURL := os.Getenv("ETH_RPC_URL")
		if httpURL != "" {
			if len(httpURL) >= 5 && httpURL[:5] == "https" {
				rpcURL = "wss" + httpURL[5:]
			} else if len(httpURL) >= 4 && httpURL[:4] == "http" {
				rpcURL = "ws" + httpURL[4:]
			}
		}
	}

	if rpcURL == "" {
		log.Fatal("ETH_WS_URL or ETH_RPC_URL must be set")
	}
	log.Printf("using WebSocket URL: %s", rpcURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 连接到以太坊节点
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		log.Fatalf("failed to connect to Ethereum node: %v", err)
	}
	defer client.Close()

	fmt.Println("Connected to Ethereum node")

	// 订阅 pending transactions
	// 注意：SubscribeFilterLogs 主要用于订阅日志，
	// 对于 pending transactions，我们需要使用底层 RPC 客户端
	// 但是 go-ethereum 的 SubscribeFilterLogs 可以支持订阅 newPendingTransactions

	// 创建用于接收交易哈希的 channel
	txHashes := make(chan common.Hash, 100)

	// 使用底层 RPC 客户端订阅
	// 注意：不是所有节点都支持这个订阅，特别是公共 RPC 可能有限制
	rpcClient := client.Client()

	var sub ethereum.Subscription
	sub, err = rpcClient.EthSubscribe(ctx, txHashes, "newPendingTransactions")
	if err != nil {
		log.Fatalf("failed to subscribe to pending transactions: %v", err)
	}
	defer sub.Unsubscribe()

	fmt.Println("Subscribed to pending transactions")
	fmt.Println("Listening for pending transactions...\n")

	// 捕获 Ctrl+C 退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	txCount := 0
	for {
		select {
		case txHash := <-txHashes:
			txCount++
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("[%s] Pending Transaction #%d\n",
				time.Now().Format(time.RFC3339),
				txCount)
			fmt.Printf("Transaction Hash: %s\n", txHash.Hex())

			// 尝试获取完整交易（可选，可能会失败，因为 pending tx 可能还没被节点接收）
			func() {
				ctxGet, cancelGet := context.WithTimeout(ctx, 5*time.Second) // 增加超时时间
				defer cancelGet()
				tx, isPending, err := client.TransactionByHash(ctxGet, txHash)
				if err != nil {
					// 查询失败（可能是网络问题，或者节点还没收到这笔交易）
					fmt.Printf("Query failed: %v\n", err)
				} else if !isPending {
					// 查询成功，但交易已经不再 pending 了（已经被打包进区块了）
					// 让我们看看它被打包到哪个区块了
					blockNum, err := client.BlockNumber(ctxGet)
					if err == nil {
						fmt.Printf("(already mined at block %d, not pending anymore)\n", blockNum)
					} else {
						fmt.Printf("(already mined, not pending anymore)\n")
					}
				} else {
					// 获取发送者地址
					signer := types.LatestSignerForChainID(tx.ChainId())
					from, err := types.Sender(signer, tx)
					if err == nil {
						fmt.Printf("From: %s\n", from.Hex())
					}

					// 接收者地址可能为 nil（合约创建交易）
					if tx.To() != nil {
						fmt.Printf("To: %s\n", tx.To().Hex())
					} else {
						fmt.Printf("To: (contract creation)\n")
					}

					fmt.Printf("Value: %s\n", tx.Value().String())
					fmt.Printf("Gas: %d\n", tx.Gas())
					fmt.Printf("Gas Price: %s\n", tx.GasPrice().String())
				}
			}()

			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		case err := <-sub.Err():
			log.Printf("subscription error: %v", err)
			return

		case sig := <-sigCh:
			fmt.Printf("\nReceived signal %s, shutting down...\n", sig.String())
			return

		case <-ctx.Done():
			fmt.Println("Context cancelled, shutting down...")
			return
		}
	}
}
