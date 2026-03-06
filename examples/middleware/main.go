// Package main demonstrates using the middleware chain with the web crawler SDK.
//
// This example shows how to build a middleware chain with rate limiting
// and retry, then execute requests through it.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kcenon/web_crawler/pkg/middleware"
)

func main() {
	// Terminal handler: simulates an actual HTTP request.
	handler := func(_ context.Context, req *middleware.Request) (*middleware.Response, error) {
		fmt.Printf("  -> Executing request: %s\n", req.URL)
		return &middleware.Response{
			StatusCode: 200,
			Body:       []byte("OK"),
		}, nil
	}

	// Build middleware chain with rate limiting and retry.
	chain := middleware.NewChain(handler)

	// Rate limiter: 2 RPS globally, 1 RPS per domain.
	chain.Use(middleware.NewRateLimit(middleware.RateLimitConfig{
		GlobalRPS:        2,
		DefaultDomainRPS: 1,
		BurstSize:        3,
	}))

	// Retry: up to 3 attempts with exponential backoff.
	chain.Use(middleware.NewRetry(middleware.RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		Multiplier:     2.0,
		JitterFactor:   0.1,
	}))

	// Execute requests through the chain.
	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.org/page1",
	}

	for _, u := range urls {
		req := &middleware.Request{
			URL:    u,
			Method: "GET",
		}

		fmt.Printf("Processing: %s\n", u)
		resp, err := chain.Execute(context.Background(), req)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		fmt.Printf("  <- Response: %d (%d bytes)\n\n", resp.StatusCode, len(resp.Body))
	}

	fmt.Printf("Chain has %d middleware(s)\n", chain.Len())
}
