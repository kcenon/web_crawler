package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogConfig{
		Level:  slog.LevelInfo,
		Format: "json",
		Output: &buf,
	})

	logger.Info("test message", "key", "value")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("expected valid JSON log output: %v", err)
	}

	if msg, ok := entry["msg"].(string); !ok || msg != "test message" {
		t.Errorf("expected msg=%q, got %v", "test message", entry["msg"])
	}

	if v, ok := entry["key"].(string); !ok || v != "value" {
		t.Errorf("expected key=%q, got %v", "value", entry["key"])
	}
}

func TestNewLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogConfig{
		Level:  slog.LevelInfo,
		Format: "text",
		Output: &buf,
	})

	logger.Info("hello text")

	output := buf.String()
	if len(output) == 0 {
		t.Fatal("expected non-empty text output")
	}

	// Text format should not be valid JSON
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err == nil {
		t.Error("expected non-JSON text output")
	}
}

func TestNewLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogConfig{
		Level:  slog.LevelWarn,
		Format: "json",
		Output: &buf,
	})

	logger.Info("should be filtered")
	if buf.Len() != 0 {
		t.Error("expected INFO message to be filtered at WARN level")
	}

	logger.Warn("should appear")
	if buf.Len() == 0 {
		t.Error("expected WARN message to appear")
	}
}

func TestNewLogger_DefaultOutput(t *testing.T) {
	// Should not panic with nil Output
	logger := NewLogger(LogConfig{})
	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestRequestLogFields(t *testing.T) {
	fields := RequestLogFields("req-123", "example.com", "https://example.com/page")

	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	expected := map[string]string{
		"request_id": "req-123",
		"domain":     "example.com",
		"url":        "https://example.com/page",
	}

	for _, attr := range fields {
		want, ok := expected[attr.Key]
		if !ok {
			t.Errorf("unexpected field key: %s", attr.Key)
			continue
		}
		if attr.Value.String() != want {
			t.Errorf("field %s: expected %q, got %q", attr.Key, want, attr.Value.String())
		}
	}
}
