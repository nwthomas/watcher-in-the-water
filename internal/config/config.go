package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"
)

const (
	DEFAULT_SERVER_PORT    = "8080"
	DEFAULT_STATE_PATH     = "/var/lib/watcher/state.json"
	DEFAULT_CHECK_INTERVAL = "30s"
	DEFAULT_LOG_FORMAT     = "text"
	DEFAULT_LOG_LEVEL      = "info"
	DEFAULT_EMAIL_PORT     = "587"
	DEFAULT_EMAIL_TLS      = "true"
)

type ServerConfig struct {
	LogFormat     string
	LogLevel      string
	Port          string
	StatePath     string
	CheckInterval time.Duration
	IPURLs        string
	WebhookURLs   string

	EmailHostName      string
	EmailPassword      string
	EmailPersonalEmail string
	EmailPort          int
	EmailTLS           bool
	EmailUsername      string
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

		EmailHostName:      GetEnv("EMAIL_HOST_NAME", ""),
		EmailPassword:      GetEnv("EMAIL_PASSWORD", ""),
		EmailPersonalEmail: GetEnv("EMAIL_PERSONAL_EMAIL", ""),
		EmailPort:          parseIntEnv("EMAIL_PORT", DEFAULT_EMAIL_PORT),
		EmailTLS:           parseBoolEnv("EMAIL_TLS", DEFAULT_EMAIL_TLS),
		EmailUsername:      GetEnv("EMAIL_USERNAME", ""),
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

// parseIntEnv parses an integer from the environment or returns the default value if invalid
func parseIntEnv(key, defaultVal string) int {
	s := GetEnv(key, defaultVal)
	n, err := strconv.Atoi(s)
	if err != nil {
		slog.Warn("invalid integer, using default", "key", key, "value", s, "default", defaultVal, "err", err)
		fallback, err2 := strconv.Atoi(defaultVal)
		if err2 != nil {
			return 0
		}
		return fallback
	}
	return n
}

// parseBoolEnv parses a boolean from the environment or returns the default value if invalid
func parseBoolEnv(key, defaultVal string) bool {
	s := GetEnv(key, defaultVal)
	b, err := strconv.ParseBool(s)
	if err != nil {
		slog.Warn("invalid boolean, using default", "key", key, "value", s, "default", defaultVal, "err", err)
		fallback, err2 := strconv.ParseBool(defaultVal)
		if err2 != nil {
			return false
		}
		return fallback
	}
	return b
}
