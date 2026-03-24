package main

import (
	"context"
	"errors"
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
	TIMEOUT_IDLE_S        = 10 * time.Second
	TIMEOUT_READ_HEADER_S = 5 * time.Second
	TIMEOUT_READ_S        = 5 * time.Second
	TIMEOUT_SHUTDOWN_S    = 30 * time.Second
	TIMEOUT_WRITE_S       = 5 * time.Second
)

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
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	cfg := config.LoadServerConfig()

	logger.Init(cfg.LogFormat, cfg.LogLevel)

	ipURLs := publicip.ParseURLList(cfg.IPURLs)

	var ready atomic.Bool

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", healthLiveHandler)
	mux.HandleFunc("/health/ready", readinessHandler(&ready))

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           panicRecovery(mux),
		ReadHeaderTimeout: TIMEOUT_READ_HEADER_S,
		ReadTimeout:       TIMEOUT_READ_S,
		WriteTimeout:      TIMEOUT_WRITE_S,
		IdleTimeout:       TIMEOUT_IDLE_S,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "err", err)
			os.Exit(1)
		}
		slog.Info("shutdown starting")
	}()

	go watcher.Run(ctx, watcher.Config{
		StatePath:    cfg.StatePath,
		PollInterval: cfg.CheckInterval,
		IPURLs:       ipURLs,
	}, &ready)

	<-ctx.Done()

	slog.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SHUTDOWN_S)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown timeout, forcing close", "err", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("force close failed", "err", closeErr)
		}
	}
	slog.Info("shutdown complete")
}
