package main

import (
	"fmt"
	"time"

	"farshore.ai/fast-comfy-api/core"
)

func main() {
	// 创建配置管理器，每5秒检查一次配置变化
	configManager := core.NewConfigManager(5 * time.Second)

	// 从目录加载所有API配置
