package config

import (
	"os"
	"testing"
	"time"
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

func TestLoadServerConfig(t *testing.T) {
	mustParse := func(s string) time.Duration {
		t.Helper()
		d, err := time.ParseDuration(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}

	tests := []struct {
		name string
		env  map[string]string
		want ServerConfig
	}{
		{
			name: "defaults when unset",
			env:  nil,
			want: ServerConfig{
				LogFormat:     DEFAULT_LOG_FORMAT,
				LogLevel:      DEFAULT_LOG_LEVEL,
				Port:          DEFAULT_SERVER_PORT,
				StatePath:     DEFAULT_STATE_PATH,
				CheckInterval: mustParse(DEFAULT_CHECK_INTERVAL),
				IPURLs:        "",
				WebhookURLs:   "",
				EmailPort:     587,
				EmailTLS:      true,
			},
		},
		{
			name: "overrides",
			env: map[string]string{
				"LOG_FORMAT":     "json",
				"LOG_LEVEL":      "debug",
				"PORT":           "3000",
				"STATE_PATH":     "/tmp/x.json",
				"CHECK_INTERVAL": "10s",
				"IP_URLS":        "https://example.com",
				"WEBHOOK_URLS":   "https://hooks.example.com/a, https://hooks.example.com/b",

				"EMAIL_HOST_NAME":      "smtp.mail.me.com",
				"EMAIL_PASSWORD":       "app-specific-password",
				"EMAIL_PERSONAL_EMAIL": "me@icloud.com",
				"EMAIL_PORT":           "2525",
				"EMAIL_TLS":            "false",
				"EMAIL_USERNAME":       "me@icloud.com",
			},
			want: ServerConfig{
				LogFormat:     "json",
				LogLevel:      "debug",
				Port:          "3000",
				StatePath:     "/tmp/x.json",
				CheckInterval: 10 * time.Second,
				IPURLs:        "https://example.com",
				WebhookURLs:   "https://hooks.example.com/a, https://hooks.example.com/b",

				EmailHostName:      "smtp.mail.me.com",
				EmailPassword:      "app-specific-password",
				EmailPersonalEmail: "me@icloud.com",
				EmailPort:          2525,
				EmailTLS:           false,
				EmailUsername:      "me@icloud.com",
			},
		},
		{
			name: "invalid email port and tls fall back to defaults",
			env: map[string]string{
				"EMAIL_PORT": "not-a-port",
				"EMAIL_TLS":  "maybe",
			},
			want: ServerConfig{
				LogFormat:     DEFAULT_LOG_FORMAT,
				LogLevel:      DEFAULT_LOG_LEVEL,
				Port:          DEFAULT_SERVER_PORT,
				StatePath:     DEFAULT_STATE_PATH,
				CheckInterval: mustParse(DEFAULT_CHECK_INTERVAL),
				EmailPort:     587,
				EmailTLS:      true,
			},
		},
		{
			name: "invalid check interval falls back to default duration",
			env: map[string]string{
				"CHECK_INTERVAL": "not-a-duration",
			},
			want: ServerConfig{
				LogFormat:     DEFAULT_LOG_FORMAT,
				LogLevel:      DEFAULT_LOG_LEVEL,
				Port:          DEFAULT_SERVER_PORT,
				StatePath:     DEFAULT_STATE_PATH,
				CheckInterval: mustParse(DEFAULT_CHECK_INTERVAL),
				IPURLs:        "",
				WebhookURLs:   "",
				EmailPort:     587,
				EmailTLS:      true,
			},
		},
	}

	keys := []string{"LOG_FORMAT", "LOG_LEVEL", "PORT", "STATE_PATH", "CHECK_INTERVAL", "IP_URLS", "WEBHOOK_URLS",
		"EMAIL_HOST_NAME", "EMAIL_PASSWORD", "EMAIL_PERSONAL_EMAIL", "EMAIL_PORT", "EMAIL_TLS", "EMAIL_USERNAME"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, k := range keys {
				t.Setenv(k, "")
			}
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got := LoadServerConfig()
			if got != tt.want {
				t.Fatalf("LoadServerConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
