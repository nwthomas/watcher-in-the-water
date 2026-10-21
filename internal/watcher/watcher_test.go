package watcher

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nwthomas/watcher-in-the-water/internal/email"
	"github.com/nwthomas/watcher-in-the-water/internal/ipstate"
	"github.com/nwthomas/watcher-in-the-water/internal/webhook"
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
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, nil, email.Config{}, &ready)

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
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, nil, email.Config{}, &ready)
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
	pollOnce(context.Background(), &http.Client{}, []string{srv.URL}, statePath, nil, email.Config{}, &ready)

	st, err := ipstate.Load(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if st.PublicIP != "192.0.2.2" {
		t.Fatalf("got %q", st.PublicIP)
	}
}

func TestPollOnce_updatesOnChange_callsWebhooks(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := ipstate.Save(statePath, ipstate.State{PublicIP: "192.0.2.1", UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}

	var gotMethod string
	var gotBody []byte
	wh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer wh.Close()

	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("192.0.2.2"))
	}))
	defer ipSrv.Close()

	var ready atomic.Bool
	pollOnce(context.Background(), &http.Client{}, []string{ipSrv.URL}, statePath, []string{wh.URL}, email.Config{}, &ready)

	if gotMethod != http.MethodPost {
		t.Fatalf("webhook method = %q", gotMethod)
	}
	var p webhook.ChangePayload
	if err := json.Unmarshal(gotBody, &p); err != nil {
		t.Fatal(err)
	}
	if p.PreviousIP != "192.0.2.1" || p.CurrentIP != "192.0.2.2" {
		t.Fatalf("payload = %+v", p)
	}
}

func TestPollOnce_emailsOnlyOnChange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		storedIP    string
		fetchedIP   string
		wantConnect bool
	}{
		{name: "seed does not email", storedIP: "", fetchedIP: "192.0.2.1", wantConnect: false},
		{name: "unchanged does not email", storedIP: "192.0.2.1", fetchedIP: "192.0.2.1", wantConnect: false},
		{name: "change emails", storedIP: "192.0.2.1", fetchedIP: "192.0.2.2", wantConnect: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			statePath := filepath.Join(t.TempDir(), "state.json")
			if tt.storedIP != "" {
				if err := ipstate.Save(statePath, ipstate.State{PublicIP: tt.storedIP, UpdatedAt: time.Now().UTC()}); err != nil {
					t.Fatal(err)
				}
			}

			// Count SMTP connection attempts; closing immediately makes the send fail fast.
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = ln.Close() }()
			var connections atomic.Int32
			go func() {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					connections.Add(1)
					_ = conn.Close()
				}
			}()
			_, port, _ := net.SplitHostPort(ln.Addr().String())
			portNum, _ := strconv.Atoi(port)
			emailCfg := email.Config{
				HostName:      "127.0.0.1",
				Port:          portNum,
				Username:      "me@icloud.com",
				Password:      "app-password",
				PersonalEmail: "me@icloud.com",
			}

			ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tt.fetchedIP))
			}))
			defer ipSrv.Close()

			var ready atomic.Bool
			pollOnce(context.Background(), &http.Client{}, []string{ipSrv.URL}, statePath, nil, emailCfg, &ready)

			if got := connections.Load() > 0; got != tt.wantConnect {
				t.Fatalf("smtp connected = %v, want %v", got, tt.wantConnect)
			}
		})
	}
}
