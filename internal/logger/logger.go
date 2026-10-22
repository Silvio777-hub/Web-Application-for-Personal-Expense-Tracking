package logger

import (
	"context"
	"io"
	"log/slog"
)

var Global *slog.Logger

// Init initializes the global structured logger.
func Init(level string, output io.Writer) {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}

	handler := slog.NewJSONHandler(output, opts)
	Global = slog.New(handler)
	slog.SetDefault(Global)
}

// Helpers for easier access

func Info(msg string, args ...any) {
	Global.Info(msg, args...)
}

func Error(msg string, err error, args ...any) {
	if err != nil {
		args = append(args, slog.Any("error", err))
	}
	Global.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	Global.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	Global.Debug(msg, args...)
}

func FromContext(ctx context.Context) *slog.Logger {
	// Future: Extract request ID from context
	return Global
}
