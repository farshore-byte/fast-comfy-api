package core

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type HistoryResponse map[string]struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
	} `json:"outputs"`
}

// GetFinalOutputImages 根据 prompt_id 获取最终 output 类型图片的访问 URL
func GetOutputURLs(host, promptID string) ([]string, error) {
	url := fmt.Sprintf("%s/history/%s", host, promptID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求 history 接口失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("history 接口返回状态码 %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 history 响应失败: %v", err)
	}

	var history HistoryResponse
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	task, ok := history[promptID]
	if !ok {
		return nil, fmt.Errorf("history 中未找到 prompt_id: %s", promptID)
	}

	var urls []string
	for _, output := range task.Outputs {
		for _, img := range output.Images {
			if img.Type == "output" { // 只取最终 output 类型
				url := fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=%s", host, img.Filename, img.Subfolder, img.Type)
				urls = append(urls, url)
			}
		}
	}

	return urls, nil
}

// DownloadFile 下载 URL 文件到本地指定目录，返回本地完整路径
func DownloadFile(fileURL, localDir string) (string, error) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("解析 URL 失败: %v", err)
	}

	// 尝试从 query 参数获取 filename
	filename := parsedURL.Query().Get("filename")
	if filename == "" {
		// fallback: 直接用路径最后一部分
		filename = filepath.Base(parsedURL.Path)
	}

	localPath := filepath.Join(localDir, filename)

	// 创建目录
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", fmt.Errorf("创建本地目录失败: %v", err)
	}

	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件返回状态码 %d", resp.StatusCode)
	}

	outFile, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("写入本地文件失败: %v", err)
	}

	return localPath, nil
}

// FetchOutputFiles 获取 promptID 的最终 output 图片，并下载到本地缓存目录，返回本地文件路径列表
func FetchOutputFiles(ServerURL, promptID, cacheDir string) ([]string, error) {
	urls, err := GetOutputURLs(ServerURL, promptID)
	if err != nil {
		return nil, err
	}

	var localPaths []string
	for _, url := range urls {
		localPath, err := DownloadFile(url, cacheDir)
		if err != nil {
			return nil, fmt.Errorf("下载图片失败 url=%s, err=%v", url, err)
		}
		localPaths = append(localPaths, localPath)
	}

	return localPaths, nil
}
