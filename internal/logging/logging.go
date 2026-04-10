package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

func New(appName, logDir, level string) *slog.Logger {
	var output io.Writer = os.Stderr
	if err := os.MkdirAll(logDir, 0o755); err == nil {
		output = io.MultiWriter(os.Stderr, &lumberjack.Logger{
			Filename:   filepath.Join(logDir, strings.ToLower(appName)+".log"),
			MaxSize:    10,
			MaxBackups: 10,
		})
	}
	return slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: parseLevel(level)}))
}

func parseLevel(level string) slog.Leveler {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARNING":
		return slog.LevelWarn
	case "ERROR", "CRITICAL":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
