package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ProgressEvent represents a real-time crawl progress update.
type ProgressEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	RequestsTotal   int64     `json:"requests_total"`
	SuccessCount    int64     `json:"success_count"`
	ErrorCount      int64     `json:"error_count"`
	ActiveRequests  int64     `json:"active_requests"`
	URLsQueued      int64     `json:"urls_queued"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	RequestsPerSec  float64   `json:"requests_per_sec"`
}

// ProgressTracker collects crawl progress and broadcasts updates to
// connected SSE clients.
type ProgressTracker struct {
	mu       sync.RWMutex
	current  ProgressEvent
	clients  map[chan ProgressEvent]struct{}
	interval time.Duration
}

// NewProgressTracker creates a tracker that broadcasts at the given interval.
// Default interval: 1 second.
func NewProgressTracker(interval time.Duration) *ProgressTracker {
	if interval <= 0 {
		interval = time.Second
	}
	return &ProgressTracker{
		clients:  make(map[chan ProgressEvent]struct{}),
		interval: interval,
	}
}

// Update sets the latest progress snapshot. Call this periodically
// from the crawl engine.
func (t *ProgressTracker) Update(event ProgressEvent) {
	t.mu.Lock()
	event.Timestamp = time.Now()
	t.current = event
	t.mu.Unlock()

	t.broadcast(event)
}

// Current returns the latest progress snapshot.
func (t *ProgressTracker) Current() ProgressEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.current
}

// subscribe adds a client channel that receives progress events.
func (t *ProgressTracker) subscribe() chan ProgressEvent {
	ch := make(chan ProgressEvent, 8)
	t.mu.Lock()
	t.clients[ch] = struct{}{}
	t.mu.Unlock()
	return ch
}

// unsubscribe removes a client channel.
func (t *ProgressTracker) unsubscribe(ch chan ProgressEvent) {
	t.mu.Lock()
	delete(t.clients, ch)
	t.mu.Unlock()
	close(ch)
}

// broadcast sends a progress event to all connected clients.
func (t *ProgressTracker) broadcast(event ProgressEvent) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for ch := range t.clients {
		select {
		case ch <- event:
		default:
			// Drop event if client is too slow.
		}
	}
}

// ClientCount returns the number of connected SSE clients.
func (t *ProgressTracker) ClientCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.clients)
}

// SSEHandler returns an http.Handler that streams progress events
// as Server-Sent Events (SSE).
func (t *ProgressTracker) SSEHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := t.subscribe()
		defer t.unsubscribe(ch)

		// Send current state immediately.
		current := t.Current()
		writeSSE(w, flusher, current)

		ctx := r.Context()
		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}
				writeSSE(w, flusher, event)
			case <-ctx.Done():
				return
			}
		}
	})
}

// writeSSE writes a single SSE event to the response.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, event ProgressEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// RunBroadcastLoop periodically broadcasts the current progress to all
// connected SSE clients. It runs until the context is cancelled.
func (t *ProgressTracker) RunBroadcastLoop(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.broadcast(t.Current())
		case <-ctx.Done():
			return
		}
	}
}
