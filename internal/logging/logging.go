package logging

import (
	"io"
	"log/slog"
	"strings"
)

func ParseLevel(configured string) (slog.Level, bool) {
	switch strings.ToLower(strings.TrimSpace(configured)) {
	case "debug":
		return slog.LevelDebug, true
	case "", "info":
		return slog.LevelInfo, true
	case "warn", "warning":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

func NewJSONLogger(output io.Writer, configuredLevel string) (*slog.Logger, bool) {
	level, valid := ParseLevel(configuredLevel)
	logger := slog.New(slog.NewJSONHandler(output, &slog.HandlerOptions{Level: level}))
	return logger, valid
}
