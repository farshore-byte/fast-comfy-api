package main

import (
	"log"

	"farshore.ai/fast-comfy-api/core"
)

func main() {
	apiPath := "../resource/实时生图api示例.json"
	runtime := core.NewAPIRuntime(apiPath)

	// 1️⃣ 启动服务器（comfyui 会自动连接所有 ComfyUI 节点）
	log.Printf("🚀 启动 API 服务")
	runtime.Start()
	defer runtime.Stop()

	// 2️⃣ 构造测试变量
	// 3️⃣ 使用配置中定义的变量（可以替换其中部分）
	input := map[string]interface{}{
		"wildcard_text":   "woman, portrait, golden hour, cinematic lighting",
		"filename_prefix": "AILab/test_image",
	}
	// 4️⃣  同步生成
	urls, err := runtime.GenerateSync(input)
	if err != nil {
		log.Fatalf("❌ 任务执行失败: %v", err)
	}

	log.Printf("✅ 任务完成! 生成的图片地址: %v\n", urls)
}
