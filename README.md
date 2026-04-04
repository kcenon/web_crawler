# Web Crawler SDK

A high-performance web crawling SDK with Go core engine and Python bindings.

## Overview

Web Crawler SDK provides a flexible and extensible framework for web crawling with:
- **Go Core Engine**: High-performance crawling with HTTP/1.1 and HTTP/2 support
- **Python Bindings**: Easy-to-use Python SDK via gRPC
- **JavaScript Rendering**: chromedp-based dynamic page rendering
- **Middleware System**: Extensible request/response processing pipeline
- **Plugin Architecture**: Customizable storage, export, and cache plugins

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Web Crawler SDK Architecture                  │
├─────────────────────────────────────────────────────────────────┤
│  Application Layer: Go SDK | Python SDK | CLI Tool              │
├─────────────────────────────────────────────────────────────────┤
│  Core Engine: HTTP Client | Browser | Scheduler | Extractor     │
├─────────────────────────────────────────────────────────────────┤
│  Middleware: Retry | Rate Limit | Robots.txt | Proxy | Auth     │
├─────────────────────────────────────────────────────────────────┤
│  Plugin Layer: Storage | Export | Cache | Custom                │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure: Redis | PostgreSQL | Kafka | Prometheus        │
└─────────────────────────────────────────────────────────────────┘
```

## Features

- HTTP/1.1 and HTTP/2 support
- JavaScript rendering with chromedp
- URL Frontier with priority queue
- Multiple extraction methods (CSS, XPath, JSON, Regex)
- Middleware chain pattern
- Rate limiting and politeness (robots.txt)
- Multiple storage backends (PostgreSQL, Redis, File)
- Prometheus metrics and structured logging

## Documentation

- [Product Requirements Document (PRD)](docs/PRD.md)
- [Software Requirements Specification (SRS)](docs/SRS.md)
- [Software Design Specification (SDS)](docs/SDS.md)

## Quick Start

### Go SDK

```go
package main

import (
    "context"
    "github.com/kcenon/web_crawler/pkg/crawler"
)

func main() {
    c, _ := crawler.New(
        crawler.WithConcurrency(10),
        crawler.WithRateLimit(5.0),
    )

    c.OnHTML("a[href]", func(e *crawler.HTMLElement) {
        link := e.Attr("href")
        c.AddURL(link)
    })

    c.AddURL("https://example.com")
    c.Start(context.Background())
    c.Wait()
}
```

### Python SDK

```python
from crawler import CrawlerClient

with CrawlerClient() as client:
    result = client.crawl("https://example.com")
    print(result.text)
```

### CLI

```bash
# Initialize a new project
crawler init my-project

# Crawl a single URL
crawler crawl https://example.com --render-js

# Start the gRPC server
crawler server start --port 50051
```

## Development

### Prerequisites

- Go 1.26+
- [golangci-lint](https://golangci-lint.run/) (for linting)

### Build Commands

```bash
make build    # Build the crawler binary to bin/crawler
make test     # Run all tests
make lint     # Run linters
make clean    # Remove build artifacts
```

### Project Structure

```
cmd/crawler/       CLI entry point
pkg/               Public Go packages
  crawler/         High-level crawling API
  client/          HTTP client
  browser/         Browser-based rendering
  frontier/        URL frontier management
  extractor/       Data extraction
  middleware/      Middleware chain
  storage/         Storage backends
  server/          gRPC server
  observability/   Logging and metrics
internal/          Internal packages
  scheduler/       Crawl scheduling
  pipeline/        Data pipeline
  util/            Shared utilities
api/proto/         Protobuf definitions
python/            Python SDK bindings
```

## Requirements

- Go 1.26+
- Python 3.9+ (for Python bindings)
- Chrome/Chromium (for JavaScript rendering)

## License

MIT License
