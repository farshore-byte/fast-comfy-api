package model

// S3Config 定义 MinIO/S3 存储配置
type S3Config struct {
	Endpoint     string `yaml:"endpoint"`      // MinIO 服务地址
	Bucket       string `yaml:"bucket"`        // 桶名
	Region       string `yaml:"region"`        // 区域
	AccessKey    string `yaml:"access_key"`    // 访问密钥
	SecretKey    string `yaml:"secret_key"`    // 密钥
	UseSSL       bool   `yaml:"use_ssl"`       // 是否使用 SSL
	InputPrefix  string `yaml:"input_prefix"`  // 上传文件前缀
	OutputPrefix string `yaml:"output_prefix"` // 输出文件前缀
}

// ServerConfig 定义服务配置
type ServerConfig struct {
	Port int `yaml:"port"` // 服务监听端口（建议用 int 更方便绑定端口）
}

// HotReloadConfig 定义热重载配置
type HotReloadConfig struct {
	Enabled  bool `yaml:"enabled"`  // 是否启用热重载
	Interval int  `yaml:"interval"` // 检查间隔（秒）
}

// FeishuConfig 定义飞书配置
type FeishuConfig struct {
	WebHook string `yaml:"webhook"` // 飞书 WebHook 地址
}

// Config 整体配置
type Config struct {
	S3        S3Config        `yaml:"s3"`
	Server    ServerConfig    `yaml:"server"`
	HotReload HotReloadConfig `yaml:"hot_reload"`
	Feishu    FeishuConfig    `yaml:"feishu"`
}
