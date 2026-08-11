package watcher

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nwthomas/watcher-in-the-water/internal/ipstate"
)

type recordingEmailNotifier struct {
	calls     int
	previous  string
	current   string
	updatedAt time.Time
	err       error
}

func (n *recordingEmailNotifier) NotifyIPChange(_ context.Context, previousIP, currentIP string, updatedAt time.Time) error {
	n.calls++
	n.previous = previousIP
	n.current = currentIP
	n.updatedAt = updatedAt
	return n.err
}

func TestPollOnce_seedsWhenEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.10"))
	}))
	defer srv.Close()

	var ready atomic.Bool
	notifier := &recordingEmailNotifier{}
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, notifier, &ready)

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
	if notifier.calls != 0 {
		t.Fatalf("email calls = %d, want 0", notifier.calls)
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
	notifier := &recordingEmailNotifier{}
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, notifier, &ready)
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
	if notifier.calls != 0 {
		t.Fatalf("email calls = %d, want 0", notifier.calls)
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
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, nil, &ready)

	st, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.PublicIP != "192.0.2.2" {
		t.Fatalf("got %q", st.PublicIP)
	}
}

func TestPollOnce_updatesOnChange_sendsEmail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := ipstate.Save(statePath, ipstate.State{PublicIP: "192.0.2.1", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.2"))
	}))
	defer ipSrv.Close()

	var ready atomic.Bool
	notifier := &recordingEmailNotifier{}
	pollOnce(context.Background(), &http.Client{}, []string{ipSrv.URL}, statePath, notifier, &ready)

	if notifier.calls != 1 {
		t.Fatalf("email calls = %d, want 1", notifier.calls)
	}
	if notifier.previous != "192.0.2.1" || notifier.current != "192.0.2.2" {
		t.Fatalf("email notification = previous %q current %q", notifier.previous, notifier.current)
	}
	if notifier.updatedAt.IsZero() {
		t.Fatal("expected updatedAt to be set")
	}
}

func TestPollOnce_updatesOnChange_keepsStateWhenEmailFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := ipstate.Save(statePath, ipstate.State{PublicIP: "192.0.2.1", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.2"))
	}))
	defer ipSrv.Close()

	var ready atomic.Bool
	notifier := &recordingEmailNotifier{err: errors.New("smtp down")}
	pollOnce(context.Background(), &http.Client{}, []string{ipSrv.URL}, statePath, notifier, &ready)

	st, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.PublicIP != "192.0.2.2" {
		t.Fatalf("stored ip = %q, want 192.0.2.2", st.PublicIP)
	}
	if notifier.calls != 1 {
		t.Fatalf("email calls = %d, want 1", notifier.calls)
	}
}
