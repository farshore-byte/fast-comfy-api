package core

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"farshore.ai/fast-comfy-api/model"
)

// APIManager 管理多个api，绑定统一s3资源桶，并提供统一接口，通过token路由到指定api generate_sync()方法生成
type APIManager struct {
	apis          map[string]*APIRuntime // token -> APIRuntime实例
	configFiles   map[string]string      // token -> 配置文件路径
	fileModTimes  map[string]time.Time   // 文件路径 -> 最后修改时间
	s3client      *S3Client
	resourceDir   string // 资源目录路径
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	checkInterval time.Duration
}

func NewAPIManager(resource_dir string, s3config model.S3Config, checkInterval time.Duration, enabled bool) *APIManager {
	s3client, err := NewS3Client(s3config) // 创建s3客户端
	if err != nil {
		panic(err)
	}
	api_manager := &APIManager{
		apis:          make(map[string]*APIRuntime), // ✅ 改成 *APIRuntime
		configFiles:   make(map[string]string),
		fileModTimes:  make(map[string]time.Time),
		s3client:      s3client,
		resourceDir:   resource_dir, // 记录资源目录
		stopCh:        make(chan struct{}),
		checkInterval: checkInterval,
	}
	api_manager.loadAPIs(resource_dir) // 加载api配置文件
	// 🚀 策略：启动所有 API
	api_manager.StartAll()
	// 🚀 策略：根据配置启动热重载监控
	if enabled {
		api_manager.StartHotReload()
	}
	return api_manager
}

// 加载api配置文件，遍历每个api的配置文件
func (api_manager *APIManager) loadAPIs(resource_dir string) {
	files, err := ioutil.ReadDir(resource_dir)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		api_config_file := filepath.Join(resource_dir, file.Name())
		apiruntime := NewAPIRuntime(api_config_file) // 创建APIRuntime实例
		// 启动apiruntime
		go apiruntime.Start()
		api_token := apiruntime.GetToken()
		api_manager.apis[api_token] = apiruntime // ✅ 存入指针
		api_manager.configFiles[api_token] = api_config_file

		// 记录文件修改时间
		fileInfo, err := os.Stat(api_config_file)
		if err == nil {
			api_manager.fileModTimes[api_config_file] = fileInfo.ModTime()
		}
	}

	LogAPIRuntime("✅ 加载 %d 个API配置", len(api_manager.apis))
	for token, apiruntime := range api_manager.apis {
		LogAPIRuntime("API Token: %s, 名称: %s", token, apiruntime.GetName())
	}
}

// ---------------------------------- 控制逻辑 --------------------------------
// 启动所有 API
func (m *APIManager) StartAll() {
	for _, api := range m.apis {
		go api.Start()
	}
}

// 停止所有 API
func (m *APIManager) StopAll() {
	for _, api := range m.apis {
		api.Stop()
	}
}

// 重启所有 API
func (m *APIManager) RestartAll() {
	for _, api := range m.apis {
		api.Restart()
	}
}

// 启动单个 API
func (m *APIManager) StartAPI(token string) error {
	api, ok := m.apis[token]
	if !ok {
		return fmt.Errorf("token %s not found", token)
	}
	go api.Start()
	return nil
}

// 停止单个 API
func (m *APIManager) StopAPI(token string) error {
	api, ok := m.apis[token]
	if !ok {
		return fmt.Errorf("token %s not found", token)
	}
	api.Stop()
	return nil
}

// 获取状态列表
func (m *APIManager) ListAPIs() []map[string]string {
	list := []map[string]string{}
	for token, api := range m.apis {
		list = append(list, map[string]string{
			"token":  token,
			"name":   api.GetName(),
			"status": api.GetStatus(),
			"msg":    api.GetMessage(),
		})
	}
	return list
}

// ---------------------------------- 热重载功能 --------------------------------
// StartHotReload 启动热重载监控
func (m *APIManager) StartHotReload() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.hotReloadLoop()
	}()
	LogAPIRuntime("🔥 热重载监控已启动，检查间隔: %v", m.checkInterval)
}

// StopHotReload 停止热重载监控
func (m *APIManager) StopHotReload() {
	close(m.stopCh)
	m.wg.Wait()
	LogAPIRuntime("🛑 热重载监控已停止")
}

// hotReloadLoop 热重载循环
func (m *APIManager) hotReloadLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkConfigChanges()
		}
	}
}

// checkConfigChanges 检查配置变化
func (m *APIManager) checkConfigChanges() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 获取当前目录下的所有配置文件
	currentFiles := make(map[string]string)

	files, err := ioutil.ReadDir(m.resourceDir)
	if err != nil {
		LogAPIRuntime("❌ 读取目录失败: %s, 错误: %s", m.resourceDir, err)
		return
	}

	// 扫描当前目录中的配置文件
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		configPath := filepath.Join(m.resourceDir, file.Name())
		currentFiles[configPath] = configPath
	}

	// 检查新增的文件
	for configPath := range currentFiles {
		if _, exists := m.fileModTimes[configPath]; !exists {
			LogAPIRuntime("🆕 检测到新增配置文件: %s", configPath)
			m.addNewAPI(configPath)
		}
	}

	// 检查删除的文件
	for configPath := range m.fileModTimes {
		if _, exists := currentFiles[configPath]; !exists {
			LogAPIRuntime("🗑️ 检测到删除配置文件: %s", configPath)
			m.removeDeletedAPI(configPath)
		}
	}

	// 检查现有文件的修改
	for token, configPath := range m.configFiles {
		fileInfo, err := os.Stat(configPath)
		if err != nil {
			LogAPIRuntime("❌ 检查配置文件失败: %s, 错误: %s", configPath, err)
			continue
		}

		modTime := fileInfo.ModTime()
		lastModTime, exists := m.fileModTimes[configPath]

		if exists && modTime.After(lastModTime) {
			LogAPIRuntime("🔄 检测到配置文件变化: %s", configPath)
			m.reloadAPI(token, configPath)
			m.fileModTimes[configPath] = modTime
		}
	}
}

// addNewAPI 添加新的API配置
func (m *APIManager) addNewAPI(configPath string) {
	newAPI := NewAPIRuntime(configPath)
	if newAPI == nil {
		LogAPIRuntime("❌ 创建新APIRuntime失败: %s", configPath)
		return
	}

	token := newAPI.GetToken()

	// 检查是否已存在同名token
	if _, exists := m.apis[token]; exists {
		LogAPIRuntime("⚠️ API Token已存在，跳过添加: %s", token)
		return
	}

	m.apis[token] = newAPI
	m.configFiles[token] = configPath

	// 记录文件修改时间
	fileInfo, err := os.Stat(configPath)
	if err == nil {
		m.fileModTimes[configPath] = fileInfo.ModTime()
	}

	go newAPI.Start()
	LogAPIRuntime("✅ 添加新API配置: %s", token)
}

// removeDeletedAPI 移除已删除的API配置
func (m *APIManager) removeDeletedAPI(configPath string) {
	// 找到对应的token
	var targetToken string
	for token, path := range m.configFiles {
		if path == configPath {
			targetToken = token
			break
		}
	}

	if targetToken == "" {
		return
	}

	// 停止并移除API
	if api, exists := m.apis[targetToken]; exists {
		api.Stop()
		delete(m.apis, targetToken)
		delete(m.configFiles, targetToken)
		delete(m.fileModTimes, configPath)
		LogAPIRuntime("🗑️ 移除已删除的API配置: %s", targetToken)
	}
}

// reloadAPI 重新加载API配置
func (m *APIManager) reloadAPI(token string, configPath string) {
	// 停止旧的API服务
	if oldAPI, exists := m.apis[token]; exists {
		oldAPI.Stop()
		LogAPIRuntime("🛑 停止旧API服务: %s", token)
	}

	// 加载新的配置
	newAPI := NewAPIRuntime(configPath)
	if newAPI == nil {
		LogAPIRuntime("❌ 创建新APIRuntime失败: %s", token)
		return
	}

	// 启动新的API服务
	go newAPI.Start()

	// 更新管理器
	m.apis[token] = newAPI
	LogAPIRuntime("✅ 重新加载API配置成功: %s", token)
}

// AddAPI 动态添加新的API配置
func (m *APIManager) AddAPI(configPath string) error {
	newAPI := NewAPIRuntime(configPath)
	if newAPI == nil {
		return fmt.Errorf("创建APIRuntime失败")
	}

	token := newAPI.GetToken()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.apis[token] = newAPI
	m.configFiles[token] = configPath

	// 记录文件修改时间
	fileInfo, err := os.Stat(configPath)
	if err == nil {
		m.fileModTimes[configPath] = fileInfo.ModTime()
	}

	go newAPI.Start()
	LogAPIRuntime("✅ 动态添加API配置: %s", token)
	return nil
}

// RemoveAPI 移除API配置
func (m *APIManager) RemoveAPI(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if api, exists := m.apis[token]; exists {
		api.Stop()
		delete(m.apis, token)
		delete(m.configFiles, token)
		LogAPIRuntime("🗑️ 移除API配置: %s", token)
	}
}

// ReloadAllAPIs 重新加载所有API配置（手动触发）
func (m *APIManager) ReloadAllAPIs(resourceDir string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止所有现有API
	for token, api := range m.apis {
		api.Stop()
		LogAPIRuntime("🛑 停止API服务: %s", token)
	}

	// 清空现有数据
	m.apis = make(map[string]*APIRuntime)
	m.configFiles = make(map[string]string)
	m.fileModTimes = make(map[string]time.Time)

	// 重新加载所有API配置
	files, err := ioutil.ReadDir(resourceDir)
	if err != nil {
		LogAPIRuntime("❌ 重新加载目录失败: %s, 错误: %s", resourceDir, err)
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		api_config_file := filepath.Join(resourceDir, file.Name())
		apiruntime := NewAPIRuntime(api_config_file)
		if apiruntime == nil {
			LogAPIRuntime("❌ 创建APIRuntime失败: %s", api_config_file)
			continue
		}

		go apiruntime.Start()
		api_token := apiruntime.GetToken()
		m.apis[api_token] = apiruntime
		m.configFiles[api_token] = api_config_file

		// 记录文件修改时间
		fileInfo, err := os.Stat(api_config_file)
		if err == nil {
			m.fileModTimes[api_config_file] = fileInfo.ModTime()
		}
	}

	LogAPIRuntime("🔄 重新加载完成，当前 %d 个API配置", len(m.apis))
}

// --------------------------------- 生成逻辑 ------------------------------------------
// GenerateSync 调用对应 API 的同步生成逻辑，并上传结果到 S3
func (api_manager *APIManager) GenerateSync(api_token string, vars map[string]interface{}) ([]string, error) {
	apiruntime, ok := api_manager.apis[api_token]
	if !ok {
		return nil, fmt.Errorf("api token %s not found", api_token)
	}

	comfyui_urls, prompt_id, err := apiruntime.GenerateSync(vars)
	if err != nil {
		return nil, fmt.Errorf("任务提交失败: %w", err)
	}

	s3_urls := make([]string, 0, len(comfyui_urls)) // ✅ 不要预填充

	for _, comfyui_url := range comfyui_urls {
		// 下载 ComfyUI 生成的文件
		local_file, err := downloadFile(comfyui_url, prompt_id)
		if err != nil {
			return nil, fmt.Errorf("下载失败: %w", err)
		}

		// 上传到 S3（假设 UploadOutputFile(ctx, prefix, localPath)）
		s3_url, err := api_manager.s3client.UploadOutputFile(context.Background(), prompt_id, local_file)
		if err != nil {
			_ = deleteFile(local_file)
			return nil, fmt.Errorf("上传失败: %w", err)
		}

		s3_urls = append(s3_urls, s3_url)

		// 删除本地文件
		if err := deleteFile(local_file); err != nil {
			fmt.Printf("⚠️ 删除本地文件失败: %v\n", err)
		}
	}

	return s3_urls, nil
}

// 辅助函数

func get_filename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// 解析失败就退回到直接取 path base
		return filepath.Base(rawURL)
	}

	// 优先从 query 参数里取 filename
	q := u.Query().Get("filename")
	if q != "" {
		return q
	}

	// 否则取 URL path 最后部分
	return filepath.Base(u.Path)
}

// 下载 comfyui 输出文件到本地 ./tmp/{prefix}/
func downloadFile(url string, prefix string) (string, error) {
	log.Printf("⏬ 任务结束, 正在下载结果文件....")
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	local_dir := filepath.Join("./tmp", prefix)
	if err := os.MkdirAll(local_dir, os.ModePerm); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	local_file := filepath.Join(local_dir, get_filename(url))
	f, err := os.Create(local_file)
	if err != nil {
		return "", fmt.Errorf("创建文件失败: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}
	log.Printf("✅ 下载完成: %s", local_file)

	return local_file, nil
}

// 删除本地文件及其 tmp 下的所有路径，默认所有临时文件下载到/tmp下
func deleteFile(localFile string) error {
	// 确保路径存在
	if _, err := os.Stat(localFile); os.IsNotExist(err) {
		return nil
	}

	// 找到 tmp 根目录之后的部分
	tmpRoot := "tmp"
	absPath, err := filepath.Abs(localFile)
	if err != nil {
		return err
	}

	// 找出 tmp 开始的位置
	idx := strings.Index(absPath, string(filepath.Separator)+tmpRoot+string(filepath.Separator))
	if idx == -1 {
		// 不在 tmp 路径下，只删除文件本身
		return os.Remove(localFile)
	}

	// 构造出 tmp 子目录路径
	tmpPath := absPath[idx+1:] // 去掉前导 /
	tmpDir := filepath.Dir(tmpPath)

	// 拼出完整 tmp 目录路径
	fullTmpDir := filepath.Join(".", tmpDir)

	// 删除整个 tmp 子目录
	return os.RemoveAll(fullTmpDir)
}
