package config

const (
	WarningQueueSize   = 5   // 每个comfyui服务器触发报警的上限
	WarningCPUPercent  = 70. // CPU使用率报警阈值
	WarningVRAMPercent = 98. // 显存使用率报警阈值
	WarningRAMPercent  = 80. // 内存使用量报警阈值
	WarningGPUTemp     = 70  // GPU温度报警
	WarningInterval    = 10  // 同种报警的报警间隔

)
