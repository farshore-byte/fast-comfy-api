package core

import (
	"fmt"
	"log"
	"os"
)

// 日志颜色常量
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// 日志类型
type LogType string

const (
	LogTypeMessageWorker LogType = "MESSAGE_WORKER"
	LogTypeAPIRuntime    LogType = "API_RUNTIME"
)

// 日志配置
type LogConfig struct {
	Type  LogType
	Name  string
	Color string
}

// 预定义的日志配置
var logConfigs = map[LogType]LogConfig{
	LogTypeMessageWorker: {
		Type:  LogTypeMessageWorker,
		Name:  "MessageWorker",
		Color: ColorCyan,
	},
	LogTypeAPIRuntime: {
		Type:  LogTypeAPIRuntime,
		Name:  "APIRuntime",
		Color: ColorYellow,
	},
}

// Logger 结构体
type Logger struct {
	logType LogType
}

// NewLogger 创建新的日志器
func NewLogger(logType LogType) *Logger {
	return &Logger{
		logType: logType,
	}
}

// 获取颜色前缀
func (l *Logger) getColorPrefix() string {
	if config, exists := logConfigs[l.logType]; exists {
		return config.Color + "[" + config.Name + "]" + ColorReset
	}
	return ColorWhite + "[Unknown]" + ColorReset
}

// 检查是否支持颜色输出
func supportsColor() bool {
	// 检查终端是否支持颜色
	if os.Getenv("TERM") == "dumb" {
		return false
	}

	// 检查是否是 Windows 系统（Windows 终端支持颜色）
	if os.Getenv("OS") == "Windows_NT" {
		return true
	}

	// 检查是否是 TTY
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// Printf 格式化输出日志
func (l *Logger) Printf(format string, args ...interface{}) {
	if supportsColor() {
		// 整行颜色包裹
		config := logConfigs[l.logType]
		coloredFormat := fmt.Sprintf("%s[%s] %s%s",
			config.Color,
			config.Name,
			format,
			ColorReset,
		)
		log.Printf(coloredFormat, args...)
	} else {
		plainFormat := fmt.Sprintf("[%s] %s", l.logType, format)
		log.Printf(plainFormat, args...)
	}
}

// Println 输出日志
func (l *Logger) Println(args ...interface{}) {
	if supportsColor() {
		config := logConfigs[l.logType]
		message := fmt.Sprint(args...)
		coloredMessage := fmt.Sprintf("%s[%s] %s%s",
			config.Color,
			config.Name,
			message,
			ColorReset,
		)
		log.Println(coloredMessage)
	} else {
		plainMessage := fmt.Sprintf("[%s] %s", l.logType, fmt.Sprint(args...))
		log.Println(plainMessage)
	}
}

// 全局日志器实例
var (
	MessageWorkerLogger = NewLogger(LogTypeMessageWorker)
	APIRuntimeLogger    = NewLogger(LogTypeAPIRuntime)
)

// 注册新的日志类型
func RegisterLogType(logType LogType, name string, color string) {
	logConfigs[logType] = LogConfig{
		Type:  logType,
		Name:  name,
		Color: color,
	}
}

// 获取所有已注册的日志类型
func GetRegisteredLogTypes() []LogType {
	types := make([]LogType, 0, len(logConfigs))
	for logType := range logConfigs {
		types = append(types, logType)
	}
	return types
}

// 创建自定义日志器
func CreateLogger(logType LogType, name string, color string) *Logger {
	RegisterLogType(logType, name, color)
	return NewLogger(logType)
}

// 快捷函数
func LogMessageWorker(format string, args ...interface{}) {
	MessageWorkerLogger.Printf(format, args...)
}

func LogAPIRuntime(format string, args ...interface{}) {
	APIRuntimeLogger.Printf(format, args...)
}

// 示例：如何添加新的日志类型
/*
// 1. 定义新的日志类型常量
const LogTypeDatabase LogType = "DATABASE"

// 2. 注册新的日志类型（可选颜色：ColorRed, ColorGreen, ColorBlue, ColorPurple, ColorCyan, ColorYellow, ColorWhite）
func init() {
	RegisterLogType(LogTypeDatabase, "Database", ColorGreen)
}

// 3. 创建全局日志器实例
var DatabaseLogger = NewLogger(LogTypeDatabase)

// 4. 创建快捷函数（可选）
func LogDatabase(format string, args ...interface{}) {
	DatabaseLogger.Printf(format, args...)
}

// 5. 使用示例
// LogDatabase("数据库连接成功: %s", "mysql://localhost:3306")
*/
