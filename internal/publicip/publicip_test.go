package publicip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseURLList(t *testing.T) {
	t.Parallel()
	got := ParseURLList(" https://a.test , https://b.test , ")
	if len(got) != 2 || got[0] != "https://a.test" || got[1] != "https://b.test" {
		t.Fatalf("ParseURLList = %#v", got)
	}
	if len(ParseURLList("")) != 0 {
		t.Fatal("empty input should yield empty slice")
	}
}

func TestFetch_plainText(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("  203.0.113.10 \n"))
	}))
	defer srv.Close()

	client := &http.Client{}
	ip, err := Fetch(context.Background(), client, []string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if ip != "203.0.113.10" {
		t.Fatalf("ip = %q", ip)
	}
}

func TestFetch_json(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ip":"2001:db8::1"}`))
	}))
	defer srv.Close()

	client := &http.Client{}
	ip, err := Fetch(context.Background(), client, []string{srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if ip != "2001:db8::1" {
		t.Fatalf("ip = %q", ip)
	}
}

func TestFetch_fallback(t *testing.T) {
	t.Parallel()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("198.51.100.2"))
	}))
	defer good.Close()

	client := &http.Client{}
	ip, err := Fetch(context.Background(), client, []string{bad.URL, good.URL})
	if err != nil {
		t.Fatal(err)
	}
	if ip != "198.51.100.2" {
		t.Fatalf("ip = %q", ip)
	}
}

func TestFetch_invalidBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-an-ip"))
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), &http.Client{}, []string{srv.URL})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "public IP fetch") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestFetch_emptyURLs(t *testing.T) {
	t.Parallel()
	_, err := Fetch(context.Background(), &http.Client{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
