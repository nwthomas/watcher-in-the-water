package watcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nwthomas/watcher-in-the-water/internal/ipstate"
)

func TestPollOnce_seedsWhenEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.10"))
	}))
	defer srv.Close()

	var ready atomic.Bool
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, &ready)

	if !ready.Load() {
		t.Fatal("expected ready after successful poll")
	}
	st, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.PublicIP != "192.0.2.10" {
		t.Fatalf("stored ip = %q", st.PublicIP)
	}
}

func TestPollOnce_noChangeWhenSame(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := ipstate.Save(statePath, ipstate.State{PublicIP: "192.0.2.20", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.20"))
	}))
	defer srv.Close()

	var ready atomic.Bool
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, &ready)
	if !ready.Load() {
		t.Fatal("expected ready")
	}
	before, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	// UpdatedAt should be unchanged path — we return early without save; verify IP still same
	if before.PublicIP != "192.0.2.20" {
		t.Fatalf("unexpected mutation: %+v", before)
	}
}

func TestPollOnce_updatesOnChange(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := ipstate.Save(statePath, ipstate.State{PublicIP: "192.0.2.1", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.2"))
	}))
	defer srv.Close()

	var ready atomic.Bool
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, &ready)

	st, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.PublicIP != "192.0.2.2" {
		t.Fatalf("got %q", st.PublicIP)
	}
}
