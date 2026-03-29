package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNotifyIPChange_postsJSON(t *testing.T) {
	t.Parallel()
	var gotMethod string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	at := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	NotifyIPChange(context.Background(), &http.Client{}, []string{srv.URL}, "192.0.2.1", "192.0.2.2", at)

	if gotMethod != http.MethodPost {
		t.Fatalf("method = %q", gotMethod)
	}
	var p ChangePayload
	if err := json.Unmarshal(gotBody, &p); err != nil {
		t.Fatal(err)
	}
	if p.PreviousIP != "192.0.2.1" || p.CurrentIP != "192.0.2.2" {
		t.Fatalf("payload = %+v", p)
	}
	if !p.UpdatedAt.Equal(at.UTC()) {
		t.Fatalf("updated_at = %v", p.UpdatedAt)
	}
}

func TestNotifyIPChange_noURLs(t *testing.T) {
	t.Parallel()
	NotifyIPChange(context.Background(), &http.Client{}, nil, "a", "b", time.Now())
}

func TestNotifyIPChange_logsNon2xx(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer srv.Close()

	// Exercise error path; no panic.
	NotifyIPChange(context.Background(), &http.Client{Timeout: 5 * time.Second}, []string{srv.URL}, "1.1.1.1", "2.2.2.2", time.Now().UTC())
}

func TestChangePayload_roundTrip(t *testing.T) {
	t.Parallel()
	at := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	b, err := json.Marshal(ChangePayload{
		PreviousIP: "192.0.2.1",
		CurrentIP:  "192.0.2.2",
		UpdatedAt:  at,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"previous_ip"`)) {
		t.Fatalf("json: %s", b)
	}
}
