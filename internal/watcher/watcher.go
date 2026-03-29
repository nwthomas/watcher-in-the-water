package watcher

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/nwthomas/watcher-in-the-water/internal/ipstate"
	"github.com/nwthomas/watcher-in-the-water/internal/publicip"
	"github.com/nwthomas/watcher-in-the-water/internal/webhook"
)

// Config drives the public IP polling loop.
type Config struct {
	StatePath    string
	PollInterval time.Duration
	IPURLs       []string
	WebhookURLs  []string
	HTTPClient   *http.Client
}

// Run polls public IP on an interval until ctx is cancelled. On first successful fetch,
// ready is set (if non-nil). Logs each successful check (unchanged or changed) and state errors.
func Run(ctx context.Context, cfg Config, ready *atomic.Bool) {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	urls := cfg.IPURLs
	if len(urls) == 0 {
		urls = publicip.DefaultURLs
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	pollOnce(ctx, client, urls, cfg.StatePath, cfg.WebhookURLs, ready)

	for {
		select {
		case <-ctx.Done():
			slog.Info("ip watcher stopped")
			return
		case <-ticker.C:
			pollOnce(ctx, client, urls, cfg.StatePath, cfg.WebhookURLs, ready)
		}
	}
}

func pollOnce(ctx context.Context, client *http.Client, urls []string, statePath string, webhookURLs []string, ready *atomic.Bool) {
	previous, err := ipstate.Load(statePath)
	if err != nil {
		slog.Error("load ip state", "path", statePath, "err", err)
	}

	ip, err := publicip.Fetch(ctx, client, urls)
	if err != nil {
		slog.Error("fetch public IP", "err", err)
		return
	}
	if ready != nil {
		ready.Store(true)
	}

	if previous.PublicIP == "" {
		next := ipstate.State{PublicIP: ip, UpdatedAt: time.Now().UTC()}
		if err := ipstate.Save(statePath, next); err != nil {
			slog.Error("save initial ip state", "path", statePath, "err", err)
			return
		}
		slog.Info("seeded public ip", "public_ip", ip)
		return
	}
	if previous.PublicIP == ip {
		slog.Info("public ip verified", "public_ip", ip, "changed", false)
		return
	}

	next := ipstate.State{PublicIP: ip, UpdatedAt: time.Now().UTC()}
	if err := ipstate.Save(statePath, next); err != nil {
		slog.Error("save ip state after change", "path", statePath, "err", err)
		return
	}

	slog.Info("public ip changed",
		"previous_ip", previous.PublicIP,
		"current_ip", ip,
	)
	webhook.NotifyIPChange(ctx, client, webhookURLs, previous.PublicIP, ip, next.UpdatedAt)
}
