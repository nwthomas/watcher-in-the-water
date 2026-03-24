package logger

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want slog.Level
	}{
		{name: "empty", in: "", want: slog.LevelInfo},
		{name: "whitespace", in: "  ", want: slog.LevelInfo},
		{name: "info", in: "info", want: slog.LevelInfo},
		{name: "info_upper", in: "INFO", want: slog.LevelInfo},
		{name: "debug", in: "debug", want: slog.LevelDebug},
		{name: "warn", in: "warn", want: slog.LevelWarn},
		{name: "warning", in: "warning", want: slog.LevelWarn},
		{name: "error", in: "error", want: slog.LevelError},
		{name: "unknown_defaults_info", in: "bogus", want: slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseLevel(tt.in)
			if got != tt.want {
				t.Fatalf("parseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsKnownLogLevel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in    string
		known bool
	}{
		{"", true},
		{"info", true},
		{"debug", true},
		{"bogus", false},
	}
	for _, tc := range cases {
		if got := isKnownLogLevel(tc.in); got != tc.known {
			t.Fatalf("isKnownLogLevel(%q) = %v, want %v", tc.in, got, tc.known)
		}
	}
}

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
