package utils

import (
	"log"
	"os"
	"strings"
)

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// Logger provides structured logging
type Logger struct {
	level  LogLevel
	logger *log.Logger
}

// NewLogger creates a new logger with specified level
func NewLogger(levelStr string) *Logger {
	level := parseLogLevel(levelStr)
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	
	return &Logger{
		level:  level,
		logger: logger,
	}
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToLower(levelStr) {
	case "debug":
		return LogLevelDebug
	case "info":
		return LogLevelInfo
	case "warn", "warning":
		return LogLevelWarn
	case "error":
		return LogLevelError
	default:
		return LogLevelInfo
	}
}

// Debug logs debug messages
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level <= LogLevelDebug {
		l.logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs info messages
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level <= LogLevelInfo {
		l.logger.Printf("[INFO] "+format, args...)
	}
}

// Warn logs warning messages
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level <= LogLevelWarn {
		l.logger.Printf("[WARN] "+format, args...)
	}
}

// Error logs error messages
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level <= LogLevelError {
		l.logger.Printf("[ERROR] "+format, args...)
	}
}

// Fatal logs fatal messages and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.logger.Printf("[FATAL] "+format, args...)
	os.Exit(1)
}