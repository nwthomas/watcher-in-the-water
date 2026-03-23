package config

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		setValue     bool
		value        string
		defaultValue string
		want         string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_CONFIG_SET",
			setValue:     true,
			value:        "configured",
			defaultValue: "fallback",
			want:         "configured",
		},
		{
			name:         "returns default when unset",
			key:          "TEST_CONFIG_UNSET",
			setValue:     false,
			defaultValue: "fallback",
			want:         "fallback",
		},
		{
			name:         "returns default when env is empty",
			key:          "TEST_CONFIG_EMPTY",
			setValue:     true,
			value:        "",
			defaultValue: "fallback",
			want:         "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setValue {
				t.Setenv(tt.key, tt.value)
			} else if err := os.Unsetenv(tt.key); err != nil {
				t.Fatalf("Unsetenv(%q) failed: %v", tt.key, err)
			}

			got := GetEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Fatalf("GetEnv(%q, %q) = %q, want %q", tt.key, tt.defaultValue, got, tt.want)
			}
		})
	}
}
