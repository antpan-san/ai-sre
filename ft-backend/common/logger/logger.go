package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// 日志等级
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// 当前日志等级和 logger 实例
var (
	currentLevel LogLevel
	logger       *log.Logger
)

// 日志等级映射
var levelMap = map[string]LogLevel{
	"debug": DEBUG,
	"info":  INFO,
	"warn":  WARN,
	"error": ERROR,
}

// 颜色映射
var colorMap = map[LogLevel]string{
	DEBUG: "",         // 默认颜色
	INFO:  "\033[32m", // 绿色
	WARN:  "\033[33m", // 黄色
	ERROR: "\033[31m", // 红色
}

// 重置颜色
const resetColor = "\033[0m"

// InitLogger 初始化日志系统
// levelStr: "debug", "info", "warn", "error"
// writer: 输出目标（nil 表示输出到控制台）
func InitLogger(levelStr string, writer io.Writer) {
	levelStr = strings.ToLower(levelStr)
	level, ok := levelMap[levelStr]
	if !ok {
		level = INFO
	}

	currentLevel = level

	if writer == nil {
		writer = os.Stdout
	}

	logger = log.New(writer, "", log.LstdFlags|log.Lshortfile)
}

// internalLog 内部日志打印
func internalLog(level LogLevel, prefix string, format string, v ...any) {
	if level < currentLevel {
		return
	}

	msg := fmt.Sprintf(format, v...)

	color := colorMap[level]
	if color != "" {
		msg = fmt.Sprintf("%s%s%s", color, msg, resetColor)
	}

	logger.Output(3, fmt.Sprintf("[%s] %s", prefix, msg))
}

// -------------------- 对外接口 --------------------

// Debug 调试日志，支持格式化
func Debug(format string, v ...any) {
	internalLog(DEBUG, "DEBUG", format, v...)
}

// Info 普通日志，支持格式化
func Info(format string, v ...any) {
	internalLog(INFO, "INFO", format, v...)
}

// Warn 警告日志，支持格式化
func Warn(format string, v ...any) {
	internalLog(WARN, "WARN", format, v...)
}

// Error 错误日志，可传 error 或格式化字符串
func Error(v ...any) {
	if len(v) == 0 {
		return
	}

	// 如果只有一个参数，自动判断是否为 error
	if len(v) == 1 {
		switch val := v[0].(type) {
		case error:
			internalLog(ERROR, "ERROR", "%v", val)
			return
		case string:
			internalLog(ERROR, "ERROR", "%s", val)
			return
		}
	}

	// 多参数时，视为格式化输出
	format := fmt.Sprint(v[0])
	args := v[1:]
	internalLog(ERROR, "ERROR", format, args...)
}
