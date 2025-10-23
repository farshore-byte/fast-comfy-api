package core

import (
	"encoding/json"
	"sync"

	"farshore.ai/fast-comfy-api/model"
)

// 定义接口
type TaskNotifier interface {
	NotifyTaskDone(promptID string, addresses []model.Address)
}

// MessageWorker 简化的消息工作者，面向特定 host 建立 WebSocket 连接并消费消息
type MessageWorker struct {
	host     string
	clientID string
	wsClient *WsClient
	msgChan  chan ComfyUIMessage //websocket client 放入消息， worker 消费消息
	stopCh   chan struct{}       //如果没有存入值，一直取会堵塞
	wg       sync.WaitGroup
	mu       sync.Mutex
	closed   bool
	// 接口注入
	notifier TaskNotifier
}

// NewMessageWorker 创建新的消息工作者
func NewMessageWorker(host string, clientID string, notifier TaskNotifier) *MessageWorker {
	return &MessageWorker{
		host:     host,
		clientID: clientID,
		msgChan:  make(chan ComfyUIMessage, 1000),
		stopCh:   make(chan struct{}),
		notifier: notifier, // ✅ 接口注入
	}
}

// Start 启动消息工作者，建立 WebSocket 连接并开始消费消息
func (w *MessageWorker) Start() error {
	// 创建 WebSocket 客户端
	w.wsClient = NewWsClient(w.host, w.clientID)

	// 启动 WebSocket 客户端
	err := w.wsClient.Start()
	if err != nil {
		return err
	}

	// 启动消息处理协程
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.processMessages()
	}()

	LogMessageWorker("MessageWorker 已启动，连接到: %s, 客户端ID: %s", w.host, w.clientID)
	return nil

}

// Stop 停止消息工作者
func (w *MessageWorker) Stop() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.stopCh)
	w.mu.Unlock()

	// 停止 WebSocket 客户端
	if w.wsClient != nil {
		err := w.wsClient.Stop()
		if err != nil {
			LogMessageWorker("停止 WebSocket 客户端失败: %s", err)
			return err
		}
	}

	// 等待所有协程结束
	w.wg.Wait()
	close(w.msgChan)

	LogMessageWorker("MessageWorker 已停止")
	return nil
}

// Messages 获取消息通道
func (w *MessageWorker) Messages() <-chan ComfyUIMessage {
	return w.msgChan
}

// processMessages 处理从 WebSocket 接收到的消息
func (w *MessageWorker) processMessages() {
	for {
		select {
		case <-w.stopCh:
			return
		case msg, ok := <-w.wsClient.Messages():
			if !ok {
				return
			}
			w.handleMessage(msg)
		}
	}
}

// handleMessage 处理单条消息
func (w *MessageWorker) handleMessage(msg ComfyUIMessage) {
	switch msg.Type {
	case "progress":
		var data ProgressData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleProgress(data)
		}
	case "executing":
		var data ExecutingData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleExecuting(data)
		}
	case "execution_cached":
		var data ExecutionCachedData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleExecutionCached(data)
		}
	case "execution_start":
		var data ExecutionData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleExecutionStart(data)
		}
	case "execution_success":
		var data ExecutionData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleExecutionSuccess(data)
		}
	case "status":
		var data StatusData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleStatus(data)
		}
	case "crystools.monitor":
		var data MonitorData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleMonitor(data)
		}
	case "executed":
		//先打印一下msg.Data
		LogMessageWorker("msg.Data: %s", string(msg.Data))
		var data ExecutedData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleExecuted(data)
		}
	case "progress_state":
		var data ProgressStateData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleProgressState(data)
		}
	case "impact-node-feedback":
		var data ImpactNodeFeedbackData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleImpactNodeFeedback(data)
		}
	case WS_CONNECTED:
		w.handleWSConnected()
	case WS_RECONNECT_ATTEMPT:
		w.handleWSReconnect()
	case WS_RECONNECT_FAILED:
		w.handleWSExit()
	case WS_CONNECTION_ERROR, WS_READ_ERROR, WS_PARSE_ERROR:
		var data SystemData
		if err := json.Unmarshal(msg.Data, &data); err == nil {
			w.handleError(data)
		}
	default:
		LogMessageWorker("未知消息类型: %s, 完整消息: %s", msg.Type, string(msg.Data))
	}
}

// ============================== 消息类型定义 =======================================

// 进度消息
type ProgressData struct {
	Max      int    `json:"max"`       // 总量
	Node     string `json:"node"`      // 节点名称
	PromptID string `json:"prompt_id"` // 任务 ID
	Value    int    `json:"value"`     // 当前进度
}

// 节点执行中
type ExecutingData struct {
	DisplayNode string `json:"display_node"` // 显示名称
	Node        string `json:"node"`         // 节点 ID
	PromptID    string `json:"prompt_id"`
}

// 缓存执行
type ExecutionCachedData struct {
	Nodes     []string `json:"nodes"` // 缓存的节点列表
	PromptID  string   `json:"prompt_id"`
	Timestamp int64    `json:"timestamp"` // 缓存时间戳
}

// 执行开始 / 执行成功
type ExecutionData struct {
	PromptID  string `json:"prompt_id"`
	Timestamp int64  `json:"timestamp"`
}

// 状态消息
type StatusData struct {
	SID    string `json:"sid"`
	Status struct {
		ExecInfo struct {
			QueueRemaining int `json:"queue_remaining"` // 队列剩余任务数
		} `json:"exec_info"`
	} `json:"status"`
}

// 系统消息
type SystemData struct {
	Message string `json:"msg"`
}

// 监控消息
type MonitorData struct {
	CPUUtilization float64 `json:"cpu_utilization"`
	RAMUsed        int64   `json:"ram_used"`
	RAMTotal       int64   `json:"ram_total"`
	RAMUsedPercent float64 `json:"ram_used_percent"`
	GPUs           []struct {
		GPUTemperature  int     `json:"gpu_temperature"`
		VRAMUsed        int64   `json:"vram_used"`
		VRAMTotal       int64   `json:"vram_total"`
		VRAMUsedPercent float64 `json:"vram_used_percent"`
	} `json:"gpus"`
}

// 执行完成消息
type ExecutedData struct {
	Node        string `json:"node"`
	DisplayNode string `json:"display_node"`
	Output      struct {
		// 图片
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
		// 音频
		Audio []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"audios"`
		// 视频
		Videos []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"videos"`
	} `json:"output"`

	PromptID string `json:"prompt_id"`
}

// 进度状态消息
type ProgressStateData struct {
	PromptID string `json:"prompt_id"`
	Nodes    map[string]struct {
		Value         int    `json:"value"`
		Max           int    `json:"max"`
		State         string `json:"state"`
		NodeID        string `json:"node_id"`
		PromptID      string `json:"prompt_id"`
		DisplayNodeID string `json:"display_node_id"`
		ParentNodeID  string `json:"parent_node_id"`
		RealNodeID    string `json:"real_node_id"`
	} `json:"nodes"`
}

// Impact 节点反馈消息
type ImpactNodeFeedbackData struct {
	NodeID     string `json:"node_id"`
	WidgetName string `json:"widget_name"`
	Type       string `json:"type"`
	Value      string `json:"value"`
}

// ============================== 消息处理逻辑 handleProgress  =======================================

func (w *MessageWorker) handleProgress(data ProgressData) {
	LogMessageWorker("[Progress] prompt_id: %s node: %s %d/%d", data.PromptID, data.Node, data.Value, data.Max)
}

func (w *MessageWorker) handleExecuting(data ExecutingData) {
	LogMessageWorker("[Executing] prompt_id: %s node: %s display_node: %s", data.PromptID, data.Node, data.DisplayNode)
}

func (w *MessageWorker) handleExecutionCached(data ExecutionCachedData) {
	LogMessageWorker("[ExecutionCached] prompt_id: %s nodes: %v timestamp: %d", data.PromptID, data.Nodes, data.Timestamp)
}

func (w *MessageWorker) handleExecutionStart(data ExecutionData) {
	LogMessageWorker("[ExecutionStart] prompt_id: %s timestamp: %d", data.PromptID, data.Timestamp)
}

func (w *MessageWorker) handleExecutionSuccess(data ExecutionData) {
	LogMessageWorker("[ExecutionSuccess] prompt_id: %s timestamp: %d", data.PromptID, data.Timestamp)
}

func (w *MessageWorker) handleStatus(data StatusData) {
	LogMessageWorker("[Status] SID: %s 队列剩余: %v", data.SID, data.Status.ExecInfo.QueueRemaining)
}

func (w *MessageWorker) handleWSConnected() {
	LogMessageWorker("[WSConnected] WebSocket 连接成功")
}

func (w *MessageWorker) handleWSReconnect() {
	LogMessageWorker("[WSReconnect] 尝试重连")
}

func (w *MessageWorker) handleWSExit() {
	LogMessageWorker("[WSExit] WebSocket 连接断开")
}

func (w *MessageWorker) handleError(data SystemData) {
	LogMessageWorker("[Error] %s", data.Message)
}

func (w *MessageWorker) handleMonitor(data MonitorData) {
	if len(data.GPUs) > 0 {
		LogMessageWorker("[Monitor] 服务器: %s, CPU: %.1f%%, 内存: %.1f%% (%d/%d MB), GPU: %d°C, VRAM: %.1f%% (%d/%d MB)",
			w.host,
			data.CPUUtilization,
			data.RAMUsedPercent,
			data.RAMUsed/1024/1024,
			data.RAMTotal/1024/1024,
			data.GPUs[0].GPUTemperature,
			data.GPUs[0].VRAMUsedPercent,
			data.GPUs[0].VRAMUsed/1024/1024,
			data.GPUs[0].VRAMTotal/1024/1024)
	} else {
		LogMessageWorker("[Monitor] 服务器: %s, CPU: %.1f%%, 内存: %.1f%% (%d/%d MB)",
			w.host,
			data.CPUUtilization,
			data.RAMUsedPercent,
			data.RAMUsed/1024/1024,
			data.RAMTotal/1024/1024)
	}
}

func (w *MessageWorker) handleExecuted(data ExecutedData) {

	// 任意长度数组
	addresses := make([]model.Address, 0)
	// 1️⃣ 获取图像结果
	if len(data.Output.Images) > 0 {
		//addresses := make([]model.Address, 0, len(data.Output.Images))
		for _, img := range data.Output.Images {
			addresses = append(addresses, model.Address{
				Subfolder: img.Subfolder,
				Filename:  img.Filename,
			})
		}

	}

	// 2️⃣ 获取音频结果
	if len(data.Output.Audio) > 0 {
		//addresses := make([]model.Address, 0, len(data.Output.Audio))
		for _, audio := range data.Output.Audio {
			addresses = append(addresses, model.Address{
				Subfolder: audio.Subfolder,
				Filename:  audio.Filename,
			})
		}
	}

	// 3️⃣ 获取视频结果
	if len(data.Output.Videos) > 0 {
		//addresses := make([]model.Address, 0, len(data.Output.Videos))
		for _, video := range data.Output.Videos {
			addresses = append(addresses, model.Address{
				Subfolder: video.Subfolder,
				Filename:  video.Filename,
			})
		}
	}

	LogMessageWorker("[Executed] prompt_id: %s node: %s 生成结果数量: %d",
		data.PromptID, data.Node, len(addresses))

	// ✅ 通知上层任务完成
	if w.notifier != nil {
		LogMessageWorker("⚡️ 通知上层事务任务完成")
		w.notifier.NotifyTaskDone(data.PromptID, addresses)
	}

}

func (w *MessageWorker) handleProgressState(data ProgressStateData) {
	for nodeID, node := range data.Nodes {
		LogMessageWorker("[ProgressState] prompt_id: %s node: %s 进度: %d/%d 状态: %s",
			data.PromptID, nodeID, node.Value, node.Max, node.State)
	}
}

func (w *MessageWorker) handleImpactNodeFeedback(data ImpactNodeFeedbackData) {
	LogMessageWorker("[ImpactNodeFeedback] node_id: %s widget: %s type: %s value: %s",
		data.NodeID, data.WidgetName, data.Type, data.Value)
}
