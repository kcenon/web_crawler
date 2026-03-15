package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// AuthType enumerates the supported authentication schemes.
type AuthType string

const (
	AuthTypeBasic  AuthType = "basic"  // HTTP Basic Authentication (RFC 7617).
	AuthTypeBearer AuthType = "bearer" // Bearer token (RFC 6750).
	AuthTypeOAuth2 AuthType = "oauth2" // OAuth 2.0 Client Credentials (RFC 6749 §4.4).
)

// OAuth2Config holds the parameters required for the OAuth2 Client Credentials flow.
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
}

// AuthConfig configures the authentication middleware.
type AuthConfig struct {
	// Type selects the authentication scheme. Required.
	Type AuthType

	// Basic Auth fields (used when Type == AuthTypeBasic).
	Username string
	Password string

	// Bearer token (used when Type == AuthTypeBearer).
	Token string

	// OAuth2 configuration (used when Type == AuthTypeOAuth2).
	OAuth2 *OAuth2Config

	// HTTPClient overrides the HTTP client used for OAuth2 token requests.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client
}

// oauth2Token holds a cached access token and its expiry time.
type oauth2Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"` // seconds; 0 means no expiry information.
	TokenType   string `json:"token_type"`

	expiry time.Time // computed from ExpiresIn at fetch time
}

// expired reports whether the token should be refreshed.
// A 10-second buffer is applied to account for clock skew.
func (t *oauth2Token) expired() bool {
	if t.expiry.IsZero() {
		return false
	}
	return time.Now().Add(10 * time.Second).After(t.expiry)
}

// Auth is a middleware that injects authentication credentials into each request.
//
//   - Basic: sets "Authorization: Basic <base64(user:pass)>" (RFC 7617).
//   - Bearer: sets "Authorization: Bearer <token>" (RFC 6750).
//   - OAuth2: fetches a token via the Client Credentials flow on first use and
//     automatically refreshes it when it expires (RFC 6749 §4.4).
type Auth struct {
	cfg        AuthConfig
	httpClient *http.Client

	mu    sync.Mutex   // guards cachedToken
	token *oauth2Token // OAuth2 cached token; nil until first request
}

// NewAuth creates an authentication middleware from cfg.
// Panics if cfg.Type is empty.
func NewAuth(cfg AuthConfig) *Auth {
	if cfg.Type == "" {
		panic("middleware.NewAuth: AuthConfig.Type must not be empty")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &Auth{cfg: cfg, httpClient: client}
}

// ProcessRequest implements Middleware. It attaches the appropriate authentication
// header to req before forwarding the request.
func (a *Auth) ProcessRequest(ctx context.Context, req *Request, next NextFunc) (*Response, error) {
	header, err := a.buildHeader(ctx)
	if err != nil {
		return nil, err
	}

	if req.Headers == nil {
		req.Headers = make(map[string]string)
	}
	req.Headers["Authorization"] = header

	return next(ctx, req)
}

// buildHeader returns the value for the "Authorization" header.
func (a *Auth) buildHeader(ctx context.Context) (string, error) {
	switch a.cfg.Type {
	case AuthTypeBasic:
		return basicAuthHeader(a.cfg.Username, a.cfg.Password), nil
	case AuthTypeBearer:
		return "Bearer " + a.cfg.Token, nil
	case AuthTypeOAuth2:
		return a.oauth2Header(ctx)
	default:
		return "", fmt.Errorf("middleware.Auth: unsupported auth type %q", a.cfg.Type)
	}
}

// basicAuthHeader constructs an HTTP Basic Auth header value per RFC 7617.
func basicAuthHeader(username, password string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return "Basic " + encoded
}

// oauth2Header returns "Bearer <access_token>", fetching or refreshing the token as needed.
func (a *Auth) oauth2Header(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token == nil || a.token.expired() {
		tok, err := a.fetchToken(ctx)
		if err != nil {
			return "", err
		}
		a.token = tok
	}

	return "Bearer " + a.token.AccessToken, nil
}

// fetchToken requests a new access token from the OAuth2 token endpoint using
// the Client Credentials grant (RFC 6749 §4.4).
func (a *Auth) fetchToken(ctx context.Context) (*oauth2Token, error) {
	cfg := a.cfg.OAuth2
	if cfg == nil {
		return nil, fmt.Errorf("middleware.Auth: OAuth2 config is nil")
	}

	// Build the standard token request body per RFC 6749 §4.4.
	// Field names are defined by the OAuth2 spec; values come from runtime config.
	body := url.Values{}
	body.Set("grant_type", "client_credentials")
	body.Set("client_id", cfg.ClientID)
	body.Set("client_secret", cfg.ClientSecret)
	if len(cfg.Scopes) > 0 {
		body.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL,
		strings.NewReader(body.Encode()))
	if err != nil {
		return nil, fmt.Errorf("middleware.Auth: building token request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("middleware.Auth: token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("middleware.Auth: token endpoint returned HTTP %d", resp.StatusCode)
	}

	var tok oauth2Token
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, fmt.Errorf("middleware.Auth: decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("middleware.Auth: token response missing access_token")
	}

	if tok.ExpiresIn > 0 {
		tok.expiry = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	}

	return &tok, nil
}
