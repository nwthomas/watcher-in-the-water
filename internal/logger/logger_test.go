package logger

import (
	"log/slog"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		format string
		level  string
	}{
		{name: "text format", format: "text", level: "info"},
		{name: "json format", format: "json", level: "info"},
		{name: "unknown format falls back", format: "unknown", level: "info"},
		{name: "debug level", format: "text", level: "debug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.format, tt.level)
			slog.Default().Info("logger initialized for test")
		})
	}
}
