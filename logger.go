package main

import (
	"log/slog"
	"os"
	"strings"
)

var logger = newLogger()

func newLogger() *slog.Logger {
	levelVar := new(slog.LevelVar)
	levelVar.Set(slog.LevelInfo)

	if levelEnv := strings.TrimSpace(os.Getenv("LOG_LEVEL")); levelEnv != "" {
		switch strings.ToLower(levelEnv) {
		case "debug":
			levelVar.Set(slog.LevelDebug)
		case "info":
			levelVar.Set(slog.LevelInfo)
		case "warn", "warning":
			levelVar.Set(slog.LevelWarn)
		case "error":
			levelVar.Set(slog.LevelError)
		default:
			levelVar.Set(slog.LevelInfo)
		}
	}

	opts := &slog.HandlerOptions{Level: levelVar}

	format := strings.TrimSpace(os.Getenv("LOG_FORMAT"))
	var handler slog.Handler
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
