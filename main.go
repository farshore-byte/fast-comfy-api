package main

import (
	"fmt"

	"time"

	"farshore.ai/fast-comfy-api/core"
	"farshore.ai/fast-comfy-api/handler"
	"farshore.ai/fast-comfy-api/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	configPath := "./config.yaml"
	// 2️⃣ 加载配置
	config, err := core.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	s3config := config.S3
	serverconfig := config.Server
	port := serverconfig.Port
	// ✅ 创建 handler（内部自动加载并启动所有 API）
	checkInterval := time.Duration(config.HotReload.Interval) * time.Second
	h := handler.NewAPIHandler("./resource/apis", s3config, checkInterval, config.HotReload.Enabled)

	// 设置路由
	r := gin.Default()
	routes.RegisterAPIRoutes(r, h)

	// 启动 HTTP 服务
	r.Run(fmt.Sprintf(":%d", port))
}
