// Package notifier provides built-in NotifierPlugin implementations.
package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kcenon/web_crawler/pkg/plugin"
)

// WebhookConfig configures the webhook notifier.
type WebhookConfig struct {
	// URL is the endpoint to POST event payloads to. Required.
	URL string

	// Timeout is the HTTP request timeout. Defaults to 10 seconds.
	Timeout time.Duration

	// Headers are extra HTTP headers sent with each request.
	Headers map[string]string
}

// Webhook sends crawl events as JSON POST requests to a configured URL.
type Webhook struct {
	cfg    WebhookConfig
	client *http.Client
}

// NewWebhook creates a webhook notifier with the given configuration.
func NewWebhook(cfg WebhookConfig) *Webhook {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &Webhook{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Name returns the plugin identifier.
func (w *Webhook) Name() string { return "webhook" }

// Init validates the configuration. The URL can be overridden via the
// config map key "url".
func (w *Webhook) Init(config map[string]any) error {
	if u, ok := config["url"].(string); ok && u != "" {
		w.cfg.URL = u
	}
	if w.cfg.URL == "" {
		return fmt.Errorf("webhook notifier: url is required")
	}
	return nil
}

// Close is a no-op; Webhook holds no persistent resources.
func (w *Webhook) Close() error { return nil }

// webhookPayload is the JSON body sent to the webhook endpoint.
type webhookPayload struct {
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data,omitempty"`
	SentAt  time.Time      `json:"sent_at"`
}

// Notify posts the crawl event as JSON to the configured webhook URL.
func (w *Webhook) Notify(ctx context.Context, event *plugin.CrawlEvent) error {
	payload := webhookPayload{
		Type:    string(event.Type),
		Message: event.Message,
		Data:    event.Data,
		SentAt:  time.Now(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook notifier: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook notifier: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook notifier: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook notifier: server returned %d", resp.StatusCode)
	}

	return nil
}

// Compile-time interface check.
var _ plugin.NotifierPlugin = (*Webhook)(nil)
