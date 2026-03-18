package observability

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// Test fixture values — intentionally obvious fakes to avoid
// false positives from secret scanners (e.g. GitGuardian).
const (
	fixtureDummyVal = "test-fixture-dummy-val"
)

// newTestLogger returns a sanitizing JSON logger writing to buf.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return NewLogger(LogConfig{
		Level:    slog.LevelInfo,
		Format:   "json",
		Output:   buf,
		Sanitize: true,
	})
}

// parseLogEntry decodes the first JSON log line from buf.
func parseLogEntry(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("invalid JSON log: %v\nraw: %s", err, buf.String())
	}
	return entry
}

func TestSanitize_BearerToken(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("auth header", "header", "Bearer "+fixtureDummyVal)

	entry := parseLogEntry(t, &buf)
	val := entry["header"].(string)
	if val != "Bearer [REDACTED]" {
		t.Errorf("expected Bearer token redacted, got %q", val)
	}
}

func TestSanitize_BasicAuth(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("auth header", "header", "Basic "+fixtureDummyVal)

	entry := parseLogEntry(t, &buf)
	val := entry["header"].(string)
	if val != "Basic [REDACTED]" {
		t.Errorf("expected Basic auth redacted, got %q", val)
	}
}

func TestSanitize_URLWithCredentials(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("proxy", "url", "http://admin:dummy@proxy.example.com:8080/path")

	entry := parseLogEntry(t, &buf)
	val := entry["url"].(string)
	expected := "http://[REDACTED]@proxy.example.com:8080/path"
	if val != expected {
		t.Errorf("expected URL credentials redacted:\n got: %q\nwant: %q", val, expected)
	}
}

func TestSanitize_SensitiveKeyNames(t *testing.T) {
	tests := []struct {
		key  string
		val  string
		want string
	}{
		{"password", fixtureDummyVal, "[REDACTED]"},
		{"api_key", fixtureDummyVal, "[REDACTED]"},
		{"Authorization", "Bearer xyz", "[REDACTED]"},
		{"db_password", fixtureDummyVal, "[REDACTED]"},
		{"x-api-key", fixtureDummyVal, "[REDACTED]"},
		{"client_secret", fixtureDummyVal, "[REDACTED]"},
		{"access_token", fixtureDummyVal, "[REDACTED]"},
		{"private_key", fixtureDummyVal, "[REDACTED]"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newTestLogger(&buf)

			logger.Info("test", tt.key, tt.val)

			entry := parseLogEntry(t, &buf)
			got := entry[tt.key].(string)
			if got != tt.want {
				t.Errorf("key %q: expected %q, got %q", tt.key, tt.want, got)
			}
		})
	}
}

func TestSanitize_SafeKeysUnchanged(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("safe data",
		"request_id", "req-123",
		"domain", "example.com",
		"status", "200",
	)

	entry := parseLogEntry(t, &buf)
	if entry["request_id"] != "req-123" {
		t.Errorf("request_id should not be redacted")
	}
	if entry["domain"] != "example.com" {
		t.Errorf("domain should not be redacted")
	}
}

func TestSanitize_MessageRedaction(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("connecting to http://user:dummy@db.example.com:5432/mydb")

	entry := parseLogEntry(t, &buf)
	msg := entry["msg"].(string)
	expected := "connecting to http://[REDACTED]@db.example.com:5432/mydb"
	if msg != expected {
		t.Errorf("message URL credentials not redacted:\n got: %q\nwant: %q", msg, expected)
	}
}

func TestSanitize_GroupAttributes(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSanitizingHandler(slog.NewJSONHandler(&buf, nil))
	logger := slog.New(handler)

	logger.Info("request",
		slog.Group("auth",
			slog.String("token", fixtureDummyVal),
			slog.String("type", "bearer"),
		),
	)

	entry := parseLogEntry(t, &buf)
	group := entry["auth"].(map[string]any)
	if group["token"] != "[REDACTED]" {
		t.Errorf("nested token should be redacted, got %q", group["token"])
	}
	if group["type"] != "bearer" {
		t.Errorf("non-sensitive nested key should be unchanged, got %q", group["type"])
	}
}

func TestSanitize_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSanitizingHandler(slog.NewJSONHandler(&buf, nil))
	child := handler.WithAttrs([]slog.Attr{
		slog.String("api_key", fixtureDummyVal),
		slog.String("service", "crawler"),
	})
	logger := slog.New(child)

	logger.Info("test")

	entry := parseLogEntry(t, &buf)
	if entry["api_key"] != "[REDACTED]" {
		t.Errorf("pre-set api_key should be redacted, got %q", entry["api_key"])
	}
	if entry["service"] != "crawler" {
		t.Errorf("safe pre-set key should be unchanged")
	}
}

func TestSanitize_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := NewSanitizingHandler(slog.NewJSONHandler(&buf, nil))
	child := handler.WithGroup("conn")
	logger := slog.New(child)

	logger.Info("connect", "password", fixtureDummyVal)

	entry := parseLogEntry(t, &buf)
	group := entry["conn"].(map[string]any)
	if group["password"] != "[REDACTED]" {
		t.Errorf("password in group should be redacted, got %q", group["password"])
	}
}

func TestSanitize_MultiplePatternsInOneValue(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.Info("config",
		"upstream", "http://user:dummy@api.example.com with Bearer "+fixtureDummyVal,
	)

	entry := parseLogEntry(t, &buf)
	val := entry["upstream"].(string)
	if val == "http://user:dummy@api.example.com with Bearer "+fixtureDummyVal {
		t.Errorf("both URL creds and Bearer token should be redacted, got %q", val)
	}
}

func TestSanitize_Disabled(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(LogConfig{
		Level:    slog.LevelInfo,
		Format:   "json",
		Output:   &buf,
		Sanitize: false,
	})

	logger.Info("auth", "password", "visible-value")

	entry := parseLogEntry(t, &buf)
	if entry["password"] != "visible-value" {
		t.Errorf("sanitization disabled: password should be visible, got %q", entry["password"])
	}
}
