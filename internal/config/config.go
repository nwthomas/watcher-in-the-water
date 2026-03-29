package config

import (
	"log/slog"
	"os"
	"time"
)

const (
	DEFAULT_SERVER_PORT    = "8080"
	DEFAULT_STATE_PATH     = "/var/lib/watcher/state.json"
	DEFAULT_CHECK_INTERVAL = "30s"
	DEFAULT_LOG_FORMAT     = "text"
	DEFAULT_LOG_LEVEL      = "info"
)

type ServerConfig struct {
	LogFormat     string
	LogLevel      string
	Port          string
	StatePath     string
	CheckInterval time.Duration
	IPURLs        string
	WebhookURLs   string
}

// GetEnv returns the environment variable value if set, otherwise the default value
func GetEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// LoadServerConfig reads server and watcher settings from the environment
func LoadServerConfig() ServerConfig {
	return ServerConfig{
		LogFormat:     GetEnv("LOG_FORMAT", DEFAULT_LOG_FORMAT),
		LogLevel:      GetEnv("LOG_LEVEL", DEFAULT_LOG_LEVEL),
		Port:          GetEnv("PORT", DEFAULT_SERVER_PORT),
		StatePath:     GetEnv("STATE_PATH", DEFAULT_STATE_PATH),
		CheckInterval: parseDurationEnv("CHECK_INTERVAL", DEFAULT_CHECK_INTERVAL),
		IPURLs:        GetEnv("IP_URLS", ""),
		WebhookURLs:   GetEnv("WEBHOOK_URLS", ""),
	}
}

// parseDurationEnv parses a duration string from the environment or returns the default value if invalid
func parseDurationEnv(key, defaultVal string) time.Duration {
	s := GetEnv(key, defaultVal)
	d, err := time.ParseDuration(s)
	if err != nil {
		slog.Warn("invalid duration, using default", "key", key, "value", s, "default", defaultVal, "err", err)
		fallback, err2 := time.ParseDuration(defaultVal)
		if err2 != nil {
			return 5 * time.Minute
		}
		return fallback
	}
	return d
}
