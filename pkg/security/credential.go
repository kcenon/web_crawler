// Package security provides credential management utilities for safe handling
// of sensitive values such as passwords, tokens, and API keys.
package security

import (
	"fmt"
	"os"
)

// Credential holds a sensitive string value with safe printing behavior.
// Its String and GoString methods always return a redacted placeholder,
// preventing accidental credential leakage through fmt, logging, or
// debug output.
//
// Use [NewCredential] to create a credential from a literal value, or
// [CredentialFromEnv] to load one from an environment variable.
type Credential struct {
	value string
}

// NewCredential creates a Credential from a plaintext value.
func NewCredential(value string) Credential {
	return Credential{value: value}
}

// CredentialFromEnv loads a credential from the named environment variable.
// Returns an error if the variable is unset or empty.
func CredentialFromEnv(envVar string) (Credential, error) {
	v := os.Getenv(envVar)
	if v == "" {
		return Credential{}, fmt.Errorf("security: environment variable %s is not set or empty", envVar)
	}
	return Credential{value: v}, nil
}

// Value returns the plaintext credential. Use sparingly and only when the
// actual value is needed (e.g., constructing an HTTP header).
func (c Credential) Value() string {
	return c.value
}

// IsEmpty reports whether the credential holds an empty value.
func (c Credential) IsEmpty() bool {
	return c.value == ""
}

// String always returns "[REDACTED]" to prevent accidental logging.
// This implements fmt.Stringer.
func (c Credential) String() string {
	return "[REDACTED]"
}

// GoString always returns "[REDACTED]" to prevent exposure via %#v.
// This implements fmt.GoStringer.
func (c Credential) GoString() string {
	return "[REDACTED]"
}

// MarshalText returns "[REDACTED]" when the credential is serialized as text
// (e.g., JSON, YAML). This prevents accidental serialization of secrets.
func (c Credential) MarshalText() ([]byte, error) {
	return []byte("[REDACTED]"), nil
}
