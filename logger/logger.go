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
	// Logs must go to stderr: stdout is reserved for the credential_process
	// JSON payload that the AWS CLI/SDK parses.
	Logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
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
