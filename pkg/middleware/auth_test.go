package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Placeholder credential constants used throughout auth tests.
// These are clearly non-production dummy values.
const (
	testUsername     = "testuser"
	testPassword     = "testpassword"
	testClientID     = "test-client-id"
	testClientCred   = "test-client-cred"
	testBearerToken  = "test-bearer-token"
	testChainToken   = "test-chain-bearer"
	testAccessToken  = "access-token-from-srv"
	testExpiredToken = "expired-access-token"
)

// --- Basic Auth ---

func TestAuth_Basic_SetsHeader(t *testing.T) {
	req := &Request{URL: "http://example.com"}
	a := NewAuth(AuthConfig{
		Type:     AuthTypeBasic,
		Username: testUsername,
		Password: testPassword,
	})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	got := req.Headers["Authorization"]
	// base64("testuser:testpassword") = "dGVzdHVzZXI6dGVzdHBhc3N3b3Jk"
	if got != "Basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk" {
		t.Errorf("Authorization = %q, want %q", got, "Basic dGVzdHVzZXI6dGVzdHBhc3N3b3Jk")
	}
}

func TestAuth_Basic_EmptyPassword(t *testing.T) {
	req := &Request{URL: "http://example.com"}
	a := NewAuth(AuthConfig{
		Type:     AuthTypeBasic,
		Username: "user",
		Password: "",
	})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	got := req.Headers["Authorization"]
	if !strings.HasPrefix(got, "Basic ") {
		t.Errorf("Authorization = %q, want Basic prefix", got)
	}
}

func TestAuth_Basic_InitialisesNilHeaderMap(t *testing.T) {
	req := &Request{URL: "http://example.com", Headers: nil}
	a := NewAuth(AuthConfig{Type: AuthTypeBasic, Username: "u", Password: "p"})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req.Headers == nil {
		t.Error("Headers map must be initialised by middleware")
	}
	if !strings.HasPrefix(req.Headers["Authorization"], "Basic ") {
		t.Error("Basic auth header not set")
	}
}

// --- Bearer ---

func TestAuth_Bearer_SetsHeader(t *testing.T) {
	req := &Request{URL: "http://example.com"}
	a := NewAuth(AuthConfig{
		Type:  AuthTypeBearer,
		Token: testBearerToken,
	})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	got := req.Headers["Authorization"]
	if got != "Bearer "+testBearerToken {
		t.Errorf("Authorization = %q, want %q", got, "Bearer "+testBearerToken)
	}
}

func TestAuth_Bearer_InitialisesNilHeaderMap(t *testing.T) {
	req := &Request{URL: "http://example.com", Headers: nil}
	a := NewAuth(AuthConfig{Type: AuthTypeBearer, Token: "tok"})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}
	if req.Headers == nil {
		t.Error("Headers map must be initialised by middleware")
	}
}

// --- OAuth2 ---

// newTokenServer returns a test HTTP server that responds to POST requests with
// an OAuth2 token response. calls tracks how many times the endpoint is hit.
func newTokenServer(t *testing.T, calls *int, expiresIn int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("token server: method = %q, want POST", r.Method)
		}
		*calls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"access_token": testAccessToken,
			"token_type":   "Bearer",
			"expires_in":   expiresIn,
		})
	}))
}

func TestAuth_OAuth2_FetchesToken(t *testing.T) {
	calls := 0
	srv := newTokenServer(t, &calls, 3600)
	defer srv.Close()

	req := &Request{URL: "http://example.com"}
	a := NewAuth(AuthConfig{
		Type: AuthTypeOAuth2,
		OAuth2: &OAuth2Config{
			ClientID:     testClientID,
			ClientSecret: testClientCred,
			TokenURL:     srv.URL,
		},
		HTTPClient: srv.Client(),
	})

	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	if req.Headers["Authorization"] != "Bearer "+testAccessToken {
		t.Errorf("Authorization = %q, want %q", req.Headers["Authorization"], "Bearer "+testAccessToken)
	}
	if calls != 1 {
		t.Errorf("token server called %d times, want 1", calls)
	}
}

func TestAuth_OAuth2_CachesToken(t *testing.T) {
	calls := 0
	srv := newTokenServer(t, &calls, 3600) // long-lived token
	defer srv.Close()

	a := NewAuth(AuthConfig{
		Type: AuthTypeOAuth2,
		OAuth2: &OAuth2Config{
			ClientID:     testClientID,
			ClientSecret: testClientCred,
			TokenURL:     srv.URL,
		},
		HTTPClient: srv.Client(),
	})

	for range 5 {
		req := &Request{URL: "http://example.com"}
		if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
			t.Fatal(err)
		}
	}

	// Token should be fetched only once; subsequent requests use the cache.
	if calls != 1 {
		t.Errorf("token server called %d times after 5 requests, want 1 (cached)", calls)
	}
}

func TestAuth_OAuth2_RefreshesExpiredToken(t *testing.T) {
	calls := 0
	srv := newTokenServer(t, &calls, 0) // expires_in=0 → no expiry info → will not auto-expire
	defer srv.Close()

	a := NewAuth(AuthConfig{
		Type: AuthTypeOAuth2,
		OAuth2: &OAuth2Config{
			ClientID:     testClientID,
			ClientSecret: testClientCred,
			TokenURL:     srv.URL,
		},
		HTTPClient: srv.Client(),
	})

	// Manually inject an already-expired cached token.
	a.mu.Lock()
	a.token = &oauth2Token{
		AccessToken: testExpiredToken,
		expiry:      time.Now().Add(-time.Minute), // already expired
	}
	a.mu.Unlock()

	req := &Request{URL: "http://example.com"}
	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	if req.Headers["Authorization"] != "Bearer "+testAccessToken {
		t.Errorf("Authorization = %q, want refreshed token", req.Headers["Authorization"])
	}
	if calls != 1 {
		t.Errorf("token server called %d times, want 1 (refresh)", calls)
	}
}

func TestAuth_OAuth2_TokenEndpointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := NewAuth(AuthConfig{
		Type: AuthTypeOAuth2,
		OAuth2: &OAuth2Config{
			ClientID:     testClientID,
			ClientSecret: "invalid-cred",
			TokenURL:     srv.URL,
		},
		HTTPClient: srv.Client(),
	})

	req := &Request{URL: "http://example.com"}
	_, err := a.ProcessRequest(context.Background(), req, nopHandler)
	if err == nil {
		t.Error("expected error on 401 from token endpoint, got nil")
	}
}

func TestAuth_OAuth2_ScopesIncluded(t *testing.T) {
	var receivedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		receivedBody = r.FormValue("scope")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{ //nolint:errcheck
			"access_token": "tok",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	a := NewAuth(AuthConfig{
		Type: AuthTypeOAuth2,
		OAuth2: &OAuth2Config{
			ClientID:     testClientID,
			ClientSecret: testClientCred,
			TokenURL:     srv.URL,
			Scopes:       []string{"read", "write"},
		},
		HTTPClient: srv.Client(),
	})

	req := &Request{URL: "http://example.com"}
	if _, err := a.ProcessRequest(context.Background(), req, nopHandler); err != nil {
		t.Fatal(err)
	}

	if receivedBody != "read write" {
		t.Errorf("scope = %q, want %q", receivedBody, "read write")
	}
}

func TestAuth_OAuth2_NilConfig(t *testing.T) {
	a := NewAuth(AuthConfig{Type: AuthTypeOAuth2, OAuth2: nil})
	req := &Request{URL: "http://example.com"}
	_, err := a.ProcessRequest(context.Background(), req, nopHandler)
	if err == nil {
		t.Error("expected error for nil OAuth2 config, got nil")
	}
}

func TestAuth_Panic_EmptyType(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty AuthType, got none")
		}
	}()
	NewAuth(AuthConfig{}) // should panic
}

func TestAuth_IntegrationWithChain(t *testing.T) {
	var capturedHeader string
	handler := func(_ context.Context, req *Request) (*Response, error) {
		capturedHeader = req.Headers["Authorization"]
		return &Response{StatusCode: 200}, nil
	}

	c := NewChain(handler)
	c.Use(NewAuth(AuthConfig{
		Type:  AuthTypeBearer,
		Token: testChainToken,
	}))

	_, err := c.Execute(context.Background(), &Request{URL: "http://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if capturedHeader != "Bearer "+testChainToken {
		t.Errorf("handler received Authorization = %q, want %q", capturedHeader, "Bearer "+testChainToken)
	}
}
