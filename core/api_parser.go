package core

import (
	"encoding/json"
	"errors"
	"farshore.ai/fast-comfy-api/model"
	"fmt"
	"github.com/google/uuid"
	"math/rand"
	"reflect"
	"strings"
	"time"
)

type APIParser struct {
	api *model.API // ✅ 小写：内部字段，不暴露
}

// NewAPIParser 创建解析器
func NewAPIParser(apijson []byte) (*APIParser, error) {
	api := &model.API{}
	if err := json.Unmarshal(apijson, api); err != nil {
		return nil, err
	}
	return &APIParser{api: api}, nil
}

// ✅ 封装访问方法

func (p *APIParser) GetName() string {
	if p.api == nil {
		return ""
	}
	return p.api.Name
}

func (p *APIParser) GetDescription() string {
	if p.api == nil {
		return ""
	}
	return p.api.Description
}

func (p *APIParser) GetComfyuiNodes() []string {
	if p.api == nil {
		return nil
	}
	return p.api.ComfyuiNodes
}

func (p *APIParser) GetVariableNames() []string {
	if p.api == nil || p.api.Variables == nil {
		return nil
	}
	keys := make([]string, 0, len(p.api.Variables))
	for k := range p.api.Variables {
		keys = append(keys, k)
	}
	return keys
}

func (p *APIParser) GetToken() string {
	if p.api == nil {
		return ""
	}
	return p.api.Token
}

// ApplyVariables 将传入变量替换进 Prompt 中的指定路径，返回新的 Prompt
func (p *APIParser) ApplyVariables(vars map[string]interface{}) (map[string]model.PromptNode, error) {
	if p.api == nil {
		return nil, errors.New("API 未初始化")
	}
	if p.api.Variables == nil {
		return nil, errors.New("API 未定义变量")
	}

	// 🔄 深拷贝 Prompt，避免修改原始数据
	promptCopy := make(map[string]model.PromptNode)
	for id, node := range p.api.Prompt {
		promptCopy[id] = node
	}

	// 遍历 API 定义的变量（以定义为准）
	for varName, def := range p.api.Variables {
		// 获取实际要设置的值
		val, exists := vars[varName]
		if !exists {
			// 如果没传入，用默认值
			val = def.Default
		}

		// 跳过空路径
		if def.Path == "" {
			return nil, fmt.Errorf("变量 '%s' 缺少 path", varName)
		}

		// 🔍 路径示例: "757.inputs.wildcard_text"
		parts := strings.Split(def.Path, ".")
		if len(parts) < 3 {
			return nil, fmt.Errorf("变量 '%s' 的路径格式错误，应为: 节点ID.inputs.参数名", varName)
		}

		nodeID := parts[0]
		field := parts[1]
		key := parts[2]

		node, ok := promptCopy[nodeID]
		if !ok {
			return nil, fmt.Errorf("未找到节点ID: %s (变量: %s)", nodeID, varName)
		}

		if field != "inputs" {
			return nil, fmt.Errorf("暂仅支持修改 inputs 字段，变量 '%s' 路径: %s", varName, def.Path)
		}

		// 检查输入参数是否存在
		if _, exists := node.Inputs[key]; !exists {
			return nil, fmt.Errorf("节点 %s 中不存在输入参数 %s (变量: %s)", nodeID, key, varName)
		}

		// ✅ 类型检查
		if def.Type != "" && !checkType(def.Type, val) {
			return nil, fmt.Errorf("变量 '%s' 类型不匹配，应为 %s，实际是 %T", varName, def.Type, val)
		}

		// ✅ 替换
		node.Inputs[key] = val
		promptCopy[nodeID] = node
	}

	// 自动处理

	// 1️⃣ step 1 filename_prefix 字段自动替换输出文件名，防止冲突覆盖
	promptCopy = ApplyUniqueFilename(promptCopy)

	// 2️⃣ step 2： 随机生成 seed 字段
	promptCopy = ApplyRandomSeed(promptCopy)

	return promptCopy, nil
}

// 辅助函数：checkType 判断变量类型是否匹配
func checkType(expected string, val interface{}) bool {
	if val == nil {
		return true // 允许 nil（可选变量）
	}
	switch expected {
	case "string":
		_, ok := val.(string)
		return ok
	case "number", "float":
		switch val.(type) {
		case float64, float32, int, int64, int32:
			return true
		}
	case "bool":
		_, ok := val.(bool)
		return ok
	case "object":
		// 任意 map
		_, ok := val.(map[string]interface{})
		return ok
	case "array":
		// 任意 slice
		v := reflect.ValueOf(val)
		return v.Kind() == reflect.Slice
	}
	return true // 未定义类型则不强制检查
}

// 对字段名包含 "seed" 且值为数值类型的字段，随机生成新值替换
func ApplyRandomSeed(prompt map[string]model.PromptNode) map[string]model.PromptNode {
	rand.Seed(time.Now().UnixNano()) // 初始化随机种子

	// 深拷贝 prompt，避免修改原始数据
	newPrompt := make(map[string]model.PromptNode, len(prompt))
	for id, node := range prompt {
		nodeCopy := node
		newInputs := make(map[string]interface{}, len(node.Inputs))
		for key, val := range node.Inputs {
			// 判断字段名是否包含 seed
			if strings.Contains(strings.ToLower(key), "seed") {
				// 仅处理数值类型
				if isNumber(val) {
					newInputs[key] = rand.Int63() // 随机生成 int64
					continue
				}
			}
			newInputs[key] = val
		}
		nodeCopy.Inputs = newInputs
		newPrompt[id] = nodeCopy
	}

	return newPrompt
}

// 对 包含 filename_prefix 的字段，使用 uuid 生成唯一文件名替换
// ApplyUniqueFilename 给 prompt 中所有 filename_prefix 字段追加唯一 hex UUID
func ApplyUniqueFilename(prompt map[string]model.PromptNode) map[string]model.PromptNode {
	// 深拷贝 prompt，避免修改原始数据
	newPrompt := make(map[string]model.PromptNode, len(prompt))

	for id, node := range prompt {
		nodeCopy := node
		newInputs := make(map[string]interface{}, len(node.Inputs))

		for key, val := range node.Inputs {
			// 判断字段名是否包含 filename_prefix
			if strings.Contains(strings.ToLower(key), "filename_prefix") {
				if s, ok := val.(string); ok {
					// ✅ 生成纯 hex UUID（32 位，无连字符）
					u, err := uuid.NewRandom()
					if err != nil {
						panic(err)
					}
					uniqueID := strings.ReplaceAll(u.String(), "-", "")
					newInputs[key] = fmt.Sprintf("%s_%s", s, uniqueID)
					continue
				}
			}
			newInputs[key] = val
		}

		nodeCopy.Inputs = newInputs
		newPrompt[id] = nodeCopy
	}

	return newPrompt
}

// 辅助函数 判断是否为数字类型
func isNumber(val interface{}) bool {
	switch val.(type) {
	case int, int32, int64, float32, float64:
		return true
	default:
		// 尝试反射判断
		v := reflect.ValueOf(val)
		kind := v.Kind()
		return kind >= reflect.Int && kind <= reflect.Float64
	}
}
