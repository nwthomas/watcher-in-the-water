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
	"syscall"
	"time"

	_ "go.uber.org/automaxprocs"

	"github.com/nwthomas/bedlam/gateway/internal/config"
	"github.com/nwthomas/bedlam/gateway/internal/logger"
)

const (
	DEFAULT_SERVER_PORT = "8080"

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

func gracefulShutdown(server *http.Server, timeout time.Duration) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("shutdown timeout, forcing close", "err", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("force close failed", "err", closeErr)
		}
	}
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
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

func main() {
	logger.Init(config.GetEnv("LOG_FORMAT", "text"))

	port := config.GetEnv("PORT", DEFAULT_SERVER_PORT)

	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", healthCheckHandler)
	mux.HandleFunc("/health/ready", healthCheckHandler)

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           panicRecovery(mux),
		ReadHeaderTimeout: READ_HEADER_TIMEOUT_S,
		ReadTimeout:       READ_TIMEOUT_S,
		WriteTimeout:      WRITE_TIMEOUT_S,
		IdleTimeout:       IDLE_TIMEOUT_S,
	}

	runServer(server, port)
	gracefulShutdown(server, SHUTDOWN_TIMEOUT_S)
	slog.Info("graceful shutdown complete")
}
