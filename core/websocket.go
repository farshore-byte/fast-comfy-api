package core

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ComfyUIMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

const (
	WS_CONNECTED         = "ws_connected"
	WS_RECONNECT_ATTEMPT = "ws_reconnect_attempt"
	WS_RECONNECT_FAILED  = "ws_reconnect_failed"
	WS_CONNECTION_ERROR  = "ws_connection_error"
	WS_READ_ERROR        = "ws_read_error"
	WS_PARSE_ERROR       = "ws_parse_error"

	MAX_RETRIES    = 3
	RETRY_INTERVAL = 5 * time.Second
)

type WsClient struct {
	host     string
	clientID string
	conn     *websocket.Conn
	msgChan  chan ComfyUIMessage
	stopCh   chan struct{}
	wg       sync.WaitGroup
	mu       sync.Mutex
	closed   bool
}

func NewWsClient(host string, clientID string) *WsClient {
	return &WsClient{
		host:     host,
		clientID: clientID,
		msgChan:  make(chan ComfyUIMessage, 1000),
		stopCh:   make(chan struct{}),
	}
}

// Start 启动 WebSocket 客户端
func (c *WsClient) Start() error {
	c.wg.Add(1)
	var firstErr error

	go func() {
		defer c.wg.Done()
		retryCount := 0
		for {
			select {
			case <-c.stopCh:
				return
			default:
			}

			err := c.listen()
			if err != nil {
				if firstErr == nil {
					firstErr = err // 记录第一次连接失败
				}
				retryCount++
				c.pushMsg(WS_RECONNECT_ATTEMPT, map[string]interface{}{"msg": err.Error()})
				if retryCount > MAX_RETRIES {
					c.pushMsg(WS_RECONNECT_FAILED, map[string]interface{}{"error": err.Error()})
					return
				}
				log.Printf("WebSocket异常: %v, %d秒后重试 (第 %d 次)", err, RETRY_INTERVAL/time.Second, retryCount)
				time.Sleep(RETRY_INTERVAL)
			} else {
				retryCount = 0
				c.pushMsg(WS_CONNECTED, map[string]interface{}{"msg": "WebSocket 已连接"})
			}
		}
	}()

	return firstErr
}

// Stop 停止 WebSocket 客户端
func (c *WsClient) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	close(c.stopCh)

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("关闭 WebSocket 失败: %w", err)
		}
	}

	c.wg.Wait()
	close(c.msgChan)
	return nil
}

// Messages 获取消息通道
func (c *WsClient) Messages() <-chan ComfyUIMessage {
	return c.msgChan
}

// 内部监听逻辑
func (c *WsClient) listen() error {
	u := url.URL{
		Scheme: "wss",
		Host:   c.host,
		Path:   "/ws",
	}
	q := u.Query()
	q.Set("clientId", c.clientID)
	u.RawQuery = q.Encode()

	log.Printf("连接 WebSocket: %s", u.String())
	dialer := websocket.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		c.pushMsg(WS_CONNECTION_ERROR, map[string]interface{}{"msg": err.Error()})
		return fmt.Errorf("连接失败: %w", err)
	}
	c.conn = conn

	for {
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			c.pushMsg(WS_READ_ERROR, map[string]interface{}{"msg": err.Error()})
			return fmt.Errorf("读取消息失败: %w", err)
		}

		if msgType != websocket.TextMessage {
			continue
		}

		var parsed ComfyUIMessage
		if err := json.Unmarshal(msg, &parsed); err != nil {
			c.pushMsg(WS_PARSE_ERROR, map[string]interface{}{"msg": err.Error()})
			continue
		}

		select {
		case c.msgChan <- parsed:
		default:
			log.Printf("消息队列已满，丢弃消息 type=%s", parsed.Type)
		}
	}
}

// pushMsg 辅助函数
func (c *WsClient) pushMsg(typ string, v interface{}) {
	select {
	case c.msgChan <- ComfyUIMessage{Type: typ, Data: mustMarshal(v)}:
	default:
		log.Printf("消息队列已满，丢弃内部消息 type=%s", typ)
	}
}

func mustMarshal(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
