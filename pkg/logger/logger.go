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

// Init initializes the global logger
func Init() {
	env := strings.ToLower(os.Getenv("ENV"))
	logLevel := INFO

	// Set log level based on environment
	if env == "development" || env == "dev" {
		logLevel = DEBUG
	}

	// Enable debug level if DEBUG env var is true
	if os.Getenv("DEBUG") == "true" {
		logLevel = DEBUG
	}

	std = New(os.Stdout, os.Stderr, logLevel)
}

// New creates a new logger instance
// stdout: output for INFO and DEBUG logs
// stderr: output for WARN, ERROR, and FATAL logs
func New(stdout, stderr io.Writer, level Level) *Logger {
	flags := log.Ldate | log.Ltime

	return &Logger{
		debug: log.New(stdout, "[DEBUG] ", flags),
		info:  log.New(stdout, "[INFO]  ", flags),
		warn:  log.New(stderr, "[WARN]  ", flags),
		error: log.New(stderr, "[ERROR] ", flags),
		fatal: log.New(stderr, "[FATAL] ", flags),
		level: level,
	}
}

// Debug outputs a debug log
func Debug(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= DEBUG {
		std.debug.Printf(format, v...)
	}
}

// Info outputs an info log
func Info(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Printf(format, v...)
	}
}

// Warn outputs a warning log
func Warn(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= WARN {
		std.warn.Printf(format, v...)
	}
}

// Error outputs an error log
func Error(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= ERROR {
		std.error.Printf(format, v...)
	}
}

// Fatal outputs a fatal error and exits the program
func Fatal(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	std.fatal.Printf(format, v...)
	os.Exit(1)
}

// Println is compatible with the standard log.Println (outputs to INFO)
func Println(v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Println(v...)
	}
}

// Printf is compatible with the standard log.Printf (outputs to INFO)
func Printf(format string, v ...interface{}) {
	if std == nil {
		Init()
	}
	if std.level <= INFO {
		std.info.Printf(format, v...)
	}
}

// SetLevel sets the log level
func SetLevel(level Level) {
	if std == nil {
		Init()
	}
	std.level = level
}

// GetLevel returns the current log level
func GetLevel() Level {
	if std == nil {
		Init()
	}
	return std.level
}
