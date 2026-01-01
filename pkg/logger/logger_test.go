package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// 创建缓冲区
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// 创建测试 logger
	testLogger := New(&stdout, &stderr, DEBUG)
	std = testLogger

	// 测试 DEBUG
	Debug("debug message: %s", "test")
	if !strings.Contains(stdout.String(), "[DEBUG]") || !strings.Contains(stdout.String(), "debug message: test") {
		t.Errorf("Debug log failed")
	}

	// 清空缓冲区
	stdout.Reset()
	stderr.Reset()

	// 测试 INFO
	Info("info message: %d", 123)
	if !strings.Contains(stdout.String(), "[INFO]") || !strings.Contains(stdout.String(), "info message: 123") {
		t.Errorf("Info log failed")
	}

	// 清空缓冲区
	stdout.Reset()
	stderr.Reset()

	// 测试 WARN
	Warn("warn message")
	if !strings.Contains(stderr.String(), "[WARN]") || !strings.Contains(stderr.String(), "warn message") {
		t.Errorf("Warn log failed")
	}

	// 清空缓冲区
	stdout.Reset()
	stderr.Reset()

	// 测试 ERROR
	Error("error message")
	if !strings.Contains(stderr.String(), "[ERROR]") || !strings.Contains(stderr.String(), "error message") {
		t.Errorf("Error log failed")
	}
}

func TestLogLevel(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// 设置 WARN 级别
	testLogger := New(&stdout, &stderr, WARN)
	std = testLogger

	// DEBUG 和 INFO 不应该输出
	Debug("debug message")
	Info("info message")
	if stdout.Len() > 0 {
		t.Errorf("DEBUG/INFO should not output when level is WARN")
	}

	// WARN 应该输出
	Warn("warn message")
	if !strings.Contains(stderr.String(), "[WARN]") {
		t.Errorf("WARN should output when level is WARN")
	}
}

func TestLevelFromString(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"DEBUG", DEBUG},
		{"debug", DEBUG},
		{"INFO", INFO},
		{"info", INFO},
		{"WARN", WARN},
		{"WARNING", WARN},
		{"ERROR", ERROR},
		{"FATAL", FATAL},
		{"unknown", INFO}, // 默认值
	}

	for _, tt := range tests {
		result := LevelFromString(tt.input)
		if result != tt.expected {
			t.Errorf("LevelFromString(%s) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, result, tt.expected)
		}
	}
}

func TestInit(t *testing.T) {
	// 保存原始环境变量
	origEnv := os.Getenv("ENV")
	origDebug := os.Getenv("DEBUG")
	defer func() {
		os.Setenv("ENV", origEnv)
		os.Setenv("DEBUG", origDebug)
	}()

	// 测试开发环境
	os.Setenv("ENV", "development")
	std = nil // 重置
	Init()
	if GetLevel() != DEBUG {
		t.Errorf("Expected DEBUG level in development environment")
	}

	// 测试生产环境
	os.Setenv("ENV", "production")
	std = nil // 重置
	Init()
	if GetLevel() != INFO {
		t.Errorf("Expected INFO level in production environment")
	}

	// 测试 DEBUG 环境变量
	os.Setenv("ENV", "production")
	os.Setenv("DEBUG", "true")
	std = nil // 重置
	Init()
	if GetLevel() != DEBUG {
		t.Errorf("Expected DEBUG level when DEBUG=true")
	}
}
