package config

import (
	"farshore.ai/fast-comfy-api/model"
	"gopkg.in/yaml.v3"
	"os"
)

// LoadConfig 从 YAML 文件加载配置

func LoadConfig(path string) (*model.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config model.Config

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
