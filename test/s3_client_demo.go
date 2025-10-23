package main

import (
	"context"
	"farshore.ai/fast-comfy-api/core"
	"fmt"
)

func main() {
	// 1️⃣ 配置文件路径
	configPath := "./config.yaml"

	// 2️⃣ 加载配置
	config, err := core.LoadConfig(configPath)
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 3️⃣ 创建 S3 客户端
	s3Client, err := core.NewS3Client(config.S3)
	if err != nil {
		panic(fmt.Sprintf("创建 S3Client 失败: %v", err))
	}

	// 4️⃣ 上传示例文件到 output 目录
	ctx := context.Background()
	fileID := "4323543244"
	filePath := "./static/test.jpeg"

	url, err := s3Client.UploadOutputFile(ctx, fileID, filePath)
	if err != nil {
		panic(fmt.Sprintf("上传文件失败: %v", err))
	}

	// 5️⃣ 打印公有访问链接
	fmt.Println("✅ 文件上传成功！访问链接:")
	fmt.Println(url)
}
