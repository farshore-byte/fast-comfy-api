package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"farshore.ai/fast-comfy-api/core"
)

func main() {
	// 使用提供的测试 host
	host := "u459706-9067-0fe83346.westx.seetacloud.com:8443"
	clientID := "test-client"

	// 创建消息工作者
	worker := core.NewMessageWorker(host, clientID)

	// 启动消息工作者
	worker.Start()

	log.Printf("开始监听 WebSocket 消息，服务器: %s, 客户端ID: %s", host, clientID)
	log.Printf("按 Ctrl+C 停止监听...")
	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh

	// 停止消息工作者
	log.Println("正在停止消息工作者...")
	worker.Stop()

	log.Println("程序已退出")
}
