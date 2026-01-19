package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	fatal *log.Logger
	level Level
}

var std *Logger

// Init 初始化全局 logger
func Init() {
	env := strings.ToLower(os.Getenv("ENV"))
	logLevel := INFO

	// 根据环境设置日志级别
	if env == "development" || env == "dev" {
		logLevel = DEBUG
	}

	if os.Getenv("DEBUG") == "true" {
		logLevel = DEBUG
	}

	std = New(os.Stdout, os.Stderr, logLevel)
}

// New 创建新的 logger
// stdout: INFO 和 DEBUG 日志输出
// stderr:  WARN, ERROR, FATAL 日志输出
func New(stdout, stderr io.Writer, level Level) *Logger {
	flags := log.Ldate | log.Ltime
	// | log.Lshortfile

	return &Logger{
		debug: log.New(stdout, "[DEBUG] ", flags),
		info:  log.New(stdout, "[INFO]  ", flags),
		warn:  log.New(stderr, "[WARN]  ", flags),
		error: log.New(stderr, "[ERROR] ", flags),
		fatal: log.New(stderr, "[FATAL] ", flags),
		level: level,
	}
}

// Debug 输出调试日志
func Debug(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= DEBUG {
		std.debug.Printf(format, v...)
	}
}

// Info 输出信息日志
func Info(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Printf(format, v...)
	}
}

// Warn 输出警告日志
func Warn(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= WARN {
		std.warn.Printf(format, v...)
	}
}

// Error 输出错误日志
func Error(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= ERROR {
		std.error.Printf(format, v...)
	}
}

// Fatal 输出致命错误并退出程序
func Fatal(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	std.fatal.Printf(format, v...)
	os.Exit(1)
}

// Println 兼容标准 log.Println（输出到 INFO）
func Println(v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Println(v...)
	}
}

// Printf 兼容标准 log.Printf（输出到 INFO）
func Printf(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Printf(format, v...)
	}
}

// SetLevel 设置日志级别
func SetLevel(level Level) {
	if std == nil {
		Init()
	}
	std.level = level
}

// GetLevel 获取当前日志级别
func GetLevel() Level {
	if std == nil {
		Init()
	}
	return std.level
}
