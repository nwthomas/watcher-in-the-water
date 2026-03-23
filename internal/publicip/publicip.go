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

// DefaultURLs is the fallback chain when IP_URLS is empty.
var DefaultURLs = []string{
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
	"https://checkip.amazonaws.com",
}

const (
	defaultFetchTimeout = 15 * time.Second
	maxBodyBytes        = 4096
)

type ipifyJSON struct {
	IP string `json:"ip"`
}

// ParseURLList splits a comma-separated list of URLs, trims spaces, drops empties.
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

// Fetch returns the public IP using the first URL that succeeds.
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

func fetchOne(ctx context.Context, client *http.Client, rawURL string) (string, error) {
	if client == nil {
		client = &http.Client{Timeout: defaultFetchTimeout}
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
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
