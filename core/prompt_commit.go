package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"farshore.ai/fast-comfy-api/model"
)

// PromptCommitResponse 定义接口返回的结构
type PromptCommitResponse struct {
	PromptID   string                 `json:"prompt_id"`
	NodeErrors map[string]interface{} `json:"node_errors,omitempty"`
	Number     int                    `json:"number,omitempty"`
}

// PromptCommit 提交 prompt，如果有prompt_id，返回prompt_id
func PromptCommit(host string, prompt map[string]model.PromptNode, clientID string) (string, error) {
	fullURL := fmt.Sprintf("%s/prompt", strings.TrimRight(host, "/"))
	body := map[string]interface{}{
		"prompt":    prompt,
		"client_id": clientID,
	}

	jsonData, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompt: %w", err)
	}

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to generate prompt: %s", string(respBody))
	}

	// 解析 JSON 到结构体
	var result PromptCommitResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 日志打印
	log.Printf("🚀 Prompt upload in %s, return prompt_id: %s", host, result.PromptID)

	return result.PromptID, nil
}
