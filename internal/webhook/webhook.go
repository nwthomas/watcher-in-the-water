package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const maxBodyLog = 512

// ChangePayload is the JSON body POSTed to each webhook when the public IP changes.
type ChangePayload struct {
	PreviousIP string    `json:"previous_ip"`
	CurrentIP  string    `json:"current_ip"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NotifyIPChange POSTs application/json to each URL with the change details.
// Errors are logged; failures do not affect other URLs.
func NotifyIPChange(ctx context.Context, client *http.Client, urls []string, previousIP, currentIP string, updatedAt time.Time) {
	if len(urls) == 0 {
		return
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	payload := ChangePayload{
		PreviousIP: previousIP,
		CurrentIP:  currentIP,
		UpdatedAt:  updatedAt.UTC(),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("webhook marshal payload", "err", err)
		return
	}

	for _, rawURL := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
		if err != nil {
			slog.Error("webhook build request", "url", rawURL, "err", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "watcher-in-the-water/1.0")

		resp, err := client.Do(req)
		if err != nil {
			slog.Error("webhook request failed", "url", rawURL, "err", err)
			continue
		}
		func() {
			defer func() {
				if cerr := resp.Body.Close(); cerr != nil {
					slog.Debug("webhook close body", "url", rawURL, "err", cerr)
				}
			}()
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				snippet, rerr := io.ReadAll(io.LimitReader(resp.Body, maxBodyLog))
				if rerr != nil {
					slog.Debug("webhook read error body", "url", rawURL, "err", rerr)
				}
				slog.Error("webhook unexpected status",
					"url", rawURL,
					"status", resp.StatusCode,
					"body_prefix", string(snippet),
				)
			} else {
				if _, cerr := io.Copy(io.Discard, resp.Body); cerr != nil {
					slog.Debug("webhook drain body", "url", rawURL, "err", cerr)
				}
				slog.Info("webhook delivered", "url", rawURL, "status", resp.StatusCode)
			}
		}()
	}
}
