package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"sync/atomic"
	"syscall"
	"time"

	_ "go.uber.org/automaxprocs"

	"github.com/nwthomas/watcher-in-the-water/internal/config"
	"github.com/nwthomas/watcher-in-the-water/internal/logger"
	"github.com/nwthomas/watcher-in-the-water/internal/publicip"
	"github.com/nwthomas/watcher-in-the-water/internal/watcher"
)

const (
	DEFAULT_SERVER_PORT    = "8080"
	DEFAULT_STATE_PATH     = "/var/lib/watcher/state.json"
	DEFAULT_CHECK_INTERVAL = "5m"

	IDLE_TIMEOUT_S        = 10 * time.Second
	READ_HEADER_TIMEOUT_S = 5 * time.Second
	READ_TIMEOUT_S        = 5 * time.Second
	SHUTDOWN_TIMEOUT_S    = 30 * time.Second
	WRITE_TIMEOUT_S       = 5 * time.Second
)

func runServer(server *http.Server, port string) {
	go func() {
		slog.Info("server starting", "port", port)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "err", err)
			os.Exit(1)
		}
		slog.Info("stopped serving new connections")
	}()
}

func healthLiveHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readinessHandler(ready *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if ready == nil || !ready.Load() {
			http.Error(w, "waiting for first successful public IP poll", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func panicRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.Error("panic recovered",
					"panic", p,
					"path", r.URL.Path,
					"stack", string(debug.Stack()))
				http.Error(w, fmt.Sprintf("internal server error: %v", p), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func parseDurationEnv(key, defaultVal string) time.Duration {
	s := config.GetEnv(key, defaultVal)
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

func main() {
	logger.Init(config.GetEnv("LOG_FORMAT", "text"))

	port := config.GetEnv("PORT", DEFAULT_SERVER_PORT)
	statePath := config.GetEnv("STATE_PATH", DEFAULT_STATE_PATH)
	pollInterval := parseDurationEnv("CHECK_INTERVAL", DEFAULT_CHECK_INTERVAL)

	ipURLs := publicip.ParseURLList(config.GetEnv("IP_URLS", ""))

	var ready atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", healthLiveHandler)
	mux.HandleFunc("/health/ready", readinessHandler(&ready))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           panicRecovery(mux),
		ReadHeaderTimeout: READ_HEADER_TIMEOUT_S,
		ReadTimeout:       READ_TIMEOUT_S,
		WriteTimeout:      WRITE_TIMEOUT_S,
		IdleTimeout:       IDLE_TIMEOUT_S,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runServer(server, port)

	go watcher.Run(ctx, watcher.Config{
		StatePath:    statePath,
		PollInterval: pollInterval,
		IPURLs:       ipURLs,
	}, &ready)

	<-ctx.Done()

	slog.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), SHUTDOWN_TIMEOUT_S)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown timeout, forcing close", "err", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("force close failed", "err", closeErr)
		}
	}
	slog.Info("graceful shutdown complete")
}
