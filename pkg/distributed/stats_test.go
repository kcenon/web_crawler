package distributed

import (
	"log/slog"
	"testing"
	"time"
)

// noopLogger returns a logger that discards all output.
func noopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestStats_Counters(t *testing.T) {
	s := NewStats()

	s.RecordProcessed()
	s.RecordProcessed()
	s.RecordSuccess()
	s.RecordError()
	s.RecordDLQ()

	snap := s.Snapshot()
	if snap.Processed != 2 {
		t.Errorf("Processed: got %d, want 2", snap.Processed)
	}
	if snap.Succeeded != 1 {
		t.Errorf("Succeeded: got %d, want 1", snap.Succeeded)
	}
	if snap.Failed != 1 {
		t.Errorf("Failed: got %d, want 1", snap.Failed)
	}
	if snap.DLQCount != 1 {
		t.Errorf("DLQCount: got %d, want 1", snap.DLQCount)
	}
}

func TestStats_AvgDuration(t *testing.T) {
	s := NewStats()

	s.RecordProcessed()
	s.RecordDuration(100 * time.Millisecond)
	s.RecordProcessed()
	s.RecordDuration(200 * time.Millisecond)

	snap := s.Snapshot()
	if snap.AvgDuration != 150*time.Millisecond {
		t.Errorf("AvgDuration: got %v, want 150ms", snap.AvgDuration)
	}
}

func TestStats_ZeroProcessed(t *testing.T) {
	s := NewStats()
	snap := s.Snapshot()

	if snap.AvgDuration != 0 {
		t.Errorf("AvgDuration with zero processed: got %v, want 0", snap.AvgDuration)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/page", "example.com"},
		{"http://sub.example.com:8080/path", "sub.example.com"},
		{"not-a-url", ""},
		{"https://example.com", "example.com"},
	}

	for _, tt := range tests {
		got := extractDomain(tt.url)
		if got != tt.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
