package main

import (
	"fmt"

	"time"

	"farshore.ai/fast-comfy-api/config"
	"farshore.ai/fast-comfy-api/handler"
	"farshore.ai/fast-comfy-api/routes"
	"farshore.ai/fast-comfy-api/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	configPath := "./config.yaml"
	//2️⃣ 加载配置
	config, err := config.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	// 3️⃣ 读取配置
	s3config := config.S3
	serverconfig := config.Server
	port := serverconfig.Port

	// 获取 feishu webhook 相关配置
	feishu_webhook := config.Feishu.WebHook

	// 初始化 feishu webhook
	if feishu_webhook != "" {
		// 初始化
		utils.InitFeishuClient(feishu_webhook)
	}

	// ✅ 创建 handler（内部自动加载并启动所有 API）
	checkInterval := time.Duration(config.HotReload.Interval) * time.Second
	h := handler.NewAPIHandler("./resource/apis", s3config, checkInterval, config.HotReload.Enabled)

	// 设置路由
	r := gin.Default()
	routes.RegisterAPIRoutes(r, h)

	// 启动 HTTP 服务
	r.Run(fmt.Sprintf(":%d", port))
}
