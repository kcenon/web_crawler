package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/plugin"
)

func TestWebhook_Name(t *testing.T) {
	w := NewWebhook(WebhookConfig{URL: "http://example.com"})
	if w.Name() != "webhook" {
		t.Errorf("Name() = %q, want %q", w.Name(), "webhook")
	}
}

func TestWebhook_Init_RequiresURL(t *testing.T) {
	w := NewWebhook(WebhookConfig{})
	if err := w.Init(nil); err == nil {
		t.Error("expected error for missing URL")
	}
}

func TestWebhook_Init_URLFromConfig(t *testing.T) {
	w := NewWebhook(WebhookConfig{})
	if err := w.Init(map[string]any{"url": "http://example.com/hook"}); err != nil {
		t.Fatal(err)
	}
	if w.cfg.URL != "http://example.com/hook" {
		t.Errorf("URL = %q, want %q", w.cfg.URL, "http://example.com/hook")
	}
}

func TestWebhook_Notify_SendsJSON(t *testing.T) {
	var received webhookPayload
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{
		URL:     srv.URL,
		Headers: map[string]string{"X-Custom": "test-value"},
	})
	_ = w.Init(nil)

	event := &plugin.CrawlEvent{
		Type:    plugin.EventCompleted,
		Message: "crawl finished",
		Data:    map[string]any{"pages": 42},
	}

	if err := w.Notify(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	if received.Type != "completed" {
		t.Errorf("type = %q, want %q", received.Type, "completed")
	}
	if received.Message != "crawl finished" {
		t.Errorf("message = %q, want %q", received.Message, "crawl finished")
	}
	if received.SentAt.IsZero() {
		t.Error("expected non-zero sent_at")
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", receivedHeaders.Get("Content-Type"))
	}
	if receivedHeaders.Get("X-Custom") != "test-value" {
		t.Errorf("X-Custom = %q", receivedHeaders.Get("X-Custom"))
	}
}

func TestWebhook_Notify_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{URL: srv.URL})
	_ = w.Init(nil)

	event := &plugin.CrawlEvent{Type: plugin.EventError, Message: "test"}

	err := w.Notify(context.Background(), event)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestWebhook_Notify_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{URL: srv.URL, Timeout: 100 * time.Millisecond})
	_ = w.Init(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	event := &plugin.CrawlEvent{Type: plugin.EventStarted, Message: "start"}
	err := w.Notify(ctx, event)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestWebhook_Notify_AllEventTypes(t *testing.T) {
	types := []plugin.EventType{
		plugin.EventStarted,
		plugin.EventCompleted,
		plugin.EventError,
		plugin.EventThreshold,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := NewWebhook(WebhookConfig{URL: srv.URL})
	_ = w.Init(nil)

	for _, et := range types {
		event := &plugin.CrawlEvent{Type: et, Message: string(et)}
		if err := w.Notify(context.Background(), event); err != nil {
			t.Errorf("Notify(%s) error: %v", et, err)
		}
	}
}

func TestWebhook_Close(t *testing.T) {
	w := NewWebhook(WebhookConfig{URL: "http://example.com"})
	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

func TestWebhook_DefaultTimeout(t *testing.T) {
	w := NewWebhook(WebhookConfig{URL: "http://example.com"})
	if w.cfg.Timeout != 10*time.Second {
		t.Errorf("default timeout = %v, want 10s", w.cfg.Timeout)
	}
}
