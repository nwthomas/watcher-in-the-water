package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger for the process.
func Init(format string, level string) {
	raw := strings.TrimSpace(level)
	if raw != "" && !isKnownLogLevel(raw) {
		pre := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		pre.Warn("invalid LOG_LEVEL, using info", "level", raw)
	}
	lvl := parseLevel(raw)
	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func isKnownLogLevel(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "debug", "info", "warn", "warning", "error":
		return true
	default:
		return false
	}
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
