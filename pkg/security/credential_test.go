package security

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewCredential(t *testing.T) {
	c := NewCredential("my-secret")
	if c.Value() != "my-secret" {
		t.Errorf("expected Value()=%q, got %q", "my-secret", c.Value())
	}
}

func TestCredential_IsEmpty(t *testing.T) {
	empty := NewCredential("")
	if !empty.IsEmpty() {
		t.Error("expected empty credential to report IsEmpty=true")
	}

	full := NewCredential("value")
	if full.IsEmpty() {
		t.Error("expected non-empty credential to report IsEmpty=false")
	}
}

func TestCredential_StringRedacted(t *testing.T) {
	c := NewCredential("super-secret-token")

	if c.String() != "[REDACTED]" {
		t.Errorf("String() should return [REDACTED], got %q", c.String())
	}

	if c.GoString() != "[REDACTED]" {
		t.Errorf("GoString() should return [REDACTED], got %q", c.GoString())
	}
}

func TestCredential_FmtRedacted(t *testing.T) {
	c := NewCredential("super-secret-token")

	tests := []struct {
		format string
		want   string
	}{
		{"%s", "[REDACTED]"},
		{"%v", "[REDACTED]"},
		{"%#v", "[REDACTED]"},
	}

	for _, tt := range tests {
		got := fmt.Sprintf(tt.format, c)
		if got != tt.want {
			t.Errorf("fmt.Sprintf(%q, cred) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestCredential_JSONRedacted(t *testing.T) {
	type config struct {
		APIKey Credential `json:"api_key"`
	}

	cfg := config{APIKey: NewCredential("sk-secret-123")}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	expected := `{"api_key":"[REDACTED]"}`
	if string(data) != expected {
		t.Errorf("JSON marshal should redact credential:\n got: %s\nwant: %s", data, expected)
	}
}

func TestCredentialFromEnv(t *testing.T) {
	t.Setenv("TEST_CRED_VAR", "env-secret")

	c, err := CredentialFromEnv("TEST_CRED_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Value() != "env-secret" {
		t.Errorf("expected Value()=%q, got %q", "env-secret", c.Value())
	}
}

func TestCredentialFromEnv_Unset(t *testing.T) {
	_, err := CredentialFromEnv("NONEXISTENT_VAR_12345")
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func TestCredentialFromEnv_Empty(t *testing.T) {
	t.Setenv("TEST_EMPTY_CRED", "")

	_, err := CredentialFromEnv("TEST_EMPTY_CRED")
	if err == nil {
		t.Fatal("expected error for empty env var")
	}
}
