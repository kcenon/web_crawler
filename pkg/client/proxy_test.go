package client

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestApplyProxy_Empty(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{}, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.Proxy != nil {
		t.Error("expected nil Proxy for empty config")
	}
}

func TestApplyProxy_HTTPProxy(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{
		URL: "http://proxy.example.com:8080",
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.Proxy == nil {
		t.Fatal("expected Proxy to be set")
	}

	// Verify the proxy function returns the correct URL.
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func error: %v", err)
	}
	if proxyURL.Host != "proxy.example.com:8080" {
		t.Errorf("expected proxy host, got %q", proxyURL.Host)
	}
}

func TestApplyProxy_HTTPProxyWithAuth(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{
		URL:      "http://proxy.example.com:8080",
		Username: "user",
		Password: "pass",
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	proxyURL, _ := transport.Proxy(req)
	if proxyURL.User == nil {
		t.Fatal("expected user info in proxy URL")
	}
	if proxyURL.User.Username() != "user" {
		t.Errorf("expected username 'user', got %q", proxyURL.User.Username())
	}
	pass, _ := proxyURL.User.Password()
	if pass != "pass" {
		t.Errorf("expected password 'pass', got %q", pass)
	}
}

func TestApplyProxy_SOCKS5(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{
		URL: "socks5://localhost:1080",
	}, 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transport.DialContext == nil {
		t.Error("expected DialContext to be set for SOCKS5")
	}
}

func TestApplyProxy_UnsupportedScheme(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{
		URL: "ftp://proxy.example.com",
	}, 10*time.Second)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
	if !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestApplyProxy_InvalidURL(t *testing.T) {
	transport := &http.Transport{}
	err := applyProxy(transport, ProxyConfig{
		URL: "://invalid",
	}, 10*time.Second)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestBuildTransport_WithProxy(t *testing.T) {
	cfg := TransportConfig{
		Proxy: ProxyConfig{
			URL: "http://proxy.example.com:8080",
		},
	}.withDefaults()

	transport, err := buildTransport(cfg)
	if err != nil {
		t.Fatalf("buildTransport error: %v", err)
	}
	if transport.Proxy == nil {
		t.Error("expected proxy to be configured")
	}
}

func TestBuildTransport_WithInvalidProxy(t *testing.T) {
	cfg := TransportConfig{
		Proxy: ProxyConfig{
			URL: "ftp://invalid-scheme",
		},
	}.withDefaults()

	_, err := buildTransport(cfg)
	if err == nil {
		t.Fatal("expected error for invalid proxy scheme")
	}
}

func TestHTTPProxy_Integration(t *testing.T) {
	// Create a target server.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello from target"))
	}))
	defer target.Close()

	// Create a simple HTTP CONNECT proxy.
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			// CONNECT tunnel for HTTPS.
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack not supported", http.StatusInternalServerError)
				return
			}
			clientConn, _, err := hijacker.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer clientConn.Close()

			targetConn, err := net.DialTimeout("tcp", r.Host, 5*time.Second)
			if err != nil {
				return
			}
			defer targetConn.Close()

			_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
			go func() { _, _ = io.Copy(targetConn, clientConn) }()
			_, _ = io.Copy(clientConn, targetConn)
			return
		}

		// Forward non-CONNECT requests (plain HTTP proxy).
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}))
	defer proxyServer.Close()

	proxyURL, _ := url.Parse(proxyServer.URL)

	// Create client with proxy.
	c, err := New(Config{
		Transport: TransportConfig{
			Proxy: ProxyConfig{URL: proxyURL.String()},
		},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer c.Close()

	resp, err := c.Do(context.Background(), &Request{
		URL:    target.URL,
		Method: "GET",
	})
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if string(resp.Body) != "hello from target" {
		t.Errorf("expected body 'hello from target', got %q", string(resp.Body))
	}
}

func TestProxyConfig_Defaults(t *testing.T) {
	// ProxyConfig with empty URL should not affect transport.
	cfg := TransportConfig{
		Proxy: ProxyConfig{},
	}.withDefaults()

	transport, err := buildTransport(cfg)
	if err != nil {
		t.Fatalf("buildTransport error: %v", err)
	}
	if transport.Proxy != nil {
		t.Error("expected nil proxy for empty config")
	}
}
