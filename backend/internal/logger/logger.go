package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	mu     sync.RWMutex
	global *slog.Logger
)

func Init(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
	}
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") && parseLevel(level) == slog.LevelDebug {
		opts.Level = slog.LevelInfo
	}
	l := slog.New(slog.NewJSONHandler(w, opts))
	mu.Lock()
	global = l
	mu.Unlock()
	slog.SetDefault(l)
	return l
}

func L() *slog.Logger {
	mu.RLock()
	l := global
	mu.RUnlock()
	if l == nil {
		return Init("info", os.Stdout)
	}
	return l
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
