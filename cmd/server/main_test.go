package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/nwthomas/watcher-in-the-water/internal/config"
	"github.com/nwthomas/watcher-in-the-water/internal/email"
)

func TestHealthLiveHandler(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	healthLiveHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestReadinessHandler(t *testing.T) {
	t.Parallel()
	t.Run("nil ready acts as not ready", func(t *testing.T) {
		t.Parallel()
		h := readinessHandler(nil)
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", rec.Code)
		}
	})
	t.Run("not ready", func(t *testing.T) {
		t.Parallel()
		var ready atomic.Bool
		h := readinessHandler(&ready)
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", rec.Code)
		}
	})
	t.Run("ready", func(t *testing.T) {
		t.Parallel()
		var ready atomic.Bool
		ready.Store(true)
		h := readinessHandler(&ready)
		rec := httptest.NewRecorder()
		h(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
	})
}

func TestPanicRecovery(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	handler := panicRecovery(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "internal server error") {
		t.Fatalf("body = %q", body)
	}
}

func TestLoadEmailConfig(t *testing.T) {
	t.Parallel()
	complete := config.ServerConfig{
		EmailHostName:      "smtp.mail.me.com",
		EmailPassword:      "app-password",
		EmailPersonalEmail: "me@icloud.com",
		EmailPort:          587,
		EmailTLS:           true,
		EmailUsername:      "me@icloud.com",
	}
	incomplete := complete
	incomplete.EmailPassword = ""

	tests := []struct {
		name string
		cfg  config.ServerConfig
		want email.Config
	}{
		{name: "unset stays disabled", cfg: config.ServerConfig{EmailPort: 587, EmailTLS: true}, want: email.Config{}},
		{name: "incomplete is disabled", cfg: incomplete, want: email.Config{}},
		{
			name: "complete is enabled",
			cfg:  complete,
			want: email.Config{
				HostName:      "smtp.mail.me.com",
				Port:          587,
				TLS:           true,
				Username:      "me@icloud.com",
				Password:      "app-password",
				PersonalEmail: "me@icloud.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := loadEmailConfig(tt.cfg); got != tt.want {
				t.Fatalf("loadEmailConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
