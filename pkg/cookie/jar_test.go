package cookie

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestNew(t *testing.T) {
	j, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if j == nil {
		t.Fatal("expected non-nil jar")
	}
}

func TestSetAndGetCookies(t *testing.T) {
	j, _ := New()
	u := mustParseURL("https://example.com/path")

	j.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc123", Path: "/"},
		{Name: "lang", Value: "en", Path: "/"},
	})

	cookies := j.Cookies(u)
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	found := map[string]string{}
	for _, c := range cookies {
		found[c.Name] = c.Value
	}
	if found["session"] != "abc123" {
		t.Errorf("expected session=abc123, got %q", found["session"])
	}
	if found["lang"] != "en" {
		t.Errorf("expected lang=en, got %q", found["lang"])
	}
}

func TestDomainScoping(t *testing.T) {
	j, _ := New()

	// Set cookie for example.com.
	u1 := mustParseURL("https://example.com/")
	j.SetCookies(u1, []*http.Cookie{
		{Name: "a", Value: "1", Path: "/"},
	})

	// Set cookie for other.com.
	u2 := mustParseURL("https://other.com/")
	j.SetCookies(u2, []*http.Cookie{
		{Name: "b", Value: "2", Path: "/"},
	})

	// example.com should only see its own cookie.
	cookies := j.Cookies(u1)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie for example.com, got %d", len(cookies))
	}
	if cookies[0].Name != "a" {
		t.Errorf("expected cookie 'a', got %q", cookies[0].Name)
	}

	// other.com should only see its own cookie.
	cookies = j.Cookies(u2)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie for other.com, got %d", len(cookies))
	}
	if cookies[0].Name != "b" {
		t.Errorf("expected cookie 'b', got %q", cookies[0].Name)
	}
}

func TestSecureCookieNotSentOverHTTP(t *testing.T) {
	j, _ := New()

	// Set a Secure cookie via HTTPS.
	httpsURL := mustParseURL("https://example.com/")
	j.SetCookies(httpsURL, []*http.Cookie{
		{Name: "token", Value: "secret", Secure: true, Path: "/"},
	})

	// Should be available over HTTPS.
	cookies := j.Cookies(httpsURL)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie over HTTPS, got %d", len(cookies))
	}

	// Should NOT be available over HTTP.
	httpURL := mustParseURL("http://example.com/")
	cookies = j.Cookies(httpURL)
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies over HTTP for Secure cookie, got %d", len(cookies))
	}
}

func TestExpiredCookieRemoval(t *testing.T) {
	j, _ := New()
	u := mustParseURL("https://example.com/")

	// Set a cookie that is already expired.
	j.SetCookies(u, []*http.Cookie{
		{Name: "old", Value: "gone", Path: "/", Expires: time.Now().Add(-1 * time.Hour)},
	})

	// The stdlib jar should not return expired cookies.
	cookies := j.Cookies(u)
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies (expired), got %d", len(cookies))
	}
}

func TestClear(t *testing.T) {
	j, _ := New()
	u := mustParseURL("https://example.com/")

	j.SetCookies(u, []*http.Cookie{
		{Name: "a", Value: "1", Path: "/"},
	})

	if len(j.Cookies(u)) == 0 {
		t.Fatal("expected cookies before clear")
	}

	j.Clear()

	if len(j.Cookies(u)) != 0 {
		t.Error("expected no cookies after clear")
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")

	// Create jar and add cookies.
	j1, _ := New()
	u := mustParseURL("https://example.com/")
	j1.SetCookies(u, []*http.Cookie{
		{Name: "session", Value: "abc", Path: "/", Expires: time.Now().Add(24 * time.Hour)},
		{Name: "pref", Value: "dark", Path: "/settings"},
	})

	// Save to file.
	if err := j1.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists and is readable JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty cookie file")
	}

	// Load into a new jar.
	j2, _ := New()
	if err := j2.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cookies := j2.Cookies(u)
	if len(cookies) < 1 {
		t.Fatalf("expected cookies after load, got %d", len(cookies))
	}

	found := false
	for _, c := range cookies {
		if c.Name == "session" && c.Value == "abc" {
			found = true
		}
	}
	if !found {
		t.Error("expected session cookie after load")
	}
}

func TestSaveExcludesExpired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")

	j, _ := New()
	u := mustParseURL("https://example.com/")

	// Add a fresh cookie and a (logically) expired one.
	j.SetCookies(u, []*http.Cookie{
		{Name: "fresh", Value: "yes", Path: "/", Expires: time.Now().Add(24 * time.Hour)},
	})
	// Manually inject an expired entry for testing.
	jImpl := j.(*jar)
	jImpl.mu.Lock()
	jImpl.entries = append(jImpl.entries, entry{
		Name:    "stale",
		Value:   "no",
		Domain:  "example.com",
		Path:    "/",
		Expires: time.Now().Add(-1 * time.Hour),
		RawURL:  "https://example.com/",
	})
	jImpl.mu.Unlock()

	if err := j.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify only fresh cookie exists.
	j2, _ := New()
	if err := j2.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cookies := j2.Cookies(u)
	for _, c := range cookies {
		if c.Name == "stale" {
			t.Error("expected expired cookie to be excluded from save")
		}
	}
}

func TestLoadSkipsExpired(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")

	// Manually write a cookie file with an expired cookie.
	content := `{
  "version": 1,
  "cookies": [
    {
      "name": "valid",
      "value": "yes",
      "domain": "example.com",
      "path": "/",
      "expires": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339) + `",
      "raw_url": "https://example.com/"
    },
    {
      "name": "expired",
      "value": "no",
      "domain": "example.com",
      "path": "/",
      "expires": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
      "raw_url": "https://example.com/"
    }
  ]
}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	j, _ := New()
	if err := j.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	u := mustParseURL("https://example.com/")
	cookies := j.Cookies(u)

	for _, c := range cookies {
		if c.Name == "expired" {
			t.Error("expected expired cookie to be skipped during load")
		}
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	j, _ := New()
	err := j.Load("/nonexistent/path/cookies.json")
	if err == nil {
		t.Error("expected error loading non-existent file")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "cookies.json")

	j, _ := New()
	if err := j.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}

func TestCookieUpdate(t *testing.T) {
	j, _ := New()
	u := mustParseURL("https://example.com/")

	// Set initial cookie.
	j.SetCookies(u, []*http.Cookie{
		{Name: "token", Value: "v1", Path: "/"},
	})

	// Update with new value.
	j.SetCookies(u, []*http.Cookie{
		{Name: "token", Value: "v2", Path: "/"},
	})

	cookies := j.Cookies(u)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if cookies[0].Value != "v2" {
		t.Errorf("expected updated value v2, got %q", cookies[0].Value)
	}
}

func TestConcurrentAccess(t *testing.T) {
	j, _ := New()
	u := mustParseURL("https://example.com/")

	var wg sync.WaitGroup
	const goroutines = 50

	// Concurrent writes.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			j.SetCookies(u, []*http.Cookie{
				{Name: "c", Value: "v", Path: "/"},
			})
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = j.Cookies(u)
		}()
	}

	wg.Wait()

	// Should still have exactly one cookie.
	cookies := j.Cookies(u)
	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie after concurrent access, got %d", len(cookies))
	}
}

func TestSameSitePersistence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")

	j1, _ := New()
	u := mustParseURL("https://example.com/")
	j1.SetCookies(u, []*http.Cookie{
		{Name: "strict", Value: "s", Path: "/", SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(time.Hour)},
		{Name: "lax", Value: "l", Path: "/", SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(time.Hour)},
	})

	if err := j1.Save(path); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	j2, _ := New()
	if err := j2.Load(path); err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	cookies := j2.Cookies(u)
	if len(cookies) < 2 {
		t.Fatalf("expected at least 2 cookies, got %d", len(cookies))
	}
}

func TestPathScoping(t *testing.T) {
	j, _ := New()

	u := mustParseURL("https://example.com/admin/")
	j.SetCookies(u, []*http.Cookie{
		{Name: "admin", Value: "yes", Path: "/admin"},
	})

	// Should be visible under /admin/.
	cookies := j.Cookies(mustParseURL("https://example.com/admin/dashboard"))
	if len(cookies) != 1 {
		t.Errorf("expected 1 cookie for /admin/dashboard, got %d", len(cookies))
	}

	// Should NOT be visible under /.
	cookies = j.Cookies(mustParseURL("https://example.com/"))
	if len(cookies) != 0 {
		t.Errorf("expected 0 cookies for /, got %d", len(cookies))
	}
}
