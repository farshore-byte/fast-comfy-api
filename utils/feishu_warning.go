package utils

import (
	"bytes"
	"encoding/json"
	"farshore.ai/fast-comfy-api/config"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const PrefixWord = "🚨[fast-comfy-api]"

// 全局变量
var Feishu *FeishuClient
var (
	// 记录每个报警类型+主机的上次发送时间
	alertCache = make(map[string]time.Time)
	alertMu    sync.Mutex
	// 限频间隔
	alertInterval = config.WarningInterval * time.Second
)

// InitFeishuClient 函数初始化全局 FeishuClient
func InitFeishuClient(webhook string) {
	Feishu = NewFeishuClient(webhook)
	log.Println("✅ 飞书客户端初始化成功")
}

// Feishu 消息结构体
type FeishuMsg struct {
	MsgType string      `json:"msg_type"`
	Content interface{} `json:"content"`
}

type TextContent struct {
	Text string `json:"text"`
}

// FeishuClient 用于发送消息
type FeishuClient struct {
	Webhook string
}

// NewFeishuClient 初始化一个 FeishuClient
func NewFeishuClient(webhook string) *FeishuClient {
	return &FeishuClient{Webhook: webhook}
}

// SendFeishuMsgAsync 异步发送飞书消息
func (c *FeishuClient) SendFeishuMsgAsync(text string) {
	go func() {
		if err := c.sendFeishuMsg(text); err != nil {
			log.Printf("[Feishu] 消息发送失败: %v", err)
		}
	}()
}

// sendFeishuMsg 内部发送函数
func (c *FeishuClient) sendFeishuMsg(text string) error {
	if strings.Contains(text, "JSON 解析失败") {
		log.Println("⚠️ 忽略 JSON 解析失败 报错，不发送飞书消息")
		return nil
	}

	if c.Webhook == "" {
		return fmt.Errorf("feishu webhook not configured")
	}

	payload := FeishuMsg{
		MsgType: "text",
		Content: TextContent{
			Text: PrefixWord + text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("json marshal error: %v", err)
	}

	resp, err := http.Post(c.Webhook, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("http post error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("feishu returned status %d", resp.StatusCode)
	}

	log.Println("✅ 飞书消息发送成功")
	return nil
}

// FeishuWarning 统一发送报警
// errorType: queue_warning / cpu_warning / gpu_warning / ram_warning ...
// host: 资源或服务器标识
// message: 报警信息
func (c *FeishuClient) InternalFeishuWarning(errorType, host, message string) {
	key := fmt.Sprintf("%s_%s", errorType, host)

	alertMu.Lock()
	defer alertMu.Unlock()

	now := time.Now()
	lastTime, exists := alertCache[key]

	if !exists || now.Sub(lastTime) >= alertInterval {
		if Feishu != nil {
			c.SendFeishuMsgAsync(message)
			alertCache[key] = now
			log.Printf("✅ 飞书报警发送: %s", message)
		} else {
			log.Printf("⚠️ 飞书客户端未初始化，无法发送报警: %s", message)
		}
	}
}
