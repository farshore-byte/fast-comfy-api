# API 配置编写说明

本文档详细说明如何编写 Fast ComfyUI API 的配置文件。

## 📋 配置文件结构

每个 API 配置文件是一个 JSON 文件，包含以下主要字段：

```json
{
  "name": "API 名称",
  "description": "API 描述",
  "token": "API 标识符",
  "comfyui_nodes": [
    "ComfyUI 服务器地址"
  ],
  "prompt": {
    // ComfyUI 工作流 JSON
  },
  "variables": {
    // 可配置变量映射
  }
}
```

## 🔧 字段说明

### 1. 基本信息字段

- **name** (string): API 名称，用于显示和识别
- **description** (string): API 详细描述
- **token** (string): API 标识符，用于调用时的鉴权

### 2. ComfyUI 配置

- **comfyui_nodes** (array): ComfyUI 服务器地址列表，支持多个服务器实现负载均衡
- **prompt** (object): ComfyUI 工作流 JSON 配置

### 3. 变量配置

- **variables** (object): 可配置变量映射，定义用户可以传入的参数

## 📝 配置示例

### 视频保存示例

```json
{
  "name": "视频保存示例",
  "description": "视频保存示例",
  "prompt": {
    "1": {
      "inputs": {
        "video": "#Explore #reels #reelsinstagram #instalike #instadaily.mp4",
        "force_rate": 0,
        "custom_width": 0,
        "custom_height": 0,
        "frame_load_cap": 0,
        "skip_first_frames": 0,
        "select_every_nth": 1,
        "format": "AnimateDiff"
      },
      "class_type": "VHS_LoadVideo",
      "_meta": {
        "title": "Load Video (Upload) 🎥🅥🅗🅢"
      }
    },
    "2": {
      "inputs": {
        "fps": 30,
        "images": [
          "1",
          0
        ]
      },
      "class_type": "CreateVideo",
      "_meta": {
        "title": "创建视频"
      }
    },
    "3": {
      "inputs": {
        "filename_prefix": "video/ComfyU",
        "format": "auto",
        "codec": "auto",
        "video-preview": "",
        "video": [
          "2",
          0
        ]
      },
      "class_type": "SaveVideo",
      "_meta": {
        "title": "保存视频"
      }
    }
  },
  "comfyui_nodes": [
    "http://localhost:8000"
  ],
  "variables": {
    "filename_prefix": {
      "path": "3.inputs.filename_prefix",
      "type": "string",
      "default": "video/ComfyU"
    }
  },
  "token": "sk-23435653245666"
}
```

## 🔍 变量配置详解

### 变量结构

```json
"变量名": {
  "path": "节点路径",
  "type": "数据类型",
  "default": "默认值",
  "description": "变量描述（可选）"
}
```

### 路径格式

路径格式为：`节点编号.字段名.子字段名`

示例：
- `"3.inputs.filename_prefix"` - 节点3的inputs中的filename_prefix字段
- `"5.inputs.seed"` - 节点5的inputs中的seed字段

