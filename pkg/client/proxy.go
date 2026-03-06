package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
)

// ProxyConfig configures proxy server settings for the HTTP transport.
type ProxyConfig struct {
	// URL is the proxy server address. Supported schemes:
	// http://, https://, socks5://
	URL string

	// Username for proxy authentication (optional).
	Username string

	// Password for proxy authentication (optional).
	Password string
}

// applyProxy configures the transport to route requests through the proxy.
// For HTTP/HTTPS proxies it sets Transport.Proxy; for SOCKS5 it replaces
// the DialContext with a SOCKS5-aware dialer.
func applyProxy(transport *http.Transport, cfg ProxyConfig, dialTimeout time.Duration) error {
	if cfg.URL == "" {
		return nil
	}

	proxyURL, err := url.Parse(cfg.URL)
	if err != nil {
		return fmt.Errorf("parse proxy URL: %w", err)
	}

	// Embed credentials in the URL if provided.
	if cfg.Username != "" {
		proxyURL.User = url.UserPassword(cfg.Username, cfg.Password)
	}

	switch proxyURL.Scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(proxyURL)

	case "socks5":
		dialer, err := newSOCKS5Dialer(proxyURL, dialTimeout)
		if err != nil {
			return fmt.Errorf("create SOCKS5 dialer: %w", err)
		}
		transport.DialContext = dialer

	default:
		return fmt.Errorf("unsupported proxy scheme: %q (use http, https, or socks5)", proxyURL.Scheme)
	}

	return nil
}

// newSOCKS5Dialer creates a DialContext function that routes connections
// through a SOCKS5 proxy server.
func newSOCKS5Dialer(proxyURL *url.URL, dialTimeout time.Duration) (func(ctx context.Context, network, addr string) (net.Conn, error), error) {
	var auth *proxy.Auth
	if proxyURL.User != nil {
		pass, _ := proxyURL.User.Password()
		auth = &proxy.Auth{
			User:     proxyURL.User.Username(),
			Password: pass,
		}
	}

	forward := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	socksDialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, forward)
	if err != nil {
		return nil, err
	}

	// proxy.SOCKS5 returns a proxy.Dialer. Check if it also implements
	// proxy.ContextDialer for context-aware dialing.
	if ctxDialer, ok := socksDialer.(proxy.ContextDialer); ok {
		return ctxDialer.DialContext, nil
	}

	// Fallback: wrap the basic Dialer.
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return socksDialer.Dial(network, addr)
	}, nil
}
