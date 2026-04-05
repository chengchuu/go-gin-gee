package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// Create buffers
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Create a test logger
	testLogger := New(&stdout, &stderr, DEBUG)
	std = testLogger

	// Test DEBUG
	Debug("debug message: %s", "test")
	if !strings.Contains(stdout.String(), "[DEBUG]") || !strings.Contains(stdout.String(), "debug message: test") {
		t.Errorf("Debug log failed")
	}

	// Clear buffers
	stdout.Reset()
	stderr.Reset()

	// Test INFO
	Info("info message: %d", 123)
	if !strings.Contains(stdout.String(), "[INFO]") || !strings.Contains(stdout.String(), "info message: 123") {
		t.Errorf("Info log failed")
	}

	// Clear buffers
	stdout.Reset()
	stderr.Reset()

	// Test WARN
	Warn("warn message")
	if !strings.Contains(stderr.String(), "[WARN]") || !strings.Contains(stderr.String(), "warn message") {
		t.Errorf("Warn log failed")
	}

	// Clear buffers
	stdout.Reset()
	stderr.Reset()

	// Test ERROR
	Error("error message")
	if !strings.Contains(stderr.String(), "[ERROR]") || !strings.Contains(stderr.String(), "error message") {
		t.Errorf("Error log failed")
	}
}

func TestLogLevel(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Set log level to WARN
	testLogger := New(&stdout, &stderr, WARN)
	std = testLogger

	// DEBUG and INFO should not output
	Debug("debug message")
	Info("info message")
	if stdout.Len() > 0 {
		t.Errorf("DEBUG/INFO should not output when level is WARN")
	}

	// WARN should output
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
		{"unknown", INFO}, // Default value
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
	// Save original environment variables
	origEnv := os.Getenv("ENV")
	origDebug := os.Getenv("DEBUG")
	defer func() {
		os.Setenv("ENV", origEnv)
		os.Setenv("DEBUG", origDebug)
	}()

	// Test development environment
	os.Setenv("ENV", "development")
	std = nil // Reset
	Init()
	if GetLevel() != DEBUG {
		t.Errorf("Expected DEBUG level in development environment")
	}

	// Test production environment
	os.Setenv("ENV", "production")
	std = nil // Reset
	Init()
	if GetLevel() != INFO {
		t.Errorf("Expected INFO level in production environment")
	}

	// Test DEBUG environment variable
	os.Setenv("ENV", "production")
	os.Setenv("DEBUG", "true")
	std = nil // Reset
	Init()
	if GetLevel() != DEBUG {
		t.Errorf("Expected DEBUG level when DEBUG=true")
	}
}
