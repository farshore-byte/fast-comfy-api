// comfyui 任务提交 测试函数
package main

import (
	"farshore.ai/fast-comfy-api/core"
	"fmt"
)

func main() {
	host := "https://u459706-9067-0fe83346.westx.seetacloud.com:8443"
	prompt := map[string]interface{}{
		"2": map[string]interface{}{
			"inputs": map[string]interface{}{
				"value": "你好",
			},
			"class_type": "PrimitiveString",
			"_meta": map[string]interface{}{
				"title": "String",
			},
		},
		"3": map[string]interface{}{
			"inputs": map[string]interface{}{
				"text":  []interface{}{"2", 0},
				"text2": "你好",
			},
			"class_type": "ShowText|pysssss",
			"_meta": map[string]interface{}{
				"title": "Show Text 🐍",
			},
		},
	}

	res, err := core.PromptCommit(host, prompt, "1234567890")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(res)

}
