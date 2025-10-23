package model

// PromptNode 表示 ComfyUI 的一个节点
type PromptNode struct {
	Inputs    map[string]interface{} `json:"inputs"`     // 节点输入参数
	ClassType string                 `json:"class_type"` // 节点类型
	Meta      NodeMeta               `json:"_meta"`      // 节点元信息
}

// NodeMeta 是节点的元数据
type NodeMeta struct {
	Title string `json:"title"` // 节点标题
}

// Variable 定义一个可替换的变量（带类型、默认值、路径）
type Variable struct {
	Path    string      `json:"path"`    // 变量对应 prompt 中的路径，如 "757.inputs.wildcard_text"
	Type    string      `json:"type"`    // 变量类型，例如 "string"、"number"、"bool" 等
	Default interface{} `json:"default"` // 默认值
}

// API 是主结构体，描述一个完整的 API 配置
type API struct {
	Name         string                `json:"name"`          // API 名称
	Description  string                `json:"description"`   // API 描述
	Prompt       map[string]PromptNode `json:"prompt"`        // 节点 ID -> 节点结构
	ComfyuiNodes []string              `json:"comfyui_nodes"` // ComfyUI 节点服务器列表
	Variables    map[string]Variable   `json:"variables"`     // 可替换变量定义
	Token        string                `json:"token"`         // API Token
}
