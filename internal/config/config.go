package config

import (
	"log/slog"
	"os"
	"time"
)

// Defaults for server environment variables.
const (
	DefaultServerPort    = "8080"
	DefaultStatePath     = "/var/lib/watcher/state.json"
	DefaultCheckInterval = "5m"
	DefaultLogFormat     = "text"
)

// ServerConfig holds values read from the process environment for the HTTP server and watcher.
type ServerConfig struct {
	LogFormat     string
	Port          string
	StatePath     string
	CheckInterval time.Duration
	IPURLs        string
}

func GetEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// LoadServerConfig reads server and watcher settings from the environment.
func LoadServerConfig() ServerConfig {
	return ServerConfig{
		LogFormat:     GetEnv("LOG_FORMAT", DefaultLogFormat),
		Port:          GetEnv("PORT", DefaultServerPort),
		StatePath:     GetEnv("STATE_PATH", DefaultStatePath),
		CheckInterval: parseDurationEnv("CHECK_INTERVAL", DefaultCheckInterval),
		IPURLs:        GetEnv("IP_URLS", ""),
	}
}

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
