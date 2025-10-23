# Fast ComfyUI API

将 ComfyUI 工作流快速配置成同步调用API的工具，支持热重载，自动上传s3存储，多 API管理

## 🚀 特性

- **同步调用**: 将 ComfyUI 的异步工作流转换为同步 API 调用
- **热重载**: 支持配置文件热重载，修改 API 配置无需重启服务
- **多 API 管理**: 支持同时管理多个 ComfyUI 工作流 API
- **S3 存储**: 自动将生成结果上传到 S3 存储
- **贪婪策略**: api执行器将会选择当前队列最短的comfyui服务器发送任务
- **自动随机种子**: 检测到seed字段，自动生成随机种子
- **支持形式**: 支持音频、视频、图片形式生成，详细配置请参考示例API配置JSON 

## 📋 快速开始

### 1. 安装依赖

```bash
cd fast-comfy-api
go mod tidy
```

### 2. 配置服务

编辑 `config.yaml` 文件：

```yaml
server:
  port: 6004

s3:
  endpoint: "127.0.0.1:9000"      # MinIO 服务地址
  bucket: "fast-comfy-api"        # 桶名
  region: "us-east-1"             # 区域
  access_key: "minioadmin"        # 访问密钥
  secret_key: "minioadmin"        # 密钥
  use_ssl: false                  # 是否使用 SSL
  input_prefix: "input"           # 上传文件前缀
  output_prefix: "output"         # 输出文件前缀

hot_reload:
  enabled: true                   # 推荐启用热重载
  interval: 10                    # 检查间隔（秒）
```

### 3. 配置 API 工作流

在 `resource/apis/` 目录下有图片、音频、视频三个创建配置示例。你可以更换comfyui_nodes字段为自己的comfyui服务器进行测试，或者示例创建 自定义 配置文件。

📖 **详细配置说明**: 请参考 [API配置编写说明.md](./resource/apis/API配置编写说明.md)


### 4. 启动服务

```bash
go run main.go
```
or 
```bash
go build fast-comfy-api main.go
./fast-comfy-api
```


### 同步请求
- 图片生成
```bash
curl --location --request POST 'http://localhost:6004/api/generate_sync' \
--header 'Content-Type: application/json' \
--data-raw '{
    "token":"sk-23423546543w256hhj66",
    "vars":{
      "wildcard_text":"woman, with long dark brown hair, wearing a white baseball cap with black embroidered logo, light beige walls, wearing a light pink cardigan draped over her shoulders, subtle necklace visible around her neck, clean and modern background with no distractions, self-phone-photo, holding phone,"
    }
  }'
```

- 音频生成
```bash
curl --location --request POST 'http://localhost:6004/api/generate_sync' \
--header 'Content-Type: application/json' \
--data-raw '{
    "token":"sk-43545356788884",
    "vars":{
      "prompt":"An ancient bell flew out of the crack and landed on the ground with a clang."
    }
  }'
```

- 视频保存
```bash
curl --location --request POST 'http://localhost:6004/api/generate_sync' \
--header 'Content-Type: application/json' \
--data-raw '{
    "token":"sk-23435653245666",
    "vars":{
      "filename_prefix":"AILab/video"
    }
  }'
```

## 🔧 API 配置说明

## 存放地址
```
resource/apis/
```

### 配置文件结构

每个 API 配置文件包含以下字段：

- `name`: API 名称（用于显示）
- `token`: API 标识符（鉴权）
- `comfyui_url`: ComfyUI 服务地址
- `workflow`: ComfyUI 工作流 JSON
- `variables`: 可配置变量映射

### 变量配置

在 `variables` 字段中定义可配置参数：

```json
"variables": {
  "prompt": {
    "type": "string",
    "default": "a beautiful landscape",
    "description": "生成提示词"
  },
  "steps": {
    "type": "number", 
    "default": 20,
    "description": "生成步数"
  }
}
```

## 📡 API 接口

### 同步生成

```http
POST /api/generate_sync
Content-Type: application/json

{
  "token": "video_generation",
  "vars": {
    "prompt": "a beautiful sunset",
    "steps": 25
  }
}
```

响应：
```json
{
  "code": 0,
  "msg": "success",
  "data": [
    "https://your-s3-bucket/output/prompt_id/filename.png"
  ]
}
```

### 列出所有 API

```http
GET /api/list
```

响应：
```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "token": "sk-23435653245666",
      "name": "视频保存示例",
      "status": "running",
      "msg": "API运行中"
    }
  ]
}
```

### 启动指定 API

```http
POST /api/start/{token}
```

响应：
```json
{
  "code": 0,
  "msg": "success",
  "data": "API [sk-23435653245666] started"
}
```

### 停止指定 API

```http
POST /api/stop/{token}
```

响应：
```json
{
  "code": 0,
  "msg": "success",
  "data": "API [sk-23435653245666] stopped"
}
```

## 🔄 热重载功能

### 启用热重载

在 `config.yaml` 中设置：

```yaml
hot_reload:
  enabled: true
  interval: 10
```

### 热重载特性

- **新增文件**: 自动检测并加载新的 API 配置文件
- **文件修改**: 自动重新加载修改的配置文件
- **文件删除**: 自动停止并移除已删除的 API
- **无需重启**: 所有配置变更无需重启服务

## 🎨 日志系统

系统使用彩色日志输出，便于区分不同组件：

- **🔵 API Runtime**: 蓝色 - ComfyUI API 运行时日志
- **🟢 Message Worker**: 绿色 - 消息处理工作器日志  
- **🟡 Hot Reload**: 黄色 - 热重载监控日志
- **🔴 Error**: 红色 - 错误日志

## 📁 项目结构

```
fast-comfy-api/
├── main.go                 # 应用入口
├── config.yaml            # 配置文件
├── core/                  # 核心组件
│   ├── api_manager.go     # API 管理器（含热重载）
│   ├── api_runtime.go     # API 运行时
│   ├── message_worker.go  # 消息处理器
│   └── logger.go          # 日志系统
├── handler/               # HTTP 处理器
├── model/                 # 数据模型
├── routes/                # 路由定义
├── resource/              # 资源文件
│   └── apis/              # API 配置文件
└── test/                  # 测试文件
```

## 🔍 调试技巧

### 查看 API 状态

```bash
curl http://localhost:6004/api/list
```

### 测试热重载

1. 修改 `resource/apis/` 中的配置文件
2. 观察控制台日志，确认热重载生效
3. 无需重启服务即可应用变更

### 日志级别

系统会自动输出详细的调试信息，包括：

- API 加载状态
- 热重载检测结果
- 生成任务进度
- 错误信息

## 🚨 注意事项

1. **确保 ComfyUI 服务运行**：API 需要连接到运行的 ComfyUI 实例
2. **S3 存储配置**：确保 S3/MinIO 服务可访问
3. **文件权限**：确保应用有权限读写临时文件
4. **网络连接**：确保能访问 ComfyUI 和 S3 服务

## 📄 许可证

MIT License

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！
