# SDK Architecture

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Strategy**: Go Core + Python Bindings (Strategy C)
> **Purpose**: Comprehensive architecture design for the crawler SDK

## Overview

This document defines the architecture of the Go-based web crawler SDK, including core interfaces, plugin system, middleware chain, and extension points.

---

## 1. High-Level Architecture

### 1.1 System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client Layer                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │  Python SDK     │  │    Go SDK       │  │   REST API      │              │
│  │  (gRPC Client)  │  │  (Direct API)   │  │   (Optional)    │              │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘              │
│           │                    │                     │                       │
│           └────────────────────┼─────────────────────┘                       │
│                                │                                             │
├────────────────────────────────▼─────────────────────────────────────────────┤
│                            API Gateway Layer                                 │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        gRPC Server                                     │  │
│  │  • Request validation  • Authentication  • Rate limiting               │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────────────────────────────────┤
│                             Core Engine Layer                                │
│                                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │  Scheduler  │  │  Fetcher    │  │   Parser    │  │  Pipeline   │        │
│  │             │◄─┤             │◄─┤             │◄─┤             │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │                │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐        │
│  │URL Frontier │  │ HTTP Client │  │  Extractor  │  │  Storage    │        │
│  │             │  │  + Browser  │  │             │  │  Adapter    │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
│                                                                              │
├──────────────────────────────────────────────────────────────────────────────┤
│                           Middleware Layer                                   │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │ Retry │ RateLimit │ Proxy │ UserAgent │ Dedup │ Robots │ Cookie │ Auth  ││
│  └─────────────────────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────────────────────┤
│                            Plugin Layer                                      │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │ Storage Plugins │ Parser Plugins │ Notifier Plugins │ Custom Plugins    ││
│  └─────────────────────────────────────────────────────────────────────────┘│
├──────────────────────────────────────────────────────────────────────────────┤
│                          Infrastructure Layer                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │    Redis    │  │ PostgreSQL  │  │    Kafka    │  │ Prometheus  │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Overview

| Component | Responsibility | Technology |
|-----------|---------------|------------|
| **Scheduler** | URL scheduling, prioritization | Go, Redis |
| **Fetcher** | HTTP requests, browser automation | net/http, chromedp |
| **Parser** | HTML/JSON extraction | GoQuery, gjson |
| **Pipeline** | Data transformation, storage | Go interfaces |
| **Middleware** | Cross-cutting concerns | Chain of responsibility |
| **Plugin** | Extension points | Go interfaces |

---

## 2. Core Interfaces

### 2.1 Crawler Interface

```go
// pkg/crawler/crawler.go
package crawler

import (
    "context"
)

// Crawler is the main interface for web crawling operations
type Crawler interface {
    // Start begins the crawling process
    Start(ctx context.Context) error

    // Stop gracefully stops the crawler
    Stop(ctx context.Context) error

    // AddURL adds a URL to the crawl queue
    AddURL(url string, opts ...RequestOption) error

    // AddURLs adds multiple URLs to the crawl queue
    AddURLs(urls []string, opts ...RequestOption) error

    // OnResponse registers a callback for successful responses
    OnResponse(callback ResponseCallback)

    // OnError registers a callback for errors
    OnError(callback ErrorCallback)

    // OnHTML registers a callback for HTML elements matching selector
    OnHTML(selector string, callback HTMLCallback)

    // Stats returns current crawling statistics
    Stats() *CrawlStats
}

// ResponseCallback is called when a response is received
type ResponseCallback func(response *Response)

// ErrorCallback is called when an error occurs
type ErrorCallback func(response *Response, err error)

// HTMLCallback is called for each matching HTML element
type HTMLCallback func(element *HTMLElement)

// CrawlStats contains crawling statistics
type CrawlStats struct {
    URLsQueued     int64
    URLsCrawled    int64
    URLsSuccessful int64
    URLsFailed     int64
    BytesReceived  int64
    StartTime      time.Time
    Duration       time.Duration
}
```

### 2.2 Request/Response Types

```go
// pkg/crawler/types.go
package crawler

import (
    "net/http"
    "time"
)

// Request represents a crawl request
type Request struct {
    URL       string
    Method    string
    Headers   http.Header
    Body      []byte
    Depth     int
    Priority  int
    Metadata  map[string]interface{}
    Options   *RequestOptions
}

// RequestOptions configures individual requests
type RequestOptions struct {
    RenderJS       bool
    Timeout        time.Duration
    Proxy          string
    MaxRetries     int
    RetryDelay     time.Duration
    FollowRedirect bool
    MaxRedirects   int
}

// Response represents a crawl response
type Response struct {
    Request     *Request
    StatusCode  int
    Headers     http.Header
    Body        []byte
    ContentType string
    FetchTime   time.Duration
    Cached      bool
}

// HTMLElement represents a parsed HTML element
type HTMLElement struct {
    Tag        string
    Attributes map[string]string
    Text       string
    HTML       string
    Request    *Request
    Response   *Response
}

// Methods on HTMLElement
func (e *HTMLElement) Attr(name string) string {
    return e.Attributes[name]
}

func (e *HTMLElement) AttrOr(name, defaultValue string) string {
    if val, ok := e.Attributes[name]; ok {
        return val
    }
    return defaultValue
}
```

### 2.3 Configuration

```go
// pkg/config/config.go
package config

import (
    "time"
)

// Config holds all crawler configuration
type Config struct {
    // Crawler settings
    Crawler CrawlerConfig `mapstructure:"crawler"`

    // HTTP client settings
    HTTP HTTPConfig `mapstructure:"http"`

    // Browser settings
    Browser BrowserConfig `mapstructure:"browser"`

    // Storage settings
    Storage StorageConfig `mapstructure:"storage"`

    // Middleware settings
    Middleware MiddlewareConfig `mapstructure:"middleware"`

    // Server settings
    Server ServerConfig `mapstructure:"server"`
}

type CrawlerConfig struct {
    MaxDepth           int           `mapstructure:"max_depth"`
    MaxConcurrency     int           `mapstructure:"max_concurrency"`
    RequestsPerSecond  float64       `mapstructure:"requests_per_second"`
    AllowedDomains     []string      `mapstructure:"allowed_domains"`
    DisallowedDomains  []string      `mapstructure:"disallowed_domains"`
    URLFilters         []string      `mapstructure:"url_filters"`
    RespectRobotsTxt   bool          `mapstructure:"respect_robots_txt"`
    UserAgent          string        `mapstructure:"user_agent"`
    DefaultTimeout     time.Duration `mapstructure:"default_timeout"`
}

type HTTPConfig struct {
    MaxIdleConns        int           `mapstructure:"max_idle_conns"`
    MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
    IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
    TLSHandshakeTimeout time.Duration `mapstructure:"tls_handshake_timeout"`
    DisableCompression  bool          `mapstructure:"disable_compression"`
    DisableKeepAlives   bool          `mapstructure:"disable_keepalives"`
}

type BrowserConfig struct {
    Enabled      bool          `mapstructure:"enabled"`
    Headless     bool          `mapstructure:"headless"`
    PoolSize     int           `mapstructure:"pool_size"`
    PageTimeout  time.Duration `mapstructure:"page_timeout"`
    BlockImages  bool          `mapstructure:"block_images"`
    BlockMedia   bool          `mapstructure:"block_media"`
    BlockFonts   bool          `mapstructure:"block_fonts"`
}

type StorageConfig struct {
    Type     string `mapstructure:"type"` // redis, postgres, memory
    RedisURL string `mapstructure:"redis_url"`
    DBUrl    string `mapstructure:"db_url"`
}

type MiddlewareConfig struct {
    Retry     RetryConfig     `mapstructure:"retry"`
    RateLimit RateLimitConfig `mapstructure:"rate_limit"`
    Proxy     ProxyConfig     `mapstructure:"proxy"`
}

type ServerConfig struct {
    GRPCPort int    `mapstructure:"grpc_port"`
    HTTPPort int    `mapstructure:"http_port"`
    Host     string `mapstructure:"host"`
}
```

---

## 3. Middleware System

### 3.1 Middleware Interface

```go
// pkg/middleware/middleware.go
package middleware

import (
    "context"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

// Middleware processes requests and responses
type Middleware interface {
    // ProcessRequest is called before sending the request
    // Return error to stop the request
    ProcessRequest(ctx context.Context, req *crawler.Request) error

    // ProcessResponse is called after receiving the response
    // Return error to mark the request as failed
    ProcessResponse(ctx context.Context, resp *crawler.Response) error

    // Name returns the middleware name for logging
    Name() string

    // Priority determines execution order (lower = earlier)
    Priority() int
}

// Chain holds and executes middleware in order
type Chain struct {
    middlewares []Middleware
}

// NewChain creates a new middleware chain
func NewChain(middlewares ...Middleware) *Chain {
    // Sort by priority
    sort.Slice(middlewares, func(i, j int) bool {
        return middlewares[i].Priority() < middlewares[j].Priority()
    })

    return &Chain{middlewares: middlewares}
}

// ProcessRequest runs all middleware ProcessRequest methods
func (c *Chain) ProcessRequest(ctx context.Context, req *crawler.Request) error {
    for _, m := range c.middlewares {
        if err := m.ProcessRequest(ctx, req); err != nil {
            return fmt.Errorf("middleware %s: %w", m.Name(), err)
        }
    }
    return nil
}

// ProcessResponse runs all middleware ProcessResponse methods (reverse order)
func (c *Chain) ProcessResponse(ctx context.Context, resp *crawler.Response) error {
    for i := len(c.middlewares) - 1; i >= 0; i-- {
        if err := c.middlewares[i].ProcessResponse(ctx, resp); err != nil {
            return fmt.Errorf("middleware %s: %w", c.middlewares[i].Name(), err)
        }
    }
    return nil
}
```

### 3.2 Built-in Middleware

#### Retry Middleware

```go
// pkg/middleware/retry.go
package middleware

import (
    "context"
    "math"
    "time"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type RetryConfig struct {
    MaxRetries     int
    BaseDelay      time.Duration
    MaxDelay       time.Duration
    Multiplier     float64
    RetryOnStatus  []int
}

type RetryMiddleware struct {
    config RetryConfig
}

func NewRetryMiddleware(config RetryConfig) *RetryMiddleware {
    if config.MaxRetries == 0 {
        config.MaxRetries = 3
    }
    if config.BaseDelay == 0 {
        config.BaseDelay = time.Second
    }
    if config.MaxDelay == 0 {
        config.MaxDelay = 30 * time.Second
    }
    if config.Multiplier == 0 {
        config.Multiplier = 2.0
    }
    if len(config.RetryOnStatus) == 0 {
        config.RetryOnStatus = []int{429, 500, 502, 503, 504}
    }

    return &RetryMiddleware{config: config}
}

func (m *RetryMiddleware) Name() string     { return "retry" }
func (m *RetryMiddleware) Priority() int    { return 10 }

func (m *RetryMiddleware) ProcessRequest(ctx context.Context, req *crawler.Request) error {
    // Initialize retry count in metadata
    if req.Metadata == nil {
        req.Metadata = make(map[string]interface{})
    }
    if _, ok := req.Metadata["retry_count"]; !ok {
        req.Metadata["retry_count"] = 0
    }
    return nil
}

func (m *RetryMiddleware) ProcessResponse(ctx context.Context, resp *crawler.Response) error {
    if !m.shouldRetry(resp.StatusCode) {
        return nil
    }

    retryCount := resp.Request.Metadata["retry_count"].(int)
    if retryCount >= m.config.MaxRetries {
        return fmt.Errorf("max retries exceeded")
    }

    // Calculate delay with exponential backoff
    delay := m.calculateDelay(retryCount)
    time.Sleep(delay)

    // Increment retry count and re-queue
    resp.Request.Metadata["retry_count"] = retryCount + 1

    return &RetryError{
        Request: resp.Request,
        Delay:   delay,
    }
}

func (m *RetryMiddleware) shouldRetry(statusCode int) bool {
    for _, code := range m.config.RetryOnStatus {
        if statusCode == code {
            return true
        }
    }
    return false
}

func (m *RetryMiddleware) calculateDelay(retryCount int) time.Duration {
    delay := float64(m.config.BaseDelay) * math.Pow(m.config.Multiplier, float64(retryCount))
    if delay > float64(m.config.MaxDelay) {
        delay = float64(m.config.MaxDelay)
    }
    return time.Duration(delay)
}

type RetryError struct {
    Request *crawler.Request
    Delay   time.Duration
}

func (e *RetryError) Error() string {
    return fmt.Sprintf("retry request after %v", e.Delay)
}
```

#### Rate Limit Middleware

```go
// pkg/middleware/ratelimit.go
package middleware

import (
    "context"
    "sync"

    "golang.org/x/time/rate"
    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type RateLimitConfig struct {
    RequestsPerSecond float64
    BurstSize         int
    PerDomain         bool
}

type RateLimitMiddleware struct {
    config   RateLimitConfig
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
    return &RateLimitMiddleware{
        config:   config,
        limiters: make(map[string]*rate.Limiter),
    }
}

func (m *RateLimitMiddleware) Name() string     { return "ratelimit" }
func (m *RateLimitMiddleware) Priority() int    { return 20 }

func (m *RateLimitMiddleware) ProcessRequest(ctx context.Context, req *crawler.Request) error {
    limiter := m.getLimiter(req.URL)
    return limiter.Wait(ctx)
}

func (m *RateLimitMiddleware) ProcessResponse(ctx context.Context, resp *crawler.Response) error {
    return nil
}

func (m *RateLimitMiddleware) getLimiter(url string) *rate.Limiter {
    key := "global"
    if m.config.PerDomain {
        u, _ := urlpkg.Parse(url)
        if u != nil {
            key = u.Host
        }
    }

    m.mu.RLock()
    limiter, exists := m.limiters[key]
    m.mu.RUnlock()

    if exists {
        return limiter
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    // Double-check
    if limiter, exists = m.limiters[key]; exists {
        return limiter
    }

    limiter = rate.NewLimiter(rate.Limit(m.config.RequestsPerSecond), m.config.BurstSize)
    m.limiters[key] = limiter
    return limiter
}
```

#### Proxy Rotation Middleware

```go
// pkg/middleware/proxy.go
package middleware

import (
    "context"
    "sync"
    "sync/atomic"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type ProxyConfig struct {
    Proxies      []string
    RotationMode string // round-robin, random, health-based
}

type ProxyMiddleware struct {
    config  ProxyConfig
    index   uint64
    health  map[string]*proxyHealth
    mu      sync.RWMutex
}

type proxyHealth struct {
    success int64
    failure int64
    blocked bool
}

func NewProxyMiddleware(config ProxyConfig) *ProxyMiddleware {
    m := &ProxyMiddleware{
        config: config,
        health: make(map[string]*proxyHealth),
    }

    for _, proxy := range config.Proxies {
        m.health[proxy] = &proxyHealth{}
    }

    return m
}

func (m *ProxyMiddleware) Name() string     { return "proxy" }
func (m *ProxyMiddleware) Priority() int    { return 30 }

func (m *ProxyMiddleware) ProcessRequest(ctx context.Context, req *crawler.Request) error {
    if len(m.config.Proxies) == 0 {
        return nil
    }

    proxy := m.selectProxy()
    if req.Options == nil {
        req.Options = &crawler.RequestOptions{}
    }
    req.Options.Proxy = proxy
    req.Metadata["proxy"] = proxy

    return nil
}

func (m *ProxyMiddleware) ProcessResponse(ctx context.Context, resp *crawler.Response) error {
    proxy, ok := resp.Request.Metadata["proxy"].(string)
    if !ok {
        return nil
    }

    m.mu.Lock()
    defer m.mu.Unlock()

    health := m.health[proxy]
    if resp.StatusCode >= 200 && resp.StatusCode < 400 {
        atomic.AddInt64(&health.success, 1)
    } else {
        atomic.AddInt64(&health.failure, 1)
        if resp.StatusCode == 403 || resp.StatusCode == 407 {
            health.blocked = true
        }
    }

    return nil
}

func (m *ProxyMiddleware) selectProxy() string {
    switch m.config.RotationMode {
    case "random":
        return m.config.Proxies[rand.Intn(len(m.config.Proxies))]
    case "health-based":
        return m.selectHealthyProxy()
    default: // round-robin
        idx := atomic.AddUint64(&m.index, 1)
        return m.config.Proxies[idx%uint64(len(m.config.Proxies))]
    }
}

func (m *ProxyMiddleware) selectHealthyProxy() string {
    m.mu.RLock()
    defer m.mu.RUnlock()

    var best string
    var bestScore float64 = -1

    for proxy, health := range m.health {
        if health.blocked {
            continue
        }

        total := float64(health.success + health.failure)
        if total == 0 {
            return proxy // Try unused proxy first
        }

        score := float64(health.success) / total
        if score > bestScore {
            bestScore = score
            best = proxy
        }
    }

    if best == "" {
        // All blocked, reset and try again
        for _, health := range m.health {
            health.blocked = false
        }
        return m.config.Proxies[0]
    }

    return best
}
```

---

## 4. Plugin System

### 4.1 Plugin Interface

```go
// pkg/plugin/plugin.go
package plugin

import (
    "context"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

// Plugin is the base interface for all plugins
type Plugin interface {
    // Name returns the plugin name
    Name() string

    // Init initializes the plugin with configuration
    Init(config map[string]interface{}) error

    // Close cleans up plugin resources
    Close() error
}

// StoragePlugin stores crawled data
type StoragePlugin interface {
    Plugin

    // Store saves the crawled data
    Store(ctx context.Context, data *crawler.CrawledData) error

    // Query retrieves stored data
    Query(ctx context.Context, query *StorageQuery) ([]*crawler.CrawledData, error)

    // Delete removes stored data
    Delete(ctx context.Context, query *StorageQuery) error
}

// ParserPlugin extracts data from responses
type ParserPlugin interface {
    Plugin

    // CanParse returns true if this parser can handle the content type
    CanParse(contentType string) bool

    // Parse extracts data from the response
    Parse(ctx context.Context, resp *crawler.Response) (*ParseResult, error)
}

// NotifierPlugin sends notifications about crawl events
type NotifierPlugin interface {
    Plugin

    // Notify sends a notification
    Notify(ctx context.Context, event *CrawlEvent) error
}

// ExporterPlugin exports crawled data to external systems
type ExporterPlugin interface {
    Plugin

    // Export sends data to external system
    Export(ctx context.Context, data []*crawler.CrawledData, format string) error
}
```

### 4.2 Plugin Registry

```go
// pkg/plugin/registry.go
package plugin

import (
    "fmt"
    "sync"
)

// Registry manages plugin registration and retrieval
type Registry struct {
    storage   map[string]StoragePlugin
    parsers   map[string]ParserPlugin
    notifiers map[string]NotifierPlugin
    exporters map[string]ExporterPlugin
    mu        sync.RWMutex
}

// NewRegistry creates a new plugin registry
func NewRegistry() *Registry {
    return &Registry{
        storage:   make(map[string]StoragePlugin),
        parsers:   make(map[string]ParserPlugin),
        notifiers: make(map[string]NotifierPlugin),
        exporters: make(map[string]ExporterPlugin),
    }
}

// RegisterStorage registers a storage plugin
func (r *Registry) RegisterStorage(name string, plugin StoragePlugin) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.storage[name]; exists {
        return fmt.Errorf("storage plugin %s already registered", name)
    }

    r.storage[name] = plugin
    return nil
}

// GetStorage retrieves a storage plugin
func (r *Registry) GetStorage(name string) (StoragePlugin, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    plugin, exists := r.storage[name]
    if !exists {
        return nil, fmt.Errorf("storage plugin %s not found", name)
    }

    return plugin, nil
}

// Similar methods for other plugin types...
```

### 4.3 Built-in Plugins

#### PostgreSQL Storage Plugin

```go
// pkg/plugin/storage/postgres.go
package storage

import (
    "context"
    "database/sql"
    "encoding/json"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/yourorg/crawler-sdk/pkg/crawler"
    "github.com/yourorg/crawler-sdk/pkg/plugin"
)

type PostgresStorage struct {
    db *sql.DB
}

func NewPostgresStorage() *PostgresStorage {
    return &PostgresStorage{}
}

func (p *PostgresStorage) Name() string { return "postgres" }

func (p *PostgresStorage) Init(config map[string]interface{}) error {
    url, ok := config["url"].(string)
    if !ok {
        return fmt.Errorf("postgres url required")
    }

    db, err := sql.Open("pgx", url)
    if err != nil {
        return err
    }

    p.db = db

    // Create tables if not exists
    return p.createTables()
}

func (p *PostgresStorage) createTables() error {
    _, err := p.db.Exec(`
        CREATE TABLE IF NOT EXISTS crawled_pages (
            id SERIAL PRIMARY KEY,
            url TEXT UNIQUE NOT NULL,
            status_code INTEGER,
            content_type TEXT,
            content BYTEA,
            metadata JSONB,
            crawled_at TIMESTAMP DEFAULT NOW(),
            created_at TIMESTAMP DEFAULT NOW()
        );

        CREATE INDEX IF NOT EXISTS idx_crawled_pages_url ON crawled_pages(url);
        CREATE INDEX IF NOT EXISTS idx_crawled_pages_crawled_at ON crawled_pages(crawled_at);
    `)
    return err
}

func (p *PostgresStorage) Store(ctx context.Context, data *crawler.CrawledData) error {
    metadata, _ := json.Marshal(data.Metadata)

    _, err := p.db.ExecContext(ctx, `
        INSERT INTO crawled_pages (url, status_code, content_type, content, metadata)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (url) DO UPDATE SET
            status_code = EXCLUDED.status_code,
            content_type = EXCLUDED.content_type,
            content = EXCLUDED.content,
            metadata = EXCLUDED.metadata,
            crawled_at = NOW()
    `, data.URL, data.StatusCode, data.ContentType, data.Content, metadata)

    return err
}

func (p *PostgresStorage) Query(ctx context.Context, query *plugin.StorageQuery) ([]*crawler.CrawledData, error) {
    // Implementation
    return nil, nil
}

func (p *PostgresStorage) Delete(ctx context.Context, query *plugin.StorageQuery) error {
    // Implementation
    return nil
}

func (p *PostgresStorage) Close() error {
    if p.db != nil {
        return p.db.Close()
    }
    return nil
}
```

---

## 5. URL Frontier Architecture

### 5.1 Frontier Interface

```go
// pkg/frontier/frontier.go
package frontier

import (
    "context"

    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

// Frontier manages the URL queue
type Frontier interface {
    // Add adds a URL to the frontier
    Add(ctx context.Context, req *crawler.Request) error

    // AddBatch adds multiple URLs
    AddBatch(ctx context.Context, reqs []*crawler.Request) error

    // Next returns the next URL to crawl
    Next(ctx context.Context) (*crawler.Request, error)

    // Done marks a URL as completed
    Done(ctx context.Context, url string) error

    // Failed marks a URL as failed
    Failed(ctx context.Context, url string, err error) error

    // Size returns the number of URLs in the frontier
    Size(ctx context.Context) (int64, error)

    // HasSeen checks if URL has been seen
    HasSeen(ctx context.Context, url string) (bool, error)

    // Close closes the frontier
    Close() error
}
```

### 5.2 Redis-backed Frontier

```go
// pkg/frontier/redis.go
package frontier

import (
    "context"
    "encoding/json"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type RedisFrontier struct {
    client *redis.Client
    prefix string
}

func NewRedisFrontier(redisURL string, prefix string) (*RedisFrontier, error) {
    opt, err := redis.ParseURL(redisURL)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }

    return &RedisFrontier{
        client: client,
        prefix: prefix,
    }, nil
}

func (f *RedisFrontier) Add(ctx context.Context, req *crawler.Request) error {
    // Check if already seen
    seen, err := f.HasSeen(ctx, req.URL)
    if err != nil {
        return err
    }
    if seen {
        return nil
    }

    // Add to seen set
    if err := f.client.SAdd(ctx, f.key("seen"), f.urlHash(req.URL)).Err(); err != nil {
        return err
    }

    // Serialize request
    data, err := json.Marshal(req)
    if err != nil {
        return err
    }

    // Add to priority queue (sorted set, score = priority)
    return f.client.ZAdd(ctx, f.key("queue"), &redis.Z{
        Score:  float64(req.Priority),
        Member: string(data),
    }).Err()
}

func (f *RedisFrontier) Next(ctx context.Context) (*crawler.Request, error) {
    // Pop highest priority item
    result, err := f.client.ZPopMax(ctx, f.key("queue"), 1).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, nil // Empty queue
        }
        return nil, err
    }

    if len(result) == 0 {
        return nil, nil
    }

    var req crawler.Request
    if err := json.Unmarshal([]byte(result[0].Member.(string)), &req); err != nil {
        return nil, err
    }

    return &req, nil
}

func (f *RedisFrontier) HasSeen(ctx context.Context, url string) (bool, error) {
    return f.client.SIsMember(ctx, f.key("seen"), f.urlHash(url)).Result()
}

func (f *RedisFrontier) Size(ctx context.Context) (int64, error) {
    return f.client.ZCard(ctx, f.key("queue")).Result()
}

func (f *RedisFrontier) key(name string) string {
    return f.prefix + ":" + name
}

func (f *RedisFrontier) urlHash(url string) string {
    // Normalize and hash URL
    h := sha256.Sum256([]byte(strings.ToLower(url)))
    return hex.EncodeToString(h[:16])
}

func (f *RedisFrontier) Close() error {
    return f.client.Close()
}
```

---

## 6. Error Handling

### 6.1 Error Types

```go
// pkg/errors/errors.go
package errors

import (
    "fmt"
)

// CrawlerError is the base error type
type CrawlerError struct {
    Code    string
    Message string
    URL     string
    Cause   error
}

func (e *CrawlerError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v (url: %s)", e.Code, e.Message, e.Cause, e.URL)
    }
    return fmt.Sprintf("[%s] %s (url: %s)", e.Code, e.Message, e.URL)
}

func (e *CrawlerError) Unwrap() error {
    return e.Cause
}

// Specific error types
var (
    ErrNetworkTimeout = &CrawlerError{Code: "NETWORK_TIMEOUT", Message: "request timed out"}
    ErrRateLimited    = &CrawlerError{Code: "RATE_LIMITED", Message: "rate limit exceeded"}
    ErrBlocked        = &CrawlerError{Code: "BLOCKED", Message: "request blocked"}
    ErrRobotsTxt      = &CrawlerError{Code: "ROBOTS_TXT", Message: "disallowed by robots.txt"}
    ErrMaxRetries     = &CrawlerError{Code: "MAX_RETRIES", Message: "max retries exceeded"}
    ErrInvalidURL     = &CrawlerError{Code: "INVALID_URL", Message: "invalid URL"}
    ErrParseFailed    = &CrawlerError{Code: "PARSE_FAILED", Message: "failed to parse content"}
)

// NewError creates a new CrawlerError
func NewError(code, message, url string, cause error) *CrawlerError {
    return &CrawlerError{
        Code:    code,
        Message: message,
        URL:     url,
        Cause:   cause,
    }
}

// IsRetryable returns true if the error is retryable
func IsRetryable(err error) bool {
    var crawlerErr *CrawlerError
    if errors.As(err, &crawlerErr) {
        switch crawlerErr.Code {
        case "NETWORK_TIMEOUT", "RATE_LIMITED":
            return true
        }
    }
    return false
}
```

---

## 7. Observability

### 7.1 Metrics

```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    RequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "crawler_requests_total",
            Help: "Total number of requests made",
        },
        []string{"status", "domain"},
    )

    RequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "crawler_request_duration_seconds",
            Help:    "Request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"domain"},
    )

    QueueSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "crawler_queue_size",
            Help: "Current size of the URL queue",
        },
    )

    ActiveWorkers = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "crawler_active_workers",
            Help: "Number of active worker goroutines",
        },
    )

    BytesReceived = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "crawler_bytes_received_total",
            Help: "Total bytes received",
        },
    )
)
```

### 7.2 Structured Logging

```go
// pkg/logging/logging.go
package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

var logger *zap.Logger

func Init(level string, production bool) error {
    var config zap.Config

    if production {
        config = zap.NewProductionConfig()
    } else {
        config = zap.NewDevelopmentConfig()
    }

    // Parse level
    var zapLevel zapcore.Level
    if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
        zapLevel = zapcore.InfoLevel
    }
    config.Level.SetLevel(zapLevel)

    var err error
    logger, err = config.Build()
    return err
}

func Info(msg string, fields ...zap.Field) {
    logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    logger.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
    logger.Debug(msg, fields...)
}

func WithFields(fields ...zap.Field) *zap.Logger {
    return logger.With(fields...)
}
```

---

## 8. Directory Structure

```
crawler-sdk/
├── cmd/
│   ├── crawler/           # Main CLI application
│   │   └── main.go
│   └── server/            # gRPC server
│       └── main.go
├── internal/
│   ├── engine/            # Core crawling engine
│   │   ├── engine.go
│   │   ├── fetcher.go
│   │   └── worker.go
│   ├── browser/           # Browser automation
│   │   ├── pool.go
│   │   └── chromedp.go
│   └── server/            # gRPC server implementation
│       └── grpc.go
├── pkg/
│   ├── crawler/           # Public crawler interfaces
│   │   ├── crawler.go
│   │   └── types.go
│   ├── config/            # Configuration
│   │   └── config.go
│   ├── middleware/        # Middleware system
│   │   ├── middleware.go
│   │   ├── retry.go
│   │   ├── ratelimit.go
│   │   └── proxy.go
│   ├── plugin/            # Plugin system
│   │   ├── plugin.go
│   │   ├── registry.go
│   │   └── storage/
│   │       ├── postgres.go
│   │       └── redis.go
│   ├── frontier/          # URL frontier
│   │   ├── frontier.go
│   │   └── redis.go
│   ├── errors/            # Error types
│   │   └── errors.go
│   ├── metrics/           # Observability
│   │   └── metrics.go
│   └── logging/           # Logging
│       └── logging.go
├── api/
│   └── proto/             # Protocol buffer definitions
│       └── crawler.proto
├── bindings/
│   └── python/            # Python gRPC client
│       ├── crawler_sdk/
│       │   ├── __init__.py
│       │   ├── client.py
│       │   └── async_client.py
│       ├── setup.py
│       └── requirements.txt
├── configs/
│   ├── default.yaml       # Default configuration
│   └── production.yaml    # Production configuration
├── docs/
│   └── reference/         # This documentation
├── examples/
│   ├── go/                # Go usage examples
│   └── python/            # Python usage examples
├── scripts/
│   ├── build.sh           # Build script
│   ├── proto.sh           # Proto generation
│   └── release.sh         # Release script
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

---

## References

- [Go Design Patterns](https://golang.org/doc/effective_go)
- [Colly Architecture](http://go-colly.org/docs/introduction/start/)
- [gRPC Best Practices](https://grpc.io/docs/guides/performance/)
- [Clean Architecture in Go](https://github.com/bxcodec/go-clean-arch)

---

*A well-designed architecture is the foundation of a maintainable and scalable SDK.*
