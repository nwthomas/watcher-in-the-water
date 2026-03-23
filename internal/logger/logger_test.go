package logger

import (
	"log/slog"
	"testing"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "text format", format: "text"},
		{name: "json format", format: "json"},
		{name: "unknown format falls back", format: "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Init(tt.format)
			slog.Default().Info("logger initialized for test")
		})
	}
}
