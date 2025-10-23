package main

import (
	"fmt"
	"io/ioutil"
	"log"

	"farshore.ai/fast-comfy-api/core"
)

func main() {
	// 1️⃣ 读取 JSON 文件
	jsonFile, err := ioutil.ReadFile("/Users/farshore/fast-comfyui-api/fast-comfy-api/resource/实时生图api示例.json")
	if err != nil {
		log.Fatalf("❌ 读取 JSON 文件失败: %v", err)
	}

	// 2️⃣ 初始化 API Parser
	apiParser, err := core.NewAPIParser(jsonFile)
	if err != nil {
		log.Fatalf("❌ 解析 API 失败: %v", err)
	}

	// 3️⃣ 打印基本信息（全部通过封装方法访问）
	fmt.Println("✅ API 名称:", apiParser.GetName())
	fmt.Println("📜 描述:", apiParser.GetDescription())
	fmt.Println("🪙 Token:", apiParser.GetToken())
	fmt.Println("🖥️ 节点服务器:", apiParser.GetComfyuiNodes())
	fmt.Println("📦 变量名:", apiParser.GetVariableNames())

	// 4️⃣ 示例：应用变量替换
	newPrompt, err := apiParser.ApplyVariables(map[string]interface{}{
		"wildcard_text": "MyCustomVAE.safetensors",
	})
	if err != nil {
		log.Fatalf("❌ 替换变量失败: %v", err)
	}

	// 5️⃣ 打印替换后的结果
	fmt.Println("🔀 变量替换后的结果:")
	fmt.Println("  - Prompt:")
	for id, node := range newPrompt {
		fmt.Printf("    - %s: %v\n", id, node.Inputs)
	}
	fmt.Println("✅ 执行完成！")
}
