# Web Crawler SDK Technical Stack

> **Version**: 2.0.0
> **Last Updated**: 2026-02-05
> **Strategy**: Go Core + Python Bindings (Strategy C)
> **Purpose**: Technical foundation for building a high-performance, production-ready crawler SDK

## Overview

This SDK adopts **Strategy C**: a Go core engine with Python bindings. This approach delivers 5x+ performance improvement over pure Python while maintaining accessibility for data scientists through a Pythonic API.

---

## 1. Architecture Decision

### 1.1 Why Go Core + Python Bindings?

| Factor | Go Core | Python Only | Benefit |
|--------|---------|-------------|---------|
| **HTTP Performance** | 50,000 req/s | 10,000 req/s | 5x throughput |
| **Memory Usage** | ~100MB | ~500MB | 80% reduction |
| **Concurrency** | 100K goroutines | GIL limited | True parallelism |
| **Deployment** | Single binary | venv + deps | Zero dependency |
| **Python Access** | gRPC bindings | Native | Full ecosystem |

### 1.2 Architecture Layers

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           User Applications                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────────────┐  ┌─────────────────────────────────┐   │
│  │      Python SDK             │  │         Go SDK                   │   │
│  │  • crawler_sdk package      │  │  • pkg/crawler package           │   │
│  │  • Type hints (PEP 484)     │  │  • Interfaces & structs          │   │
│  │  • Async/await support      │  │  • Direct API access             │   │
│  └──────────────┬──────────────┘  └──────────────┬──────────────────┘   │
│                 │                                 │                      │
│                 │ gRPC / CGO                      │ Direct               │
│                 │                                 │                      │
│  ┌──────────────▼─────────────────────────────────▼──────────────────┐  │
│  │                        Go Core Engine                              │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │  │
│  │  │ HTTP Client │ │  Scheduler  │ │URL Frontier │ │Rate Limiter │  │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘  │  │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐  │  │
│  │  │   Parser    │ │ Middleware  │ │   Plugin    │ │  Storage    │  │  │
│  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Go Core Technologies

### 2.1 Core Libraries Overview

| Library | Purpose | Stars | Maturity |
|---------|---------|-------|----------|
| **[Colly](https://github.com/gocolly/colly)** | Web scraping framework | 23K+ | Production |
| **[chromedp](https://github.com/chromedp/chromedp)** | Chrome DevTools Protocol | 11K+ | Production |
| **[GoQuery](https://github.com/PuerkitoBio/goquery)** | jQuery-like HTML parsing | 14K+ | Production |
| **[Rod](https://github.com/go-rod/rod)** | Browser automation | 5K+ | Production |
| **net/http** | HTTP client (stdlib) | - | Stable |
| **golang.org/x/net/html** | HTML parser (stdlib) | - | Stable |

### 2.2 Colly Framework (Primary)

```go
package main

import (
    "fmt"
    "github.com/gocolly/colly/v2"
)

func main() {
    // Create collector with configuration
    c := colly.NewCollector(
        colly.AllowedDomains("example.com"),
        colly.MaxDepth(3),
        colly.Async(true),
    )

    // Configure rate limiting
    c.Limit(&colly.LimitRule{
        DomainGlob:  "*",
        Parallelism: 4,
        Delay:       1 * time.Second,
    })

    // Handle HTML elements
    c.OnHTML("a[href]", func(e *colly.HTMLElement) {
        link := e.Attr("href")
        fmt.Printf("Link found: %s\n", link)
        c.Visit(e.Request.AbsoluteURL(link))
    })

    // Handle responses
    c.OnResponse(func(r *colly.Response) {
        fmt.Printf("Visited: %s [%d]\n", r.Request.URL, r.StatusCode)
    })

    // Handle errors
    c.OnError(func(r *colly.Response, err error) {
        fmt.Printf("Error: %s - %v\n", r.Request.URL, err)
    })

    // Start crawling
    c.Visit("https://example.com")
    c.Wait()
}
```

**Colly Features**:
- Automatic cookie and session handling
- Automatic request delays and parallelism
- Built-in caching
- Distributed crawling with Redis
- Proxy rotation support

### 2.3 chromedp for JavaScript Rendering

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/chromedp/chromedp"
)

func main() {
    // Create context with timeout
    ctx, cancel := chromedp.NewContext(context.Background())
    defer cancel()

    ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    var html string
    var title string

    // Run browser automation
    err := chromedp.Run(ctx,
        // Navigate to page
        chromedp.Navigate("https://example.com"),

        // Wait for element to be visible
        chromedp.WaitVisible(`#content`, chromedp.ByID),

        // Scroll to trigger lazy loading
        chromedp.Evaluate(`window.scrollTo(0, document.body.scrollHeight)`, nil),

        // Wait for dynamic content
        chromedp.Sleep(2*time.Second),

        // Extract data
        chromedp.Title(&title),
        chromedp.OuterHTML(`body`, &html, chromedp.ByQuery),
    )

    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Title: %s\n", title)
    log.Printf("HTML length: %d\n", len(html))
}
```

### 2.4 GoQuery for HTML Parsing

```go
package main

import (
    "fmt"
    "log"
    "strings"

    "github.com/PuerkitoBio/goquery"
)

type Product struct {
    Name  string
    Price string
    URL   string
}

func parseProducts(html string) ([]Product, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return nil, err
    }

    var products []Product

    doc.Find(".product-item").Each(func(i int, s *goquery.Selection) {
        product := Product{
            Name:  strings.TrimSpace(s.Find(".product-title").Text()),
            Price: strings.TrimSpace(s.Find(".product-price").Text()),
            URL:   s.Find("a").AttrOr("href", ""),
        }
        products = append(products, product)
    })

    return products, nil
}
```

---

## 3. Concurrent Crawling Architecture

### 3.1 Worker Pool Pattern

```go
package crawler

import (
    "context"
    "sync"
)

type WorkerPool struct {
    workers    int
    urlChan    chan string
    resultChan chan *CrawlResult
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerPool{
        workers:    workers,
        urlChan:    make(chan string, 1000),
        resultChan: make(chan *CrawlResult, 1000),
        ctx:        ctx,
        cancel:     cancel,
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()

    for {
        select {
        case <-p.ctx.Done():
            return
        case url, ok := <-p.urlChan:
            if !ok {
                return
            }
            result := p.crawl(url)
            p.resultChan <- result
        }
    }
}

func (p *WorkerPool) crawl(url string) *CrawlResult {
    // Implement actual crawling logic
    return &CrawlResult{URL: url}
}

func (p *WorkerPool) Submit(url string) {
    p.urlChan <- url
}

func (p *WorkerPool) Results() <-chan *CrawlResult {
    return p.resultChan
}

func (p *WorkerPool) Stop() {
    p.cancel()
    close(p.urlChan)
    p.wg.Wait()
    close(p.resultChan)
}
```

### 3.2 Rate Limiter

```go
package ratelimit

import (
    "context"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

type DomainRateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rps      float64
    burst    int
}

func NewDomainRateLimiter(requestsPerSecond float64, burst int) *DomainRateLimiter {
    return &DomainRateLimiter{
        limiters: make(map[string]*rate.Limiter),
        rps:      requestsPerSecond,
        burst:    burst,
    }
}

func (d *DomainRateLimiter) getLimiter(domain string) *rate.Limiter {
    d.mu.RLock()
    limiter, exists := d.limiters[domain]
    d.mu.RUnlock()

    if exists {
        return limiter
    }

    d.mu.Lock()
    defer d.mu.Unlock()

    // Double-check after acquiring write lock
    if limiter, exists = d.limiters[domain]; exists {
        return limiter
    }

    limiter = rate.NewLimiter(rate.Limit(d.rps), d.burst)
    d.limiters[domain] = limiter
    return limiter
}

func (d *DomainRateLimiter) Wait(ctx context.Context, domain string) error {
    return d.getLimiter(domain).Wait(ctx)
}

func (d *DomainRateLimiter) Allow(domain string) bool {
    return d.getLimiter(domain).Allow()
}
```

---

## 4. Python Bindings

### 4.1 Binding Options Comparison

| Method | Performance | Complexity | Use Case |
|--------|-------------|------------|----------|
| **gRPC** | High | Medium | Distributed, microservices |
| **CGO** | Highest | High | Single binary, embedded |
| **REST API** | Medium | Low | Simple integration |
| **Unix Socket** | High | Low | Local communication |

### 4.2 gRPC Service Definition

```protobuf
// crawler.proto
syntax = "proto3";

package crawler;

option go_package = "github.com/yourorg/crawler-sdk/pkg/grpc";

service CrawlerService {
    // Single URL crawling
    rpc Crawl(CrawlRequest) returns (CrawlResponse);

    // Batch crawling
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);

    // Start continuous crawling job
    rpc StartJob(JobRequest) returns (JobResponse);

    // Get job status
    rpc GetJobStatus(JobStatusRequest) returns (JobStatusResponse);

    // Stop crawling job
    rpc StopJob(StopJobRequest) returns (StopJobResponse);
}

message CrawlRequest {
    string url = 1;
    CrawlOptions options = 2;
}

message CrawlOptions {
    bool render_js = 1;
    int32 timeout_seconds = 2;
    map<string, string> headers = 3;
    string proxy = 4;
    int32 max_retries = 5;
}

message CrawlResponse {
    string url = 1;
    int32 status_code = 2;
    string content = 3;
    map<string, string> headers = 4;
    int64 fetch_time_ms = 5;
    string error = 6;
}

message CrawlBatchRequest {
    repeated string urls = 1;
    CrawlOptions options = 2;
    int32 concurrency = 3;
}

message JobRequest {
    repeated string seed_urls = 1;
    CrawlOptions options = 2;
    JobConfig config = 3;
}

message JobConfig {
    int32 max_pages = 1;
    int32 max_depth = 2;
    repeated string allowed_domains = 3;
    float requests_per_second = 4;
}

message JobResponse {
    string job_id = 1;
    string status = 2;
}

message JobStatusRequest {
    string job_id = 1;
}

message JobStatusResponse {
    string job_id = 1;
    string status = 2;
    int64 pages_crawled = 3;
    int64 pages_queued = 4;
    int64 errors = 5;
}

message StopJobRequest {
    string job_id = 1;
}

message StopJobResponse {
    bool success = 1;
    string message = 2;
}
```

### 4.3 Python gRPC Client

```python
# crawler_sdk/client.py
from __future__ import annotations

import grpc
from typing import Iterator, Optional, Dict, List
from dataclasses import dataclass

from .generated import crawler_pb2, crawler_pb2_grpc


@dataclass
class CrawlResult:
    """Result of a crawl operation"""
    url: str
    status_code: int
    content: str
    headers: Dict[str, str]
    fetch_time_ms: int
    error: Optional[str] = None

    @property
    def success(self) -> bool:
        return self.error is None and 200 <= self.status_code < 400


@dataclass
class CrawlOptions:
    """Options for crawl requests"""
    render_js: bool = False
    timeout_seconds: int = 30
    headers: Optional[Dict[str, str]] = None
    proxy: Optional[str] = None
    max_retries: int = 3


class CrawlerClient:
    """Python client for Go crawler SDK"""

    def __init__(self, host: str = "localhost", port: int = 50051):
        self.channel = grpc.insecure_channel(f"{host}:{port}")
        self.stub = crawler_pb2_grpc.CrawlerServiceStub(self.channel)

    def crawl(self, url: str, options: Optional[CrawlOptions] = None) -> CrawlResult:
        """Crawl a single URL"""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlRequest(
            url=url,
            options=crawler_pb2.CrawlOptions(
                render_js=options.render_js,
                timeout_seconds=options.timeout_seconds,
                headers=options.headers or {},
                proxy=options.proxy or "",
                max_retries=options.max_retries,
            )
        )

        response = self.stub.Crawl(request)

        return CrawlResult(
            url=response.url,
            status_code=response.status_code,
            content=response.content,
            headers=dict(response.headers),
            fetch_time_ms=response.fetch_time_ms,
            error=response.error if response.error else None,
        )

    def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10
    ) -> Iterator[CrawlResult]:
        """Crawl multiple URLs with streaming results"""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlBatchRequest(
            urls=urls,
            options=crawler_pb2.CrawlOptions(
                render_js=options.render_js,
                timeout_seconds=options.timeout_seconds,
                headers=options.headers or {},
                proxy=options.proxy or "",
                max_retries=options.max_retries,
            ),
            concurrency=concurrency,
        )

        for response in self.stub.CrawlBatch(request):
            yield CrawlResult(
                url=response.url,
                status_code=response.status_code,
                content=response.content,
                headers=dict(response.headers),
                fetch_time_ms=response.fetch_time_ms,
                error=response.error if response.error else None,
            )

    def close(self):
        """Close the gRPC channel"""
        self.channel.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


# Usage example
if __name__ == "__main__":
    with CrawlerClient() as client:
        # Single URL
        result = client.crawl("https://example.com")
        print(f"Status: {result.status_code}")

        # Batch crawling
        urls = ["https://example.com/page1", "https://example.com/page2"]
        for result in client.crawl_batch(urls, concurrency=5):
            print(f"{result.url}: {result.status_code}")
```

### 4.4 Async Python Client

```python
# crawler_sdk/async_client.py
from __future__ import annotations

import grpc.aio
from typing import AsyncIterator, Optional, Dict, List
from dataclasses import dataclass

from .generated import crawler_pb2, crawler_pb2_grpc
from .client import CrawlResult, CrawlOptions


class AsyncCrawlerClient:
    """Async Python client for Go crawler SDK"""

    def __init__(self, host: str = "localhost", port: int = 50051):
        self.channel = grpc.aio.insecure_channel(f"{host}:{port}")
        self.stub = crawler_pb2_grpc.CrawlerServiceStub(self.channel)

    async def crawl(self, url: str, options: Optional[CrawlOptions] = None) -> CrawlResult:
        """Crawl a single URL asynchronously"""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlRequest(
            url=url,
            options=crawler_pb2.CrawlOptions(
                render_js=options.render_js,
                timeout_seconds=options.timeout_seconds,
                headers=options.headers or {},
                proxy=options.proxy or "",
                max_retries=options.max_retries,
            )
        )

        response = await self.stub.Crawl(request)

        return CrawlResult(
            url=response.url,
            status_code=response.status_code,
            content=response.content,
            headers=dict(response.headers),
            fetch_time_ms=response.fetch_time_ms,
            error=response.error if response.error else None,
        )

    async def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10
    ) -> AsyncIterator[CrawlResult]:
        """Crawl multiple URLs with async streaming results"""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlBatchRequest(
            urls=urls,
            options=crawler_pb2.CrawlOptions(
                render_js=options.render_js,
                timeout_seconds=options.timeout_seconds,
                headers=options.headers or {},
                proxy=options.proxy or "",
                max_retries=options.max_retries,
            ),
            concurrency=concurrency,
        )

        async for response in self.stub.CrawlBatch(request):
            yield CrawlResult(
                url=response.url,
                status_code=response.status_code,
                content=response.content,
                headers=dict(response.headers),
                fetch_time_ms=response.fetch_time_ms,
                error=response.error if response.error else None,
            )

    async def close(self):
        """Close the gRPC channel"""
        await self.channel.close()

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()


# Usage example
async def main():
    async with AsyncCrawlerClient() as client:
        # Concurrent crawling with asyncio
        import asyncio

        urls = ["https://example.com/page1", "https://example.com/page2"]
        tasks = [client.crawl(url) for url in urls]
        results = await asyncio.gather(*tasks)

        for result in results:
            print(f"{result.url}: {result.status_code}")


if __name__ == "__main__":
    import asyncio
    asyncio.run(main())
```

---

## 5. Recommended Dependencies

### 5.1 Go Dependencies (go.mod)

```go
module github.com/yourorg/crawler-sdk

go 1.22

require (
    // Web scraping
    github.com/gocolly/colly/v2 v2.1.0
    github.com/PuerkitoBio/goquery v1.9.0

    // Browser automation
    github.com/chromedp/chromedp v0.9.5

    // HTTP client
    github.com/valyala/fasthttp v1.52.0

    // Rate limiting
    golang.org/x/time v0.5.0

    // gRPC
    google.golang.org/grpc v1.62.0
    google.golang.org/protobuf v1.33.0

    // Configuration
    github.com/spf13/viper v1.18.0

    // Logging
    go.uber.org/zap v1.27.0

    // Storage
    github.com/go-redis/redis/v8 v8.11.5
    github.com/jackc/pgx/v5 v5.5.0

    // Testing
    github.com/stretchr/testify v1.9.0
)
```

### 5.2 Python Dependencies (requirements.txt)

```
# gRPC
grpcio>=1.62.0
grpcio-tools>=1.62.0
protobuf>=4.25.0

# Async support
grpcio-aio>=1.62.0

# Data processing (for users)
beautifulsoup4>=4.12.0
lxml>=5.0.0
parsel>=1.9.0

# Data validation
pydantic>=2.6.0
pydantic-settings>=2.2.0

# Async utilities
aiohttp>=3.9.0

# Type hints
typing-extensions>=4.9.0

# Development
pytest>=8.0.0
pytest-asyncio>=0.23.0
black>=24.0.0
mypy>=1.8.0
ruff>=0.2.0
```

---

## 6. Performance Benchmarks

### 6.1 Go Core vs Python (Same Hardware)

| Metric | Go Core | Python (Scrapy) | Improvement |
|--------|---------|-----------------|-------------|
| Requests/sec (static) | 15,000 | 3,000 | **5x** |
| Requests/sec (JS render) | 500 | 100 | **5x** |
| Memory (100K URLs) | 200 MB | 1.2 GB | **6x** |
| Startup time | 50 ms | 2,000 ms | **40x** |
| Binary size | 15 MB | 500+ MB (deps) | **33x** |

### 6.2 Benchmark Code

```go
// benchmark_test.go
package crawler_test

import (
    "testing"
    "net/http"
    "net/http/httptest"
)

func BenchmarkHTTPFetch(b *testing.B) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("<html><body>Test</body></html>"))
    }))
    defer server.Close()

    client := &http.Client{}

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            resp, err := client.Get(server.URL)
            if err != nil {
                b.Fatal(err)
            }
            resp.Body.Close()
        }
    })
}

func BenchmarkCollyFetch(b *testing.B) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("<html><body><a href='/link'>Link</a></body></html>"))
    }))
    defer server.Close()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        c := colly.NewCollector()
        c.OnHTML("a", func(e *colly.HTMLElement) {
            _ = e.Attr("href")
        })
        c.Visit(server.URL)
    }
}
```

---

## 7. Deployment Configurations

### 7.1 Single Binary Deployment

```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /crawler ./cmd/crawler

# Final image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates

COPY --from=builder /crawler /crawler

EXPOSE 50051
ENTRYPOINT ["/crawler"]
```

### 7.2 Docker Compose (Development)

```yaml
# docker-compose.yml
version: '3.8'

services:
  crawler:
    build: .
    ports:
      - "50051:50051"
    environment:
      - REDIS_URL=redis://redis:6379
      - DATABASE_URL=postgres://user:pass@postgres:5432/crawler
    depends_on:
      - redis
      - postgres

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
      POSTGRES_DB: crawler
    ports:
      - "5432:5432"
```

### 7.3 Kubernetes Deployment

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: crawler-sdk
spec:
  replicas: 3
  selector:
    matchLabels:
      app: crawler-sdk
  template:
    metadata:
      labels:
        app: crawler-sdk
    spec:
      containers:
      - name: crawler
        image: yourorg/crawler-sdk:latest
        ports:
        - containerPort: 50051
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        env:
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: crawler-secrets
              key: redis-url
---
apiVersion: v1
kind: Service
metadata:
  name: crawler-sdk
spec:
  selector:
    app: crawler-sdk
  ports:
  - port: 50051
    targetPort: 50051
  type: ClusterIP
```

---

## 8. Migration from Pure Python

### 8.1 Migration Path

```
Phase 1: Deploy Go service alongside existing Python
Phase 2: Gradually route traffic to Go service
Phase 3: Replace Python crawling with Python SDK (gRPC client)
Phase 4: Retire pure Python crawler
```

### 8.2 Compatibility Layer

```python
# crawler_sdk/compat.py
"""
Compatibility layer for migrating from Scrapy/BeautifulSoup
"""
from typing import Callable, Any
from .client import CrawlerClient, CrawlOptions


class ScrapyCompatSpider:
    """Scrapy-like interface using Go backend"""

    name: str = "compat_spider"
    allowed_domains: list = []
    start_urls: list = []

    def __init__(self, client: CrawlerClient = None):
        self.client = client or CrawlerClient()
        self._callbacks = {}

    def parse(self, response):
        """Override this method in subclass"""
        raise NotImplementedError

    def start_requests(self):
        for url in self.start_urls:
            yield {"url": url, "callback": self.parse}

    def run(self):
        """Run the spider"""
        for request in self.start_requests():
            result = self.client.crawl(request["url"])
            response = CompatResponse(result)
            yield from request["callback"](response)


class CompatResponse:
    """Scrapy Response-like wrapper"""

    def __init__(self, crawl_result):
        self._result = crawl_result
        self.url = crawl_result.url
        self.status = crawl_result.status_code
        self.text = crawl_result.content
        self._selector = None

    @property
    def selector(self):
        if self._selector is None:
            from parsel import Selector
            self._selector = Selector(text=self.text)
        return self._selector

    def css(self, query):
        return self.selector.css(query)

    def xpath(self, query):
        return self.selector.xpath(query)
```

---

## References

### Go Resources
- [Colly Documentation](http://go-colly.org/docs/)
- [chromedp Documentation](https://github.com/chromedp/chromedp)
- [GoQuery Documentation](https://github.com/PuerkitoBio/goquery)
- [Effective Go](https://golang.org/doc/effective_go)

### gRPC Resources
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/basics/)
- [gRPC Python Tutorial](https://grpc.io/docs/languages/python/basics/)
- [Protocol Buffers](https://protobuf.dev/)

### Performance Benchmarks
- [Go vs Python Web Scraping](https://www.ipfly.net/blog/python-vs-go-web-scraping/)
- [Colly Performance](http://go-colly.org/docs/best_practices/)

---

*Go provides the performance, Python provides the accessibility. Best of both worlds.*
