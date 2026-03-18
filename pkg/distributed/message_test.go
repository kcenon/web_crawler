package distributed

import (
	"testing"
	"time"
)

func TestCrawlTask_EncodeDecode(t *testing.T) {
	original := &CrawlTask{
		ID:        "task-1",
		JobID:     "job-1",
		URL:       "https://example.com/page",
		Domain:    "example.com",
		Depth:     2,
		MaxDepth:  5,
		Priority:  10,
		Headers:   map[string]string{"Accept": "text/html"},
		Meta:      map[string]string{"source": "seed"},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := DecodeTask(data)
	if err != nil {
		t.Fatalf("DecodeTask failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, original.ID)
	}
	if decoded.URL != original.URL {
		t.Errorf("URL: got %q, want %q", decoded.URL, original.URL)
	}
	if decoded.Domain != original.Domain {
		t.Errorf("Domain: got %q, want %q", decoded.Domain, original.Domain)
	}
	if decoded.Depth != original.Depth {
		t.Errorf("Depth: got %d, want %d", decoded.Depth, original.Depth)
	}
	if decoded.Priority != original.Priority {
		t.Errorf("Priority: got %d, want %d", decoded.Priority, original.Priority)
	}
}

func TestDecodeTask_InvalidJSON(t *testing.T) {
	_, err := DecodeTask([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestControlMessage_EncodeDecode(t *testing.T) {
	original := &ControlMessage{
		Type:      ControlPause,
		JobID:     "job-1",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	data, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	decoded, err := DecodeControl(data)
	if err != nil {
		t.Fatalf("DecodeControl failed: %v", err)
	}

	if decoded.Type != ControlPause {
		t.Errorf("Type: got %q, want %q", decoded.Type, ControlPause)
	}
	if decoded.JobID != original.JobID {
		t.Errorf("JobID: got %q, want %q", decoded.JobID, original.JobID)
	}
}

func TestDecodeControl_InvalidJSON(t *testing.T) {
	_, err := DecodeControl([]byte("{invalid"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
