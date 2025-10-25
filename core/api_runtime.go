package core

import (
	"encoding/json"
	"farshore.ai/fast-comfy-api/config"
	"farshore.ai/fast-comfy-api/model"
	"farshore.ai/fast-comfy-api/utils"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

/*

API 运行时

start : 启动 API 服务
stop : 停止 API 服务
restart : 重启 API 服务

generate_sync : 同步生成接口
*/

type APIRuntime struct {
	//1. API 服务配置
	apiparser *APIParser
	//2. API 服务状态, 在线/离线/异常
	status string
	// 消息：启动成功/ 部分节点启动失败 进行消息提醒，便于及时排查问题
	msg string

	// ***************
	workerlist []*MessageWorker // 存放所有服务器的websocket消息消费者

	waiting sync.Map // 存放等待通知的任务
}

// ✅ 实现 TaskNotifier 接口
func (api *APIRuntime) NotifyTaskDone(promptID string, addresses []model.Address) {
	LogAPIRuntime("🚄 [NotifyTaskDone] 任务完成，prompt_id=%s, 地址列表=%s", promptID, addresses)
	if ch, ok := api.waiting.Load(promptID); ok {
		ch.(chan []model.Address) <- addresses
		api.waiting.Delete(promptID)
	}
}

// 初始化 API 运行时
func NewAPIRuntime(apijson_path string) *APIRuntime {
	// 读取json 文件
	apijson, err := ioutil.ReadFile(apijson_path)
	if err != nil {
		LogAPIRuntime(ColorRed+"读取 API JSON 文件失败: %s", err)
		return nil
	}
	// 解析 API 配置
	apiparser, err := NewAPIParser(apijson)
	if err != nil {
		LogAPIRuntime(ColorRed+"解析 API 配置失败: %s", err)
		return nil
	}
	// 初始化 API 运行时，状态为离线
	return &APIRuntime{
		apiparser: apiparser,
		status:    "offline",
	}
}

// 启动 API 服务
func (api *APIRuntime) Start() {
	// 1.获取api的comfyui nodes
	comfyui_nodes := api.apiparser.GetComfyuiNodes()
	// 获取token
	token := api.apiparser.GetToken()
	// comfyui_nodes 视为 hosts, token 视为 clientID
	clientID := token
	// 为所有服务器建立连接
	for _, node := range comfyui_nodes {
		host := normalizeHost(node)
		// 创建websocket消息消费者,注入APi 作为 notifier
		messageworker := NewMessageWorker(host, clientID, api)
		// 存入 workerlist
		api.workerlist = append(api.workerlist, messageworker)
		// 启动消息消费者
		err := messageworker.Start()
		if err != nil {
			LogAPIRuntime(ColorRed+"启动消息消费者失败: %s", err)
			// 记录错误信息
			api.msg = fmt.Sprintf(ColorRed+"%s 节点启动失败，请检查失败节点或者将其移除后重试 %s", host, err)
			api.status = "exception"
			// 清空 workerlist
			api.workerlist = []*MessageWorker{}
			return

		}

	}
	// 2. 启动成功，状态为在线
	api.status = "online"
	api.msg = "API 服务已启动"
}

// 辅助函数，解析 host 节点，返回 host:port 格式
func normalizeHost(node string) string {
	// 去掉协议前缀
	node = strings.TrimPrefix(node, "https://")
	node = strings.TrimPrefix(node, "http://")

	// 解析 URL 防止路径干扰
	u, err := url.Parse(node)
	if err != nil {
		return node
	}

	// 返回 host:port
	if u.Host != "" {
		return u.Host
	}
	return node
}

// 停止 API 服务
func (api *APIRuntime) Stop() {
	// 1. 关闭所有 websocket 连接
	for _, worker := range api.workerlist {
		worker.Stop()
	}
	// 2. 状态为离线
	api.status = "offline"
	api.msg = "API 服务已停止"
}

// 重启 API 服务
func (api *APIRuntime) Restart() {

	// 1. 停止 API 服务
	api.Stop()
	// 2. 启动 API 服务
	api.Start()
}

// 获取API 名字
func (api *APIRuntime) GetName() string {
	if api.apiparser == nil {
		return ""
	}
	return api.apiparser.GetName()
}

// 获取API token
func (api *APIRuntime) GetToken() string {
	if api.apiparser == nil {
		return ""
	}
	return api.apiparser.GetToken()
}

// 获取当前状态
func (api *APIRuntime) GetStatus() string {
	return api.status
}

// 获取当前状态信息
func (api *APIRuntime) GetMessage() string {
	return api.msg
}

// 辅助函数 GetBestHost 选取一个最佳 host 节点

// 🫱🫱🫱 贪婪策略，任务抵达时，检查所有nodes的服务器的队列数量，选择最小队列数量的node 作为最佳节点

func (api *APIRuntime) GetBestServer() string {
	// step 1️⃣ 获取所有节点的服务器列表
	nodes := api.apiparser.GetComfyuiNodes()
	if len(nodes) == 0 {
		LogAPIRuntime("[GetBestServer] 没有节点")
		return "" // 没有节点就返回空
	}
	// 如果只有一个节点，直接返回
	if len(nodes) == 1 {
		LogAPIRuntime("[GetBestServer] 只有一个节点，直接返回")
		return nodes[0]
	}
	// 2️⃣ 遍历所有节点，获取服务器的当前队列数量，使用 go 并发获取
	queue_map := make(map[string]int)
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			queue_remaining, err := api.GetComfyuiServerQueue(node)
			if err != nil {
				LogAPIRuntime(ColorYellow+"[GetBestServer] 获取服务器队列数量失败: %s", err)
				return
			}
			queue_map[node] = queue_remaining
		}(node)
	}
	wg.Wait()
	// 3️⃣ 选取最小队列数量的节点作为最佳节点
	min_queue := int(^uint(0) >> 1) // 最大值
	best_node := ""
	for node, queue := range queue_map {
		if queue < min_queue {
			min_queue = queue
			best_node = node
		}
	}
	// 4️⃣ 打印日志
	LogAPIRuntime(ColorGreen+"[GetBestServer] 选取节点: %s, 队列数量: %d", best_node, min_queue)
	return best_node
}

// 辅助函数 获取服务器的当前队列数量
func (api *APIRuntime) GetComfyuiServerQueue(host string) (int, error) {
	// 请求 路由 get /prompt return {"exec_info":{"queue_remaining": 0}}
	// 构造请求地址
	url := fmt.Sprintf("%s/prompt", host)
	// 发起请求
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	// 解析响应
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	// 解析响应数据
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		// 服务器请求失败，说明通信有问题或者服务异常，返回 理论上int最大值，防止该node 被选中
		LogAPIRuntime(ColorRed+"[GetComfyuiServerQueue] 服务器请求失败: %s", err)
		return int(^uint(0) >> 1), nil
	}
	// 获取队列数量
	queue_remaining := data["exec_info"].(map[string]interface{})["queue_remaining"].(float64)
	//log.Printf("[GetComfyuiServerQueue] 获取到任务，检测 %s 服务器队列数量: %d", host, int(queue_remaining))
	// ⚠️⚠️⚠️⚠️⚠️  插入一个队列监控报警
	if queue_remaining >= config.WarningQueueSize {
		// 调用feishuclient 发送报警
		warn_log := fmt.Sprintf(" %s 服务器队列数量: %d, 超过预设值: %d", host, int(queue_remaining), config.WarningQueueSize)
		utils.Feishu.InternalFeishuWarning("queue_warning", host, warn_log)
	}
	return int(queue_remaining), nil
}

/*
func (api *APIRuntime) GetBestServer() string {
	nodes := api.apiparser.GetComfyuiNodes()
	if len(nodes) == 0 {
		return "" // 没有节点就返回空
	}

	// 初始化随机种子
	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(nodes))

	// 打印日志
	log.Printf("[GetBestServer]  选取节点: %s", nodes[index])
	return nodes[index]
}

*/

// 同步生成接口，输入变量json, 返回资源url list, error

func (api *APIRuntime) GenerateSync(vars map[string]interface{}) ([]string, string, error) {
	// 1️⃣ 获取变量替换后的 prompt
	prompt_node, err := api.apiparser.ApplyVariables(vars)
	if err != nil {
		LogAPIRuntime("变量替换失败: %s", err)
		return nil, "", err
	}

	// 2️⃣ 随机向一个节点提交任务，并获取 prompt_id
	target_server := api.GetBestServer()
	if target_server == "" {
		LogAPIRuntime("没有可用的节点")
		return nil, "", fmt.Errorf("没有可用的节点")
	}

	ClientID := api.apiparser.GetToken()
	prompt_id, err := PromptCommit(target_server, prompt_node, ClientID)
	if err != nil {
		LogAPIRuntime("提交任务失败: %s", err)
		return nil, prompt_id, err // ✅ prompt_id 已经生成，返回给调用方
	}

	// 3️⃣ 注册等待 channel
	ch := make(chan []model.Address, 1)
	api.waiting.Store(prompt_id, ch)

	// 4️⃣ 等待 NotifyTaskDone 回调写入结果
	select {
	case addresses := <-ch:
		LogAPIRuntime(ColorYellow+"[GenerateSync] 任务完成，获取地址列表,prompt_id=%s", prompt_id)
		urls_from_comfyui := address2urls(addresses, target_server)
		return urls_from_comfyui, prompt_id, nil // ✅ 返回 prompt_id
	case <-time.After(time.Second * 60):
		LogAPIRuntime("[GenerateSync] 等待超时，任务结果未收到")
		return nil, prompt_id, fmt.Errorf("等待超时，任务结果未收到")
	}
}

// 辅助函数 address2urls 将地址转换为url,仅支持输出文件
func address2urls(addresses []model.Address, host string) []string {
	urls := make([]string, len(addresses))
	for i, address := range addresses {
		url := fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=output", host, address.Filename, address.Subfolder)
		urls[i] = url
	}
	return urls
}
