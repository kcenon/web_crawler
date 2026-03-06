package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// TransportConfig controls HTTP transport behavior including connection
// pooling, timeouts, and protocol negotiation.
type TransportConfig struct {
	// MaxIdleConns controls the maximum total number of idle connections
	// across all hosts. Default: 100.
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum idle connections per host.
	// Default: 10.
	MaxIdleConnsPerHost int

	// MaxConnsPerHost limits the total connections per host including
	// dialing, active, and idle. Zero means no limit.
	MaxConnsPerHost int

	// DialTimeout limits the time to establish a TCP connection.
	// Default: 10s.
	DialTimeout time.Duration

	// TLSHandshakeTimeout limits the time for the TLS handshake.
	// Default: 10s.
	TLSHandshakeTimeout time.Duration

	// ResponseHeaderTimeout limits the time waiting for response headers
	// after sending the request. Default: 15s.
	ResponseHeaderTimeout time.Duration

	// IdleConnTimeout is the maximum time an idle connection will remain
	// idle before closing itself. Default: 90s.
	IdleConnTimeout time.Duration

	// DisableHTTP2, if true, prevents HTTP/2 protocol negotiation.
	// HTTP/2 is enabled by default.
	DisableHTTP2 bool

	// DisableCompression, if true, prevents requesting gzip/deflate
	// encoding. Compression is enabled by default.
	DisableCompression bool
}

func (c TransportConfig) withDefaults() TransportConfig {
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 100
	}
	if c.MaxIdleConnsPerHost == 0 {
		c.MaxIdleConnsPerHost = 10
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = 10 * time.Second
	}
	if c.TLSHandshakeTimeout == 0 {
		c.TLSHandshakeTimeout = 10 * time.Second
	}
	if c.ResponseHeaderTimeout == 0 {
		c.ResponseHeaderTimeout = 15 * time.Second
	}
	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = 90 * time.Second
	}
	return c
}

// buildTransport creates an *http.Transport from the configuration.
func buildTransport(cfg TransportConfig) *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
		ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		MaxIdleConns:          cfg.MaxIdleConns,
		MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:       cfg.MaxConnsPerHost,
		IdleConnTimeout:       cfg.IdleConnTimeout,
		ForceAttemptHTTP2:     !cfg.DisableHTTP2,
		DisableCompression:    cfg.DisableCompression,
	}
}
