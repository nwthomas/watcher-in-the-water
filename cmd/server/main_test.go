package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestParseDurationEnv(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		def    string
		want   time.Duration
		unset  bool
	}{
		{
			name:   "uses env when valid",
			envVal: "10m",
			def:    "5m",
			want:   10 * time.Minute,
		},
		{
			name:  "uses default when unset",
			unset: true,
			def:   "7m",
			want:  7 * time.Minute,
		},
		{
			name:   "invalid env falls back to default string",
			envVal: "not-a-duration",
			def:    "3m",
			want:   3 * time.Minute,
		},
		{
			name:   "invalid env and invalid default yields 5m",
			envVal: "bad",
			def:    "also-bad",
			want:   5 * time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := "TEST_PARSE_DURATION_" + strings.ReplaceAll(tt.name, " ", "_")
			if tt.unset {
				t.Setenv(key, "")
			} else {
				t.Setenv(key, tt.envVal)
			}
			got := parseDurationEnv(key, tt.def)
			if got != tt.want {
				t.Fatalf("parseDurationEnv = %v, want %v", got, tt.want)
			}
		})
	}
}

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
