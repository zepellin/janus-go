package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Logger is the global logger instance that can be used across packages
var Logger *slog.Logger

// InitLogger initializes the global logger with the specified level
func InitLogger(level string) {
	logLevel := parseLogLevel(level)
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
