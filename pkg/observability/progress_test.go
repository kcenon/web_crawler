package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProgressTracker_UpdateAndCurrent(t *testing.T) {
	tracker := NewProgressTracker(time.Second)

	event := ProgressEvent{
		RequestsTotal:  100,
		SuccessCount:   95,
		ErrorCount:     5,
		ActiveRequests: 10,
		URLsQueued:     50,
	}

	tracker.Update(event)

	current := tracker.Current()
	if current.RequestsTotal != 100 {
		t.Errorf("RequestsTotal: got %d, want 100", current.RequestsTotal)
	}
	if current.SuccessCount != 95 {
		t.Errorf("SuccessCount: got %d, want 95", current.SuccessCount)
	}
	if current.Timestamp.IsZero() {
		t.Error("expected non-zero Timestamp after Update")
	}
}

func TestProgressTracker_SubscribeReceivesUpdates(t *testing.T) {
	tracker := NewProgressTracker(time.Second)

	ch := tracker.subscribe()
	defer tracker.unsubscribe(ch)

	tracker.Update(ProgressEvent{RequestsTotal: 42})

	select {
	case event := <-ch:
		if event.RequestsTotal != 42 {
			t.Errorf("expected RequestsTotal=42, got %d", event.RequestsTotal)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestProgressTracker_ClientCount(t *testing.T) {
	tracker := NewProgressTracker(time.Second)

	if tracker.ClientCount() != 0 {
		t.Errorf("expected 0 clients, got %d", tracker.ClientCount())
	}

	ch1 := tracker.subscribe()
	ch2 := tracker.subscribe()

	if tracker.ClientCount() != 2 {
		t.Errorf("expected 2 clients, got %d", tracker.ClientCount())
	}

	tracker.unsubscribe(ch1)
	tracker.unsubscribe(ch2)

	if tracker.ClientCount() != 0 {
		t.Errorf("expected 0 clients after unsubscribe, got %d", tracker.ClientCount())
	}
}

func TestProgressTracker_SSEHandler(t *testing.T) {
	tracker := NewProgressTracker(time.Second)

	tracker.Update(ProgressEvent{
		RequestsTotal: 100,
		SuccessCount:  90,
	})

	// Use a context we can cancel to stop the SSE stream.
	ctx, cancel := context.WithCancel(context.Background())

	req := httptest.NewRequest("GET", "/progress", nil)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		tracker.SSEHandler().ServeHTTP(rec, req)
		close(done)
	}()

	// Give the handler time to write the initial event.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	result := rec.Result()
	defer result.Body.Close()

	if ct := result.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type: got %q, want text/event-stream", ct)
	}

	body := rec.Body.String()
	if !strings.HasPrefix(body, "data: ") {
		t.Errorf("expected SSE data prefix, got %q", body[:min(len(body), 50)])
	}

	// Parse the JSON data from the SSE line.
	dataLine := strings.TrimPrefix(strings.Split(body, "\n")[0], "data: ")
	var event ProgressEvent
	if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
		t.Fatalf("failed to parse SSE JSON: %v", err)
	}
	if event.RequestsTotal != 100 {
		t.Errorf("SSE event RequestsTotal: got %d, want 100", event.RequestsTotal)
	}
}

func TestProgressTracker_SSEHandler_StreamingNotSupported(t *testing.T) {
	tracker := NewProgressTracker(time.Second)

	req := httptest.NewRequest("GET", "/progress", nil)
	// nonFlushWriter doesn't implement http.Flusher.
	w := &nonFlushWriter{httptest.NewRecorder()}

	tracker.SSEHandler().ServeHTTP(w, req)

	if w.rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for non-flusher, got %d", w.rec.Code)
	}
}

// nonFlushWriter wraps ResponseRecorder but does NOT implement http.Flusher.
type nonFlushWriter struct {
	rec *httptest.ResponseRecorder
}

func (w *nonFlushWriter) Header() http.Header         { return w.rec.Header() }
func (w *nonFlushWriter) Write(b []byte) (int, error) { return w.rec.Write(b) }
func (w *nonFlushWriter) WriteHeader(code int)        { w.rec.WriteHeader(code) }

func TestRunBroadcastLoop_SendsPeriodicUpdates(t *testing.T) {
	tracker := NewProgressTracker(50 * time.Millisecond)
	tracker.Update(ProgressEvent{RequestsTotal: 10})

	ch := tracker.subscribe()
	defer tracker.unsubscribe(ch)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go tracker.RunBroadcastLoop(ctx)

	count := 0
	for {
		select {
		case <-ch:
			count++
			if count >= 2 {
				return // Success: got periodic broadcasts.
			}
		case <-ctx.Done():
			if count < 2 {
				t.Errorf("expected at least 2 broadcasts, got %d", count)
			}
			return
		}
	}
}
