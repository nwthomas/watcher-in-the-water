package publicip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

var DefaultURLs = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://checkip.amazonaws.com",
}

const (
	TIMEOUT_FETCH_S = 15 * time.Second
	MAX_BODY_BYTES  = 4096
)

type ipifyJSON struct {
	IP string `json:"ip"`
}

func ParseURLList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseJSONBody(body []byte) (string, error) {
	var v ipifyJSON
	if err := json.Unmarshal(body, &v); err != nil {
		return "", fmt.Errorf("json decode: %w", err)
	}
	return validateIPString(strings.TrimSpace(v.IP))
}

func parsePlainBody(body []byte) (string, error) {
	s := strings.TrimSpace(string(body))
	return validateIPString(s)
}

func validateIPString(s string) (string, error) {
	if s == "" {
		return "", errors.New("empty IP string")
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return "", fmt.Errorf("invalid IP: %q", s)
	}
	return ip.String(), nil
}

func fetchOne(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: TIMEOUT_FETCH_S}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "watcher-in-the-water/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Debug("close response body", "err", cerr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, MAX_BODY_BYTES))
	if err != nil {
		return "", err
	}
	body = bytes.TrimSpace(body)

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") || bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		return parseJSONBody(body)
	}

	return parsePlainBody(body)
}

func Fetch(ctx context.Context, client *http.Client, urls []string) (string, error) {
	if len(urls) == 0 {
		return "", errors.New("no IP fetch URLs configured")
	}
	var lastErr error
	for _, raw := range urls {
		ip, err := fetchOne(ctx, client, raw)
		if err == nil {
			return ip, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("all IP fetch URLs failed")
	}
	return "", fmt.Errorf("public IP fetch: %w", lastErr)
}
