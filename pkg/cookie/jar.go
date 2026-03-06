// Package cookie provides cookie management with optional persistence
// for the web crawler SDK.
//
// The Jar type wraps Go's net/http/cookiejar with additional capabilities
// including JSON-based persistence, thread-safe access, and programmatic
// clearing. It satisfies the http.CookieJar interface for direct use
// with http.Client.
package cookie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// Jar defines the cookie management interface extending http.CookieJar
// with persistence and clearing capabilities.
type Jar interface {
	http.CookieJar
	Clear()
	Save(path string) error
	Load(path string) error
}

// entry represents a cookie stored for persistence.
type entry struct {
	Name     string    `json:"name"`
	Value    string    `json:"value"`
	Domain   string    `json:"domain"`
	Path     string    `json:"path"`
	Expires  time.Time `json:"expires,omitempty"`
	Secure   bool      `json:"secure,omitempty"`
	HttpOnly bool      `json:"http_only,omitempty"`
	SameSite string    `json:"same_site,omitempty"`
	RawURL   string    `json:"raw_url"`
}

// cookieFile is the JSON structure persisted to disk.
type cookieFile struct {
	Version int     `json:"version"`
	Cookies []entry `json:"cookies"`
}

// jar implements Jar by wrapping net/http/cookiejar with persistence.
type jar struct {
	mu      sync.RWMutex
	inner   *cookiejar.Jar
	entries []entry
}

// New creates a new Jar with an empty cookie store.
func New() (Jar, error) {
	inner, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	return &jar{inner: inner}, nil
}

// SetCookies stores cookies for the given URL. It delegates to the
// underlying stdlib jar for domain/path scoping and also records
// cookies in the persistence store.
func (j *jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.inner.SetCookies(u, cookies)
	j.trackCookies(u, cookies)
}

// Cookies returns cookies appropriate for the given URL, respecting
// domain, path, Secure, and expiry rules via the stdlib jar.
func (j *jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.inner.Cookies(u)
}

// Clear removes all stored cookies.
func (j *jar) Clear() {
	j.mu.Lock()
	defer j.mu.Unlock()

	inner, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		// publicsuffix.List never causes errors; this is defensive.
		return
	}
	j.inner = inner
	j.entries = nil
}

// Save persists cookies to a JSON file at the given path.
// Expired cookies are excluded from the output.
func (j *jar) Save(path string) error {
	j.mu.RLock()
	active := j.activeEntries()
	j.mu.RUnlock()

	data := cookieFile{
		Version: 1,
		Cookies: active,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cookies: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(path, b, 0o600); err != nil {
		return fmt.Errorf("write cookie file: %w", err)
	}

	return nil
}

// Load reads cookies from a JSON file and replays them into the jar.
// Existing cookies are cleared before loading.
func (j *jar) Load(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read cookie file: %w", err)
	}

	var data cookieFile
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("unmarshal cookies: %w", err)
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	// Reset the jar.
	inner, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return fmt.Errorf("create cookie jar: %w", err)
	}
	j.inner = inner
	j.entries = nil

	now := time.Now()
	for _, e := range data.Cookies {
		// Skip expired cookies.
		if !e.Expires.IsZero() && e.Expires.Before(now) {
			continue
		}

		u, err := url.Parse(e.RawURL)
		if err != nil {
			continue
		}

		cookie := entryToCookie(e)
		j.inner.SetCookies(u, []*http.Cookie{cookie})
		j.entries = append(j.entries, e)
	}

	return nil
}

// trackCookies updates the persistence store with the given cookies.
// Must be called with j.mu held.
func (j *jar) trackCookies(u *url.URL, cookies []*http.Cookie) {
	for _, c := range cookies {
		e := cookieToEntry(u, c)

		// Remove existing cookie with same identity.
		j.entries = removeCookieEntry(j.entries, e.Domain, e.Path, e.Name)
		j.entries = append(j.entries, e)
	}
}

// activeEntries returns entries that are not expired.
// Must be called with j.mu held for reading.
func (j *jar) activeEntries() []entry {
	now := time.Now()
	var active []entry
	for _, e := range j.entries {
		if !e.Expires.IsZero() && e.Expires.Before(now) {
			continue
		}
		active = append(active, e)
	}
	return active
}

// cookieToEntry converts an http.Cookie to a persistence entry.
func cookieToEntry(u *url.URL, c *http.Cookie) entry {
	domain := c.Domain
	if domain == "" {
		domain = u.Hostname()
	}
	path := c.Path
	if path == "" {
		path = "/"
	}

	var sameSite string
	switch c.SameSite {
	case http.SameSiteStrictMode:
		sameSite = "Strict"
	case http.SameSiteLaxMode:
		sameSite = "Lax"
	case http.SameSiteNoneMode:
		sameSite = "None"
	}

	rawURL := fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path)

	return entry{
		Name:     c.Name,
		Value:    c.Value,
		Domain:   domain,
		Path:     path,
		Expires:  c.Expires,
		Secure:   c.Secure,
		HttpOnly: c.HttpOnly,
		SameSite: sameSite,
		RawURL:   rawURL,
	}
}

// entryToCookie converts a persistence entry back to an http.Cookie.
func entryToCookie(e entry) *http.Cookie {
	c := &http.Cookie{
		Name:     e.Name,
		Value:    e.Value,
		Domain:   e.Domain,
		Path:     e.Path,
		Expires:  e.Expires,
		Secure:   e.Secure,
		HttpOnly: e.HttpOnly,
	}

	switch e.SameSite {
	case "Strict":
		c.SameSite = http.SameSiteStrictMode
	case "Lax":
		c.SameSite = http.SameSiteLaxMode
	case "None":
		c.SameSite = http.SameSiteNoneMode
	}

	return c
}

// removeCookieEntry removes a cookie entry by identity (domain+path+name).
func removeCookieEntry(entries []entry, domain, path, name string) []entry {
	n := 0
	for _, e := range entries {
		if e.Domain == domain && e.Path == path && e.Name == name {
			continue
		}
		entries[n] = e
		n++
	}
	return entries[:n]
}
