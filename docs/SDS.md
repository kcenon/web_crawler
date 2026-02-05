# Software Design Specification (SDS)
# Web Crawler SDK

> **Version**: 1.0.0
> **Created**: 2026-02-05
> **Last Updated**: 2026-02-05
> **Status**: Draft
> **Parent Document**: [SRS.md](./SRS.md)
> **Traceability**: Full bidirectional traceability with SRS

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Architecture](#2-system-architecture)
3. [Module Design](#3-module-design)
4. [Component Design](#4-component-design)
5. [Interface Design](#5-interface-design)
6. [Data Design](#6-data-design)
7. [Algorithm Design](#7-algorithm-design)
8. [Error Handling Design](#8-error-handling-design)
9. [Traceability Matrix](#9-traceability-matrix)
10. [Appendix](#10-appendix)

---

## 1. Introduction

### 1.1 Purpose

본 문서는 Web Crawler SDK의 소프트웨어 설계 명세서(SDS)입니다. SRS에서 정의된 기술 요구사항을 구현 가능한 설계로 변환하며, 개발팀이 시스템을 구현하는 데 필요한 상세한 설계 명세를 제공합니다.

### 1.2 Scope

**포함 범위:**
- 시스템 아키텍처 설계
- 모듈/컴포넌트 상세 설계
- 인터페이스 설계
- 데이터 구조 설계
- 알고리즘 설계
- 에러 처리 전략

**제외 범위:**
- 배포 인프라 상세 구성
- 운영 절차
- 사용자 가이드

### 1.3 Document Conventions

#### 설계 ID 체계

| Prefix | Category | Example |
|--------|----------|---------|
| **SDS-ARCH** | Architecture | SDS-ARCH-001 |
| **SDS-MOD** | Module | SDS-MOD-001 |
| **SDS-COMP** | Component | SDS-COMP-001 |
| **SDS-IF** | Interface | SDS-IF-001 |
| **SDS-DATA** | Data Structure | SDS-DATA-001 |
| **SDS-ALG** | Algorithm | SDS-ALG-001 |
| **SDS-ERR** | Error Handling | SDS-ERR-001 |

### 1.4 References

| Document | Version | Location |
|----------|---------|----------|
| PRD | 1.0.0 | `docs/PRD.md` |
| SRS | 1.0.0 | `docs/SRS.md` |
| SDK Architecture | 1.0.0 | `docs/reference/14-sdk-architecture.md` |

---

## 2. System Architecture

### 2.1 Architecture Overview (SDS-ARCH-001)

**Traces To:** SRS-CORE-001, SRS-CORE-002, SRS-PY-001

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           Web Crawler SDK Architecture                           │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                          Application Layer                               │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                   │    │
│  │  │   Go SDK     │  │ Python SDK   │  │   CLI Tool   │                   │    │
│  │  │   Client     │  │   Client     │  │   (Cobra)    │                   │    │
│  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                   │    │
│  └─────────┼─────────────────┼─────────────────┼───────────────────────────┘    │
│            │                 │                 │                                 │
│            │    ┌────────────┴──────┐          │                                 │
│            │    │   gRPC Protocol   │          │                                 │
│            │    └────────────┬──────┘          │                                 │
│            │                 │                 │                                 │
│  ┌─────────┴─────────────────┴─────────────────┴───────────────────────────┐    │
│  │                           Core Engine Layer                              │    │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │    │
│  │  │                     Crawler Controller                          │    │    │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │    │    │
│  │  │  │ Scheduler  │ │   Engine   │ │  Pipeline  │ │   Stats    │   │    │    │
│  │  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │    │    │
│  │  └─────────────────────────────────────────────────────────────────┘    │    │
│  │                                                                          │    │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │    │
│  │  │                     Processing Components                        │    │    │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐   │    │    │
│  │  │  │HTTP Client │ │  Browser   │ │  Extractor │ │  Frontier  │   │    │    │
│  │  │  │  Module    │ │   Module   │ │   Module   │ │   Module   │   │    │    │
│  │  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘   │    │    │
│  │  └─────────────────────────────────────────────────────────────────┘    │    │
│  │                                                                          │    │
│  │  ┌─────────────────────────────────────────────────────────────────┐    │    │
│  │  │                     Middleware Chain                             │    │    │
│  │  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐        │    │    │
│  │  │  │Retry │→│Rate  │→│Robot │→│Proxy │→│ UA   │→│ Auth │        │    │    │
│  │  │  │      │ │Limit │ │ .txt │ │ Rot. │ │ Rot. │ │      │        │    │    │
│  │  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘        │    │    │
│  │  └─────────────────────────────────────────────────────────────────┘    │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐    │
│  │                          Plugin Layer                                    │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐            │    │
│  │  │  Storage   │ │   Export   │ │   Cache    │ │   Custom   │            │    │
│  │  │  Plugins   │ │  Plugins   │ │  Plugins   │ │  Plugins   │            │    │
│  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘            │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐    │
│  │                        Infrastructure Layer                              │    │
│  │  ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐            │    │
│  │  │   Redis    │ │ PostgreSQL │ │   Kafka    │ │ Prometheus │            │    │
│  │  └────────────┘ └────────────┘ └────────────┘ └────────────┘            │    │
│  └──────────────────────────────────────────────────────────────────────────┘    │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Layer Descriptions

| Layer | Responsibility | Components |
|-------|---------------|------------|
| **Application** | 사용자 인터페이스 제공 | Go SDK, Python SDK, CLI |
| **Core Engine** | 크롤링 핵심 로직 | Controller, Scheduler, Engine |
| **Processing** | 요청/응답 처리 | HTTP Client, Browser, Extractor |
| **Middleware** | 횡단 관심사 처리 | Retry, Rate Limit, Auth |
| **Plugin** | 확장 기능 | Storage, Export, Cache |
| **Infrastructure** | 외부 서비스 연동 | Redis, PostgreSQL, Kafka |

### 2.3 Package Structure (SDS-ARCH-002)

**Traces To:** SRS-API-001

```
github.com/webcrawler/crawler/
├── cmd/
│   └── crawler/             # CLI 엔트리포인트
│       └── main.go
├── pkg/
│   ├── crawler/             # 공개 API
│   │   ├── crawler.go       # Crawler 인터페이스
│   │   ├── config.go        # 설정 구조체
│   │   ├── options.go       # 옵션 패턴
│   │   └── builder.go       # Builder 패턴
│   ├── client/              # HTTP 클라이언트
│   │   ├── client.go
│   │   ├── transport.go
│   │   └── pool.go
│   ├── browser/             # 브라우저 엔진
│   │   ├── pool.go
│   │   ├── page.go
│   │   └── render.go
│   ├── frontier/            # URL 관리
│   │   ├── frontier.go
│   │   ├── queue.go
│   │   └── dedup.go
│   ├── extractor/           # 데이터 추출
│   │   ├── css.go
│   │   ├── xpath.go
│   │   ├── json.go
│   │   └── regex.go
│   ├── middleware/          # 미들웨어
│   │   ├── chain.go
│   │   ├── retry.go
│   │   ├── ratelimit.go
│   │   ├── robots.go
│   │   ├── proxy.go
│   │   ├── useragent.go
│   │   └── auth.go
│   ├── storage/             # 저장소
│   │   ├── plugin.go
│   │   ├── postgres.go
│   │   ├── redis.go
│   │   └── file.go
│   ├── server/              # gRPC 서버
│   │   ├── server.go
│   │   └── handlers.go
│   └── observability/       # 관측성
│       ├── metrics.go
│       └── logging.go
├── api/
│   └── proto/               # Protocol Buffers
│       └── crawler/
│           └── v1/
│               └── crawler.proto
├── internal/                # 비공개 패키지
│   ├── scheduler/
│   ├── pipeline/
│   └── util/
└── python/                  # Python 바인딩
    └── crawler/
        ├── __init__.py
        ├── client.py
        ├── async_client.py
        └── types.py
```

### 2.4 Communication Patterns (SDS-ARCH-003)

**Traces To:** SRS-PY-001, SRS-CLI-001

#### 2.4.1 Go SDK Direct Call

```
┌─────────────┐     Direct Call      ┌─────────────┐
│   Go App    │ ──────────────────► │  Core SDK   │
└─────────────┘                      └─────────────┘
```

#### 2.4.2 Python SDK via gRPC

```
┌─────────────┐       gRPC         ┌─────────────┐     Direct      ┌─────────────┐
│ Python App  │ ─────────────────► │ gRPC Server │ ─────────────► │  Core SDK   │
└─────────────┘                    └─────────────┘                 └─────────────┘
```

#### 2.4.3 CLI via Embedded Engine

```
┌─────────────┐     Embedded        ┌─────────────┐
│     CLI     │ ──────────────────► │  Core SDK   │
└─────────────┘                      └─────────────┘
```

---

## 3. Module Design

### 3.1 Core Engine Module (SDS-MOD-001)

**Traces To:** SRS-CORE-001, SRS-CORE-002, SRS-CORE-003

#### 3.1.1 Module Structure

```go
package crawler

// Crawler is the main interface
type Crawler interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Wait() error

    // URL Management
    AddURL(url string, opts ...RequestOption) error
    AddURLs(urls []string, opts ...RequestOption) error

    // Callbacks
    OnRequest(callback RequestCallback)
    OnResponse(callback ResponseCallback)
    OnError(callback ErrorCallback)
    OnHTML(selector string, callback HTMLCallback)

    // Stats
    Stats() *CrawlStats
}
```

#### 3.1.2 Internal Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        Core Engine Module                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │                    CrawlerImpl                           │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │   │
│  │  │  config    │  │  frontier  │  │  engine    │         │   │
│  │  │  *Config   │  │  Frontier  │  │  *Engine   │         │   │
│  │  └────────────┘  └────────────┘  └────────────┘         │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐         │   │
│  │  │ middleware │  │  plugins   │  │   stats    │         │   │
│  │  │  *Chain    │  │  []Plugin  │  │  *Stats    │         │   │
│  │  └────────────┘  └────────────┘  └────────────┘         │   │
│  │                                                          │   │
│  │  ┌────────────┐  ┌────────────┐                         │   │
│  │  │ callbacks  │  │    ctx     │                         │   │
│  │  │ *Callbacks │  │  context   │                         │   │
│  │  └────────────┘  └────────────┘                         │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

#### 3.1.3 Class Diagram

```
┌──────────────────────────┐
│       <<interface>>      │
│         Crawler          │
├──────────────────────────┤
│ +Start(ctx) error        │
│ +Stop(ctx) error         │
│ +Wait() error            │
│ +AddURL(url) error       │
│ +OnRequest(cb)           │
│ +OnResponse(cb)          │
│ +Stats() *CrawlStats     │
└───────────┬──────────────┘
            │ implements
            ▼
┌──────────────────────────┐
│      CrawlerImpl         │
├──────────────────────────┤
│ -config: *Config         │
│ -frontier: Frontier      │
│ -engine: *Engine         │
│ -middleware: *Chain      │
│ -plugins: []Plugin       │
│ -stats: *Stats           │
│ -callbacks: *Callbacks   │
│ -ctx: context.Context    │
│ -cancel: CancelFunc      │
│ -wg: sync.WaitGroup      │
├──────────────────────────┤
│ +Start(ctx) error        │
│ +Stop(ctx) error         │
│ +Wait() error            │
│ -processURL(entry)       │
│ -handleResponse(resp)    │
│ -runWorkers()            │
└──────────────────────────┘
            │
            │ uses
            ▼
┌──────────────────────────┐     ┌──────────────────────────┐
│        Engine            │     │       Frontier           │
├──────────────────────────┤     ├──────────────────────────┤
│ -client: HTTPClient      │     │ -queue: PriorityQueue    │
│ -browser: BrowserPool    │     │ -dedup: Deduplicator     │
│ -extractor: Extractor    │     │ -filter: URLFilter       │
├──────────────────────────┤     ├──────────────────────────┤
│ +Fetch(req) (*Resp, err) │     │ +Add(entry) error        │
│ +Render(url) (*Page, err)│     │ +Next(ctx) (*Entry, err) │
│ +Extract(resp) (any, err)│     │ +Size() int64            │
└──────────────────────────┘     └──────────────────────────┘
```

### 3.2 HTTP Client Module (SDS-MOD-002)

**Traces To:** SRS-CORE-001, SRS-CORE-002, SRS-CORE-004, SRS-CORE-005

#### 3.2.1 Design

```go
package client

type HTTPClient interface {
    Do(ctx context.Context, req *Request) (*Response, error)
    Close() error
}

type clientImpl struct {
    transport   *http.Transport
    cookieJar   http.CookieJar
    config      *HTTPClientConfig
    concurrency *ConcurrencyManager
    metrics     *ClientMetrics
}
```

#### 3.2.2 Transport Configuration

```go
type HTTPClientConfig struct {
    // Connection Pool
    MaxIdleConns        int           `yaml:"max_idle_conns"`
    MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host"`
    MaxConnsPerHost     int           `yaml:"max_conns_per_host"`
    IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"`

    // Timeouts
    DialTimeout         time.Duration `yaml:"dial_timeout"`
    TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"`
    ResponseHeaderTimeout time.Duration `yaml:"response_header_timeout"`

    // HTTP/2
    HTTP2Enabled        bool          `yaml:"http2_enabled"`

    // TLS
    TLSConfig           *TLSConfig    `yaml:"tls"`

    // Proxy
    Proxy               *ProxyConfig  `yaml:"proxy"`
}
```

#### 3.2.3 Connection Pool Design

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Client Pool                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────────────────────────────────────────────┐     │
│  │              Transport (http.Transport)             │     │
│  │                                                      │     │
│  │  ┌──────────────────────────────────────────────┐  │     │
│  │  │         Connection Pool (per host)            │  │     │
│  │  │                                                │  │     │
│  │  │  example.com  │  api.site.com  │  cdn.net     │  │     │
│  │  │  ┌──┐ ┌──┐    │  ┌──┐ ┌──┐    │  ┌──┐        │  │     │
│  │  │  │C1│ │C2│    │  │C1│ │C2│    │  │C1│        │  │     │
│  │  │  └──┘ └──┘    │  └──┘ └──┘    │  └──┘        │  │     │
│  │  │  MaxPerHost=10│  MaxPerHost=10│  MaxPerHost=10│  │     │
│  │  └──────────────────────────────────────────────┘  │     │
│  │                                                      │     │
│  │  Total: MaxIdleConns = 100                          │     │
│  └────────────────────────────────────────────────────┘     │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 3.3 Browser Module (SDS-MOD-003)

**Traces To:** SRS-JS-001, SRS-JS-002, SRS-JS-003, SRS-JS-004

#### 3.3.1 Browser Pool Design

```go
package browser

type BrowserPool interface {
    Acquire(ctx context.Context) (*Browser, error)
    Release(b *Browser)
    Close() error
}

type poolImpl struct {
    pool      chan *Browser
    config    BrowserPoolConfig
    ctx       context.Context
    cancel    context.CancelFunc
    mu        sync.Mutex
    active    int
    healthCheck *time.Ticker
}
```

#### 3.3.2 Pool State Machine

```
                    ┌─────────────┐
                    │    INIT     │
                    └──────┬──────┘
                           │ Start()
                           ▼
          ┌────────────────────────────────┐
          │                                │
          ▼                                │
    ┌───────────┐   Acquire()        ┌───────────┐
    │   IDLE    │ ────────────────► │   BUSY    │
    │ (in pool) │                    │ (in use)  │
    └───────────┘ ◄──────────────── └───────────┘
          │            Release()           │
          │                                │
          │ Error/Timeout                  │ Crash
          │                                │
          ▼                                ▼
    ┌───────────┐                   ┌───────────┐
    │  RESTART  │ ◄──────────────── │  CRASHED  │
    │           │   Auto-recover    │           │
    └───────────┘                   └───────────┘
          │
          │ Recreate
          ▼
    ┌───────────┐
    │   IDLE    │
    └───────────┘
```

#### 3.3.3 Render Pipeline

```
┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐
│ Acquire │───►│Navigate │───►│  Wait   │───►│ Extract │───►│ Release │
│ Browser │    │  to URL │    │Condition│    │  HTML   │    │ Browser │
└─────────┘    └─────────┘    └─────────┘    └─────────┘    └─────────┘
     │                             │
     │                             ▼
     │                      ┌───────────────────┐
     │                      │   Wait Options    │
     │                      ├───────────────────┤
     │                      │ • Selector        │
     │                      │ • NetworkIdle     │
     │                      │ • Timeout         │
     │                      │ • Custom Function │
     │                      └───────────────────┘
     │
     └─── On Error ───► Recreate Browser ───► Retry
```

### 3.4 Frontier Module (SDS-MOD-004)

**Traces To:** SRS-URL-001, SRS-URL-002, SRS-URL-003, SRS-URL-004

#### 3.4.1 Dual Queue Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          URL Frontier                                │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                      Front Queue (Priority Heap)               │ │
│  │                                                                 │ │
│  │   Priority 10 ─► [URL1, URL2]                                   │ │
│  │   Priority 8  ─► [URL3]                                         │ │
│  │   Priority 5  ─► [URL4, URL5, URL6]                             │ │
│  │   Priority 1  ─► [URL7]                                         │ │
│  │                                                                 │ │
│  │   Next() selects from highest priority with domain rotation    │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                 ▲                                    │
│                                 │ Refill                             │
│                                 │                                    │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                      Back Queues (Per Domain)                  │ │
│  │                                                                 │ │
│  │   example.com    ─► [URL1, URL2, URL3] ──► Rate Limit: 1/s    │ │
│  │   api.site.com   ─► [URL4, URL5]       ──► Rate Limit: 2/s    │ │
│  │   cdn.service.io ─► [URL6]             ──► Rate Limit: 5/s    │ │
│  │                                                                 │ │
│  │   Politeness maintained per domain                             │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                 ▲                                    │
│                                 │ Add (after dedup)                  │
│                                 │                                    │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    Deduplication Layer                         │ │
│  │                                                                 │ │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │ │
│  │   │  In-Memory   │  │    Redis     │  │ Bloom Filter │        │ │
│  │   │   (Set)      │  │   (Set)      │  │ (Approx.)    │        │ │
│  │   └──────────────┘  └──────────────┘  └──────────────┘        │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                 ▲                                    │
│                                 │ Canonicalize                       │
│                                 │                                    │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    URL Canonicalizer                           │ │
│  │                                                                 │ │
│  │   Input:  HTTP://Example.COM:80/path/../page?b=2&a=1#section  │ │
│  │   Output: http://example.com/page?a=1&b=2                      │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

#### 3.4.2 Interface Definition

```go
package frontier

type Frontier interface {
    // URL Management
    Add(ctx context.Context, entry *URLEntry) error
    AddBatch(ctx context.Context, entries []*URLEntry) error
    Next(ctx context.Context) (*URLEntry, error)

    // Statistics
    Size() int64
    PendingByDomain(domain string) int64

    // Lifecycle
    Close() error
}

type URLEntry struct {
    URL          string
    Priority     int           // 1-10, higher = more important
    Depth        int           // crawl depth from seed
    Domain       string        // extracted domain
    DiscoveredAt time.Time
    ScheduledAt  time.Time     // for delayed scheduling
    RetryCount   int
    Metadata     map[string]any
}

type Deduplicator interface {
    IsSeen(url string) (bool, error)
    MarkSeen(url string) error
    Clear() error
    Size() int64
}

type URLFilter interface {
    Allow(entry *URLEntry) bool
}
```

### 3.5 Extractor Module (SDS-MOD-005)

**Traces To:** SRS-EXT-001, SRS-EXT-002, SRS-EXT-003, SRS-EXT-004, SRS-EXT-005

#### 3.5.1 Extractor Interface

```go
package extractor

type Extractor interface {
    CSS(selector string) *Selection
    CSSAll(selector string) []*Selection
    XPath(expr string) *Selection
    XPathAll(expr string) []*Selection
    JSON(path string) gjson.Result
    Regex(pattern string) []string
}

type Selection struct {
    node *html.Node
}

func (s *Selection) Text() string
func (s *Selection) HTML() string
func (s *Selection) Attr(name string) string
func (s *Selection) CSS(selector string) *Selection
func (s *Selection) XPath(expr string) *Selection
```

#### 3.5.2 Extraction Pipeline

```
┌──────────────────────────────────────────────────────────────────────┐
│                        Extraction Pipeline                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐                  │
│  │  Response  │───►│  Encoding  │───►│   Parse    │                  │
│  │   Body     │    │  Detection │    │    DOM     │                  │
│  └────────────┘    └────────────┘    └────────────┘                  │
│                                             │                         │
│              ┌──────────────────────────────┼──────────────────┐     │
│              │                              │                   │     │
│              ▼                              ▼                   ▼     │
│        ┌───────────┐               ┌───────────┐         ┌──────────┐│
│        │    CSS    │               │   XPath   │         │   JSON   ││
│        │ Selector  │               │  Query    │         │   Path   ││
│        └───────────┘               └───────────┘         └──────────┘│
│              │                              │                   │     │
│              └──────────────────────────────┼───────────────────┘     │
│                                             │                         │
│                                             ▼                         │
│                                    ┌────────────────┐                 │
│                                    │  Extracted     │                 │
│                                    │    Data        │                 │
│                                    └────────────────┘                 │
│                                                                       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.6 Middleware Module (SDS-MOD-006)

**Traces To:** SRS-MW-001, SRS-MW-002, SRS-MW-003, SRS-MW-004, SRS-MW-005, SRS-MW-006, SRS-MW-007

#### 3.6.1 Chain Pattern Design

```go
package middleware

type Middleware interface {
    ProcessRequest(ctx context.Context, req *Request) error
    ProcessResponse(ctx context.Context, resp *Response) error
    Name() string
    Priority() int
}

type Chain struct {
    middlewares []Middleware
    sorted      bool
}

func NewChain() *Chain
func (c *Chain) Add(m Middleware) *Chain
func (c *Chain) ProcessRequest(ctx context.Context, req *Request) error
func (c *Chain) ProcessResponse(ctx context.Context, resp *Response) error
```

#### 3.6.2 Middleware Execution Flow

```
                        REQUEST FLOW
                            │
                            ▼
┌──────────────────────────────────────────────────────────────────┐
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│   │  Retry   │─►│   Rate   │─►│  Robots  │─►│  Proxy   │──────►│
│   │ Priority │  │  Limit   │  │   .txt   │  │ Rotation │       │
│   │   =10    │  │   =20    │  │   =30    │  │   =40    │       │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
│                                                                  │
│                    ┌──────────────────────┐                     │
│                    │                      │                     │
│                    │     HTTP Client      │                     │
│                    │                      │                     │
│                    └──────────────────────┘                     │
│                                                                  │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│◄──│  Retry   │◄─│   Rate   │◄─│  Robots  │◄─│  Proxy   │◄──────│
│   │          │  │  Limit   │  │   .txt   │  │ Rotation │       │
│   └──────────┘  └──────────┘  └──────────┘  └──────────┘       │
└──────────────────────────────────────────────────────────────────┘
                            │
                            ▼
                       RESPONSE FLOW
```

#### 3.6.3 Built-in Middlewares

| Middleware | Priority | Description | SRS Trace |
|------------|----------|-------------|-----------|
| RetryMiddleware | 10 | Exponential backoff retry | SRS-MW-002 |
| RateLimitMiddleware | 20 | Token bucket rate limiting | SRS-MW-003 |
| RobotsMiddleware | 30 | robots.txt compliance | SRS-MW-004 |
| ProxyMiddleware | 40 | Proxy rotation | SRS-MW-005 |
| UserAgentMiddleware | 50 | User-Agent rotation | SRS-MW-006 |
| AuthMiddleware | 60 | Authentication handling | SRS-MW-007 |

### 3.7 Storage Module (SDS-MOD-007)

**Traces To:** SRS-STOR-001, SRS-STOR-002, SRS-STOR-003, SRS-STOR-004

#### 3.7.1 Plugin Interface

```go
package storage

type StoragePlugin interface {
    Plugin
    Store(ctx context.Context, data *CrawledData) error
    StoreBatch(ctx context.Context, data []*CrawledData) error
    Query(ctx context.Context, query *StorageQuery) ([]*CrawledData, error)
    Delete(ctx context.Context, query *StorageQuery) (int64, error)
}

type Plugin interface {
    Name() string
    Init(config map[string]any) error
    Close() error
}
```

#### 3.7.2 Storage Implementations

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Storage Plugin System                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                   StoragePlugin Interface                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                               ▲                                      │
│          ┌────────────────────┼────────────────────┐                │
│          │                    │                    │                 │
│  ┌───────┴───────┐   ┌───────┴───────┐   ┌───────┴───────┐         │
│  │  PostgreSQL   │   │     Redis     │   │     File      │         │
│  │    Plugin     │   │    Plugin     │   │    Plugin     │         │
│  ├───────────────┤   ├───────────────┤   ├───────────────┤         │
│  │ • UPSERT      │   │ • URL Queue   │   │ • JSON/JSONL  │         │
│  │ • Batch Insert│   │ • Dedup Set   │   │ • CSV         │         │
│  │ • JSONB Query │   │ • Cache       │   │ • Gzip        │         │
│  │ • Connection  │   │ • Rate Limit  │   │ • Rotation    │         │
│  │   Pool        │   │   Buckets     │   │               │         │
│  └───────────────┘   └───────────────┘   └───────────────┘         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.8 Python Bindings Module (SDS-MOD-008)

**Traces To:** SRS-PY-001, SRS-PY-002, SRS-PY-003, SRS-PY-004

#### 3.8.1 gRPC Service Design

```protobuf
// api/proto/crawler/v1/crawler.proto
syntax = "proto3";
package crawler.v1;

option go_package = "github.com/webcrawler/crawler/api/proto/crawler/v1;crawlerv1";

service CrawlerService {
    // Single URL
    rpc Crawl(CrawlRequest) returns (CrawlResponse);

    // Batch with streaming
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);

    // Job management
    rpc StartJob(StartJobRequest) returns (JobResponse);
    rpc GetJobStatus(JobStatusRequest) returns (JobResponse);
    rpc StopJob(StopJobRequest) returns (JobResponse);

    // Streaming results
    rpc StreamResults(StreamRequest) returns (stream CrawlResponse);

    // Health
    rpc HealthCheck(HealthRequest) returns (HealthResponse);
}
```

#### 3.8.2 Python Client Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Python SDK Architecture                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                       Public API Layer                        │  │
│  │                                                                │  │
│  │   ┌─────────────────┐         ┌─────────────────┐            │  │
│  │   │  CrawlerClient  │         │AsyncCrawlerClient│            │  │
│  │   │   (Sync API)    │         │   (Async API)   │            │  │
│  │   └────────┬────────┘         └────────┬────────┘            │  │
│  └────────────┼──────────────────────────┼──────────────────────┘  │
│               │                          │                          │
│  ┌────────────┼──────────────────────────┼──────────────────────┐  │
│  │            ▼                          ▼                       │  │
│  │   ┌─────────────────────────────────────────────────────┐    │  │
│  │   │              gRPC Client Layer                       │    │  │
│  │   │                                                       │    │  │
│  │   │  ┌─────────────────┐      ┌─────────────────┐       │    │  │
│  │   │  │ grpc.Channel    │      │ grpc.aio.Channel│       │    │  │
│  │   │  └─────────────────┘      └─────────────────┘       │    │  │
│  │   └───────────────────────────────────────────────────────┘    │  │
│  │                                                                │  │
│  │                       Types & Validation                       │  │
│  │   ┌─────────────────────────────────────────────────────┐    │  │
│  │   │  CrawlResult │ CrawlOptions │ JobResponse           │    │  │
│  │   └─────────────────────────────────────────────────────┘    │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.9 CLI Module (SDS-MOD-009)

**Traces To:** SRS-CLI-001, SRS-CLI-002, SRS-CLI-003

#### 3.9.1 Command Structure

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLI Command Tree                             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   crawler (rootCmd)                                                  │
│   │                                                                  │
│   ├── init <name>           # SRS-CLI-002                           │
│   │   ├── --template        # basic, ecommerce, news, api           │
│   │   └── --language        # go, python                            │
│   │                                                                  │
│   ├── crawl <url>           # SRS-CLI-003                           │
│   │   ├── --render-js       # Enable JS rendering                   │
│   │   ├── --timeout         # Request timeout                       │
│   │   ├── --header, -H      # Custom headers (repeatable)           │
│   │   ├── --proxy           # Proxy URL                             │
│   │   ├── --json            # JSON output                           │
│   │   └── --output, -o      # Output file                           │
│   │                                                                  │
│   ├── run [spider]          # Run crawler/spider                    │
│   │   ├── --config, -c      # Config file                           │
│   │   └── --concurrency     # Override concurrency                  │
│   │                                                                  │
│   ├── server                # gRPC server management                │
│   │   ├── start             # Start server                          │
│   │   │   ├── --port        # gRPC port                             │
│   │   │   └── --http-port   # HTTP/metrics port                     │
│   │   └── stop              # Stop server                           │
│   │                                                                  │
│   ├── job                   # Job management                        │
│   │   ├── start             # Start job                             │
│   │   ├── status <id>       # Job status                            │
│   │   ├── stop <id>         # Stop job                              │
│   │   └── list              # List jobs                             │
│   │                                                                  │
│   ├── config                # Configuration                         │
│   │   ├── show              # Show current config                   │
│   │   └── validate          # Validate config file                  │
│   │                                                                  │
│   └── version               # Version info                          │
│                                                                      │
│   Global Flags:                                                      │
│   --config, -c    Config file path                                  │
│   --verbose, -v   Verbose output                                    │
│   --quiet, -q     Minimal output                                    │
│   --debug         Debug mode                                        │
│   --no-color      Disable colors                                    │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Component Design

### 4.1 Retry Component (SDS-COMP-001)

**Traces To:** SRS-MW-002

#### 4.1.1 Exponential Backoff Algorithm

```go
type RetryMiddleware struct {
    config RetryConfig
    rand   *rand.Rand
}

type RetryConfig struct {
    MaxRetries    int           // default: 3
    BaseDelay     time.Duration // default: 1s
    MaxDelay      time.Duration // default: 30s
    Multiplier    float64       // default: 2.0
    Jitter        float64       // default: 0.1
    RetryOnStatus []int         // default: [429, 500, 502, 503, 504]
}

func (m *RetryMiddleware) calculateDelay(attempt int) time.Duration {
    // delay = min(BaseDelay * Multiplier^attempt, MaxDelay)
    delay := float64(m.config.BaseDelay) * math.Pow(m.config.Multiplier, float64(attempt))
    delay = math.Min(delay, float64(m.config.MaxDelay))

    // Apply jitter: delay * (1 ± Jitter)
    jitter := (m.rand.Float64()*2 - 1) * m.config.Jitter
    delay = delay * (1 + jitter)

    return time.Duration(delay)
}
```

#### 4.1.2 Retry State Diagram

```
        ┌─────────────────────────────────────────────────────────┐
        │                                                          │
        ▼                                                          │
   ┌─────────┐         ┌─────────┐         ┌─────────┐           │
   │ Request │────────►│ Execute │────────►│ Success │           │
   │         │         │         │         │         │           │
   └─────────┘         └────┬────┘         └─────────┘           │
                            │                                      │
                            │ Retryable Error                      │
                            │ (429, 5xx, timeout)                  │
                            ▼                                      │
                     ┌────────────┐                                │
                     │  Check     │                                │
                     │  Attempts  │                                │
                     └─────┬──────┘                                │
                           │                                       │
           ┌───────────────┴───────────────┐                      │
           │                               │                       │
           ▼                               ▼                       │
    ┌────────────┐                  ┌────────────┐                │
    │  attempts  │                  │  attempts  │                │
    │  < max     │                  │  >= max    │                │
    └─────┬──────┘                  └─────┬──────┘                │
          │                               │                        │
          ▼                               ▼                        │
    ┌────────────┐                  ┌────────────┐                │
    │   Wait     │                  │   Return   │                │
    │  (backoff) │                  │   Error    │                │
    └─────┬──────┘                  └────────────┘                │
          │                                                        │
          └────────────────────────────────────────────────────────┘
```

### 4.2 Rate Limiter Component (SDS-COMP-002)

**Traces To:** SRS-MW-003

#### 4.2.1 Token Bucket Design

```go
type RateLimiter struct {
    global     *rate.Limiter
    perDomain  map[string]*rate.Limiter
    mu         sync.RWMutex
    config     RateLimitConfig
}

type RateLimitConfig struct {
    RequestsPerSecond float64 // global RPS
    BurstSize         int     // burst capacity
    PerDomain         bool    // enable per-domain limiting
    DomainRPS        float64 // per-domain RPS
}

func (r *RateLimiter) Wait(ctx context.Context, domain string) error {
    // Global rate limit
    if err := r.global.Wait(ctx); err != nil {
        return err
    }

    // Per-domain rate limit
    if r.config.PerDomain {
        limiter := r.getOrCreateDomainLimiter(domain)
        return limiter.Wait(ctx)
    }

    return nil
}
```

#### 4.2.2 Token Bucket Visualization

```
┌─────────────────────────────────────────────────────────────────────┐
│                       Token Bucket Rate Limiter                      │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Global Bucket (10 tokens/sec, burst=20)                           │
│   ┌──────────────────────────────────────────────────────────────┐  │
│   │  [T] [T] [T] [T] [T] [T] [T] [T] [T] [T] [ ] [ ] [ ] [ ] ... │  │
│   │   ▲                                                           │  │
│   │   │ Refill: 10 tokens/second                                  │  │
│   └──────────────────────────────────────────────────────────────┘  │
│                                                                      │
│   Per-Domain Buckets (2 tokens/sec each)                            │
│                                                                      │
│   example.com:    [T] [T] [ ] [ ]  ─ 2/s                            │
│   api.site.com:   [T] [ ] [ ] [ ]  ─ 2/s                            │
│   cdn.net:        [T] [T] [T] [ ]  ─ 2/s (burst=4)                  │
│                                                                      │
│   Request Flow:                                                      │
│   1. Acquire global token                                           │
│   2. Acquire domain token                                           │
│   3. If no token available → wait                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.3 Robots.txt Parser Component (SDS-COMP-003)

**Traces To:** SRS-MW-004

#### 4.3.1 Parser Design

```go
type RobotsChecker struct {
    cache    map[string]*RobotsData
    cacheTTL time.Duration
    mu       sync.RWMutex
    client   HTTPClient
}

type RobotsData struct {
    Rules      map[string][]Rule // user-agent → rules
    CrawlDelay map[string]time.Duration
    Sitemaps   []string
    FetchedAt  time.Time
}

type Rule struct {
    Path  string
    Allow bool
}

func (r *RobotsChecker) IsAllowed(userAgent, url string) (bool, error) {
    domain := extractDomain(url)
    data, err := r.getRobotsData(domain)
    if err != nil {
        return true, err // Allow on error
    }

    path := extractPath(url)
    rules := r.matchingRules(data, userAgent)

    return r.evaluateRules(rules, path), nil
}
```

### 4.4 Concurrency Manager Component (SDS-COMP-004)

**Traces To:** SRS-CORE-003

#### 4.4.1 Semaphore-based Design

```go
type ConcurrencyManager struct {
    globalSem  *semaphore.Weighted
    domainSems map[string]*semaphore.Weighted
    mu         sync.RWMutex
    config     ConcurrencyConfig
}

type ConcurrencyConfig struct {
    MaxConcurrency    int  // global max (default: 100)
    MaxPerDomain      int  // per-domain max (default: 5)
    EnableDomainLimit bool // enable per-domain limiting
}

func (m *ConcurrencyManager) Acquire(ctx context.Context, domain string) error {
    // Acquire global semaphore
    if err := m.globalSem.Acquire(ctx, 1); err != nil {
        return err
    }

    // Acquire domain semaphore
    if m.config.EnableDomainLimit {
        sem := m.getOrCreateDomainSem(domain)
        if err := sem.Acquire(ctx, 1); err != nil {
            m.globalSem.Release(1) // Release global on failure
            return err
        }
    }

    return nil
}

func (m *ConcurrencyManager) Release(domain string) {
    if m.config.EnableDomainLimit {
        if sem, ok := m.domainSems[domain]; ok {
            sem.Release(1)
        }
    }
    m.globalSem.Release(1)
}
```

### 4.5 Encoding Detector Component (SDS-COMP-005)

**Traces To:** SRS-EXT-005

#### 4.5.1 Detection Pipeline

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Encoding Detection Pipeline                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Input: Response Body ([]byte)                                      │
│                                                                      │
│   Step 1: Check Content-Type Header                                  │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ Content-Type: text/html; charset=UTF-8                      │   │
│   │                                  ─────────                   │   │
│   │                                  Found? → Use it            │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                            │ Not Found                               │
│                            ▼                                         │
│   Step 2: Check BOM (Byte Order Mark)                               │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ UTF-8:    EF BB BF                                          │   │
│   │ UTF-16BE: FE FF                                             │   │
│   │ UTF-16LE: FF FE                                             │   │
│   │ Found? → Use it                                             │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                            │ Not Found                               │
│                            ▼                                         │
│   Step 3: Check HTML Meta Tag                                        │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ <meta charset="UTF-8">                                      │   │
│   │ <meta http-equiv="Content-Type" content="...; charset=..."> │   │
│   │ Found? → Use it                                             │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                            │ Not Found                               │
│                            ▼                                         │
│   Step 4: Statistical Detection (chardet)                            │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ Analyze byte patterns → Guess encoding with confidence      │   │
│   │ Confidence > 80%? → Use it                                  │   │
│   │ Otherwise → Default to UTF-8                                │   │
│   └─────────────────────────────────────────────────────────────┘   │
│                                                                      │
│   Output: Decoded string (UTF-8)                                     │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 5. Interface Design

### 5.1 Go SDK Public API (SDS-IF-001)

**Traces To:** SRS-API-001

```go
package crawler

import (
    "context"
    "time"
)

// ==================== Core Interfaces ====================

// Crawler is the main interface for web crawling operations
type Crawler interface {
    // Lifecycle Management
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Wait() error

    // URL Management
    AddURL(url string, opts ...RequestOption) error
    AddURLs(urls []string, opts ...RequestOption) error

    // Event Callbacks
    OnRequest(callback RequestCallback)
    OnResponse(callback ResponseCallback)
    OnError(callback ErrorCallback)
    OnHTML(selector string, callback HTMLCallback)
    OnXML(xpath string, callback XMLCallback)
    OnJSON(path string, callback JSONCallback)

    // Statistics
    Stats() *CrawlStats
}

// ==================== Builder Pattern ====================

// CrawlerBuilder provides fluent API for crawler configuration
type CrawlerBuilder interface {
    WithConfig(cfg *Config) CrawlerBuilder
    WithMiddleware(m Middleware) CrawlerBuilder
    WithPlugin(p Plugin) CrawlerBuilder
    WithStorage(s StoragePlugin) CrawlerBuilder
    WithLogger(l Logger) CrawlerBuilder
    Build() (Crawler, error)
}

// ==================== Factory Functions ====================

// New creates a new Crawler with the given options
func New(opts ...Option) (Crawler, error)

// NewBuilder creates a new CrawlerBuilder
func NewBuilder() CrawlerBuilder

// ==================== Callback Types ====================

type RequestCallback func(req *Request)
type ResponseCallback func(resp *Response)
type ErrorCallback func(req *Request, err error)
type HTMLCallback func(e *HTMLElement)
type XMLCallback func(e *XMLElement)
type JSONCallback func(data gjson.Result)

// ==================== Option Pattern ====================

type Option func(*Config)

func WithConcurrency(n int) Option
func WithTimeout(d time.Duration) Option
func WithUserAgent(ua string) Option
func WithProxy(url string) Option
func WithRateLimit(rps float64) Option
func WithRetry(maxRetries int) Option

// ==================== Request Options ====================

type RequestOption func(*Request)

func WithPriority(p int) RequestOption
func WithHeaders(h map[string]string) RequestOption
func WithRenderJS() RequestOption
func WithMetadata(m map[string]any) RequestOption
```

### 5.2 Python SDK API (SDS-IF-002)

**Traces To:** SRS-API-002, SRS-PY-002, SRS-PY-003, SRS-PY-004

```python
# crawler/client.py
from typing import Optional, Dict, List, Iterator, Any
from dataclasses import dataclass, field

# ==================== Data Classes ====================

@dataclass
class CrawlOptions:
    """Options for crawl requests."""
    render_js: bool = False
    timeout_ms: int = 30000
    headers: Optional[Dict[str, str]] = None
    proxy: Optional[str] = None
    max_retries: int = 3
    priority: int = 5
    metadata: Optional[Dict[str, Any]] = None


@dataclass
class CrawlResult:
    """Result of a crawl operation."""
    url: str
    status_code: int
    content: bytes
    content_type: str
    headers: Dict[str, str]
    fetch_time_ms: int
    final_url: str
    error: Optional[str] = None

    @property
    def success(self) -> bool:
        return self.error is None and 200 <= self.status_code < 400

    @property
    def text(self) -> str:
        return self.content.decode('utf-8', errors='replace')

    def css(self, selector: str) -> 'Selection':
        """Extract using CSS selector."""
        ...

    def xpath(self, expr: str) -> 'Selection':
        """Extract using XPath."""
        ...

    def json(self, path: str = '') -> Any:
        """Extract JSON data."""
        ...


# ==================== Synchronous Client ====================

class CrawlerClient:
    """Synchronous crawler client."""

    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        *,
        secure: bool = False,
        credentials: Optional[Any] = None,
        timeout: float = 30.0,
    ) -> None:
        """Initialize the crawler client."""
        ...

    def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
    ) -> CrawlResult:
        """Crawl a single URL."""
        ...

    def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> Iterator[CrawlResult]:
        """Crawl multiple URLs with streaming results."""
        ...

    def close(self) -> None:
        """Close the client connection."""
        ...

    def __enter__(self) -> "CrawlerClient":
        return self

    def __exit__(self, *args) -> None:
        self.close()


# ==================== Asynchronous Client ====================

class AsyncCrawlerClient:
    """Asynchronous crawler client."""

    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        *,
        secure: bool = False,
    ) -> None:
        ...

    async def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
    ) -> CrawlResult:
        """Crawl a single URL asynchronously."""
        ...

    async def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> AsyncIterator[CrawlResult]:
        """Crawl multiple URLs with async streaming."""
        ...

    async def close(self) -> None:
        """Close the client connection."""
        ...

    async def __aenter__(self) -> "AsyncCrawlerClient":
        return self

    async def __aexit__(self, *args) -> None:
        await self.close()
```

### 5.3 gRPC Interface (SDS-IF-003)

**Traces To:** SRS-API-003, SRS-PY-001

```protobuf
syntax = "proto3";
package crawler.v1;

option go_package = "github.com/webcrawler/crawler/api/proto/crawler/v1;crawlerv1";

// ==================== Main Service ====================

service CrawlerService {
    // Single URL crawling
    rpc Crawl(CrawlRequest) returns (CrawlResponse);

    // Batch crawling with streaming results
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);

    // Job Management
    rpc StartJob(StartJobRequest) returns (JobResponse);
    rpc GetJobStatus(JobStatusRequest) returns (JobResponse);
    rpc StopJob(StopJobRequest) returns (JobResponse);
    rpc ListJobs(ListJobsRequest) returns (ListJobsResponse);

    // Streaming
    rpc StreamResults(StreamRequest) returns (stream CrawlResponse);

    // Statistics
    rpc GetStats(StatsRequest) returns (StatsResponse);

    // Health
    rpc HealthCheck(HealthRequest) returns (HealthResponse);
}

// ==================== Request Messages ====================

message CrawlRequest {
    string url = 1;
    CrawlOptions options = 2;
}

message CrawlBatchRequest {
    repeated string urls = 1;
    CrawlOptions options = 2;
    int32 concurrency = 3;
}

message CrawlOptions {
    bool render_js = 1;
    int32 timeout_ms = 2;
    map<string, string> headers = 3;
    string proxy = 4;
    int32 max_retries = 5;
    int32 priority = 6;
    map<string, string> metadata = 7;
}

message StartJobRequest {
    string name = 1;
    repeated string seed_urls = 2;
    JobConfig config = 3;
}

message JobConfig {
    int32 max_concurrency = 1;
    int32 max_depth = 2;
    float requests_per_second = 3;
    repeated string allowed_domains = 4;
    bool respect_robots_txt = 5;
    CrawlOptions default_options = 6;
}

// ==================== Response Messages ====================

message CrawlResponse {
    string request_id = 1;
    string url = 2;
    int32 status_code = 3;
    bytes content = 4;
    string content_type = 5;
    map<string, string> headers = 6;
    int64 fetch_time_ms = 7;
    string final_url = 8;
    string error = 9;
}

message JobResponse {
    string job_id = 1;
    string name = 2;
    JobStatus status = 3;
    JobProgress progress = 4;
    string error = 5;
    int64 created_at = 6;
    int64 started_at = 7;
    int64 completed_at = 8;
}

enum JobStatus {
    JOB_STATUS_UNSPECIFIED = 0;
    JOB_STATUS_PENDING = 1;
    JOB_STATUS_RUNNING = 2;
    JOB_STATUS_PAUSED = 3;
    JOB_STATUS_COMPLETED = 4;
    JOB_STATUS_FAILED = 5;
    JOB_STATUS_CANCELLED = 6;
}

message JobProgress {
    int64 total_urls = 1;
    int64 crawled_urls = 2;
    int64 success_urls = 3;
    int64 failed_urls = 4;
    int64 pending_urls = 5;
}

message StatsResponse {
    int64 total_requests = 1;
    int64 successful_requests = 2;
    int64 failed_requests = 3;
    int64 bytes_received = 4;
    double average_latency_ms = 5;
    int64 active_workers = 6;
    int64 queue_size = 7;
}
```

### 5.4 Configuration Schema (SDS-IF-004)

**Traces To:** SRS-API-004

```yaml
# crawler.yaml - Complete Configuration Schema

# ==================== Core Settings ====================
crawler:
  name: string                      # Crawler name (required)
  max_concurrency: int              # Max concurrent requests (default: 100)
  max_depth: int                    # Max crawl depth (default: 3)
  requests_per_second: float        # Global RPS limit (default: 10.0)
  allowed_domains: [string]         # Allowed domains filter
  disallowed_domains: [string]      # Disallowed domains filter
  respect_robots_txt: bool          # Respect robots.txt (default: true)
  user_agent: string                # Default User-Agent

# ==================== HTTP Client ====================
http:
  timeout: duration                 # Request timeout (default: 30s)
  max_retries: int                  # Max retry attempts (default: 3)
  follow_redirects: bool            # Follow redirects (default: true)
  max_redirects: int                # Max redirect chain (default: 10)
  max_idle_conns: int               # Max idle connections (default: 100)
  max_idle_conns_per_host: int      # Max idle per host (default: 10)
  idle_conn_timeout: duration       # Idle timeout (default: 90s)
  http2_enabled: bool               # Enable HTTP/2 (default: true)

# ==================== TLS Settings ====================
tls:
  min_version: string               # Min TLS version (default: "1.2")
  insecure_skip_verify: bool        # Skip cert verify (default: false)
  ca_cert_file: string              # CA certificate file
  client_cert_file: string          # Client certificate
  client_key_file: string           # Client key

# ==================== Browser Settings ====================
browser:
  enabled: bool                     # Enable browser (default: false)
  headless: bool                    # Headless mode (default: true)
  pool_size: int                    # Browser pool size (default: 5)
  page_timeout: duration            # Page timeout (default: 30s)
  idle_timeout: duration            # Idle browser timeout (default: 5m)
  executable_path: string           # Chrome path (auto-detect)
  user_data_dir: string             # User data directory

# ==================== Frontier Settings ====================
frontier:
  backend: string                   # memory, redis (default: memory)
  max_size: int                     # Max queue size (default: 100000)
  default_priority: int             # Default priority 1-10 (default: 5)

  dedup:
    backend: string                 # memory, redis, bloom
    bloom_capacity: int             # Bloom filter capacity
    bloom_error_rate: float         # Bloom filter error rate

# ==================== Storage Settings ====================
storage:
  type: string                      # none, postgres, file
  postgres:
    url: string                     # PostgreSQL connection URL
    max_connections: int            # Max pool connections
    table_name: string              # Table name
    batch_size: int                 # Batch insert size
  file:
    format: string                  # json, jsonl, csv
    output_path: string             # Output file path
    max_file_size: string           # Max file size (e.g., "100MB")
    compression: string             # none, gzip

# ==================== Redis Settings ====================
redis:
  url: string                       # Redis connection URL
  password: string                  # Password (or use env var)
  db: int                           # Database number
  pool_size: int                    # Connection pool size
  key_prefix: string                # Key prefix

# ==================== Middleware Settings ====================
middleware:
  retry:
    enabled: bool                   # Enable retry (default: true)
    max_retries: int                # Max retries (default: 3)
    base_delay: duration            # Base delay (default: 1s)
    max_delay: duration             # Max delay (default: 30s)
    multiplier: float               # Backoff multiplier (default: 2.0)
    jitter: float                   # Jitter ratio (default: 0.1)
    retry_on_status: [int]          # Status codes to retry

  rate_limit:
    enabled: bool                   # Enable rate limiting (default: true)
    requests_per_second: float      # RPS limit
    burst_size: int                 # Burst capacity
    per_domain: bool                # Per-domain limiting

  robots:
    enabled: bool                   # Enable robots.txt (default: true)
    cache_ttl: duration             # Cache TTL (default: 24h)

  proxy:
    enabled: bool                   # Enable proxy rotation
    proxies: [string]               # Proxy URLs
    rotation_mode: string           # round-robin, random, health-based

  user_agent:
    enabled: bool                   # Enable UA rotation
    user_agents: [string]           # User-Agent list
    rotation_mode: string           # round-robin, random

  auth:
    type: string                    # none, basic, bearer, oauth2
    basic:
      username: string
      password: string
    bearer:
      token: string
    oauth2:
      client_id: string
      client_secret: string
      token_url: string
      scopes: [string]

# ==================== Logging Settings ====================
logging:
  level: string                     # debug, info, warn, error
  format: string                    # json, text
  output: string                    # stdout, stderr, file path
  file:
    max_size: string                # Max file size
    max_backups: int                # Max backup files
    max_age: int                    # Max age in days
    compress: bool                  # Compress old files

# ==================== Metrics Settings ====================
metrics:
  enabled: bool                     # Enable metrics (default: true)
  prometheus:
    enabled: bool                   # Enable Prometheus (default: true)
    path: string                    # Metrics path (default: /metrics)
    port: int                       # HTTP port (default: 8080)

# ==================== Server Settings ====================
server:
  grpc:
    enabled: bool                   # Enable gRPC server
    port: int                       # gRPC port (default: 50051)
    max_recv_msg_size: string       # Max message size
  http:
    enabled: bool                   # Enable HTTP server
    port: int                       # HTTP port (default: 8080)
```

---

## 6. Data Design

### 6.1 Core Data Structures (SDS-DATA-001)

**Traces To:** SRS-DATA-001, SRS-DATA-002, SRS-DATA-003

```go
// ==================== Request ====================

type Request struct {
    // Identity
    ID        string            `json:"id"`
    URL       string            `json:"url"`
    Method    string            `json:"method"`

    // HTTP Headers
    Headers   map[string]string `json:"headers"`

    // Body (for POST/PUT)
    Body      []byte            `json:"body,omitempty"`

    // Crawl Context
    Depth     int               `json:"depth"`
    Priority  int               `json:"priority"`
    ParentURL string            `json:"parent_url,omitempty"`

    // Options
    RenderJS  bool              `json:"render_js"`
    Timeout   time.Duration     `json:"timeout"`

    // User Metadata
    Metadata  map[string]any    `json:"metadata,omitempty"`

    // Timestamps
    CreatedAt time.Time         `json:"created_at"`
}

// ==================== Response ====================

type Response struct {
    // Reference to Request
    Request   *Request          `json:"request"`

    // HTTP Response
    StatusCode  int             `json:"status_code"`
    Headers     map[string]string `json:"headers"`
    Body        []byte          `json:"body"`
    ContentType string          `json:"content_type"`

    // Performance
    FetchTime   time.Duration   `json:"fetch_time"`

    // Final State (after redirects)
    FinalURL    string          `json:"final_url"`
    Redirects   []string        `json:"redirects,omitempty"`

    // Cache Status
    Cached      bool            `json:"cached"`

    // Timestamps
    ReceivedAt  time.Time       `json:"received_at"`
}

// Convenience methods
func (r *Response) Text() string
func (r *Response) CSS(selector string) *Selection
func (r *Response) XPath(expr string) *Selection
func (r *Response) JSON(path string) gjson.Result
func (r *Response) Links() []string

// ==================== Crawled Data ====================

type CrawledData struct {
    // Identity
    ID          string            `json:"id"`
    URL         string            `json:"url"`
    URLHash     string            `json:"url_hash"`

    // Response Info
    StatusCode  int               `json:"status_code"`
    ContentType string            `json:"content_type"`
    Content     []byte            `json:"content,omitempty"`

    // Extracted Data
    ExtractedData map[string]any  `json:"extracted_data,omitempty"`

    // Metadata
    Domain      string            `json:"domain"`
    Depth       int               `json:"depth"`
    Metadata    map[string]any    `json:"metadata,omitempty"`

    // Timestamps
    CrawledAt   time.Time         `json:"crawled_at"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}
```

### 6.2 Database Schema Design (SDS-DATA-002)

**Traces To:** SRS-STOR-002

```sql
-- ==================== Main Tables ====================

-- Crawled pages storage
CREATE TABLE crawled_pages (
    id              BIGSERIAL PRIMARY KEY,
    url             TEXT UNIQUE NOT NULL,
    url_hash        VARCHAR(64) NOT NULL,
    domain          VARCHAR(255) NOT NULL,
    status_code     INTEGER,
    content_type    VARCHAR(255),
    content         BYTEA,
    extracted       JSONB DEFAULT '{}',
    metadata        JSONB DEFAULT '{}',
    depth           INTEGER DEFAULT 0,
    crawled_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Crawl jobs
CREATE TABLE crawl_jobs (
    id              VARCHAR(36) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    seed_urls       TEXT[] NOT NULL,
    config          JSONB NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress        JSONB DEFAULT '{}',
    error           TEXT,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at      TIMESTAMP WITH TIME ZONE,
    completed_at    TIMESTAMP WITH TIME ZONE
);

-- URL frontier (for persistent queue)
CREATE TABLE url_frontier (
    id              BIGSERIAL PRIMARY KEY,
    job_id          VARCHAR(36) REFERENCES crawl_jobs(id) ON DELETE CASCADE,
    url             TEXT NOT NULL,
    url_hash        VARCHAR(64) NOT NULL,
    priority        INTEGER DEFAULT 5,
    depth           INTEGER DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'pending',
    retry_count     INTEGER DEFAULT 0,
    error           TEXT,
    scheduled_at    TIMESTAMP WITH TIME ZONE,
    crawled_at      TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(job_id, url_hash)
);

-- ==================== Indexes ====================

-- crawled_pages indexes
CREATE INDEX idx_crawled_pages_url_hash ON crawled_pages(url_hash);
CREATE INDEX idx_crawled_pages_domain ON crawled_pages(domain);
CREATE INDEX idx_crawled_pages_crawled_at ON crawled_pages(crawled_at DESC);
CREATE INDEX idx_crawled_pages_status ON crawled_pages(status_code);
CREATE INDEX idx_crawled_pages_extracted ON crawled_pages USING GIN(extracted);

-- crawl_jobs indexes
CREATE INDEX idx_crawl_jobs_status ON crawl_jobs(status);
CREATE INDEX idx_crawl_jobs_created_at ON crawl_jobs(created_at DESC);

-- url_frontier indexes
CREATE INDEX idx_url_frontier_job_status ON url_frontier(job_id, status);
CREATE INDEX idx_url_frontier_priority ON url_frontier(priority DESC) WHERE status = 'pending';
CREATE INDEX idx_url_frontier_scheduled ON url_frontier(scheduled_at) WHERE status = 'pending';

-- ==================== Triggers ====================

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER crawled_pages_updated_at
    BEFORE UPDATE ON crawled_pages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### 6.3 Redis Data Structures (SDS-DATA-003)

**Traces To:** SRS-STOR-003

```
# ==================== Key Patterns ====================

# URL Frontier (Sorted Set - priority as score)
{prefix}:frontier:{job_id}:queue
  - Score: priority * 1000000 + timestamp
  - Member: URL hash

# Seen URLs (Set)
{prefix}:frontier:{job_id}:seen
  - Members: URL hashes

# Domain Rate Limiting (String with TTL)
{prefix}:ratelimit:{domain}
  - Value: token count
  - TTL: 1 second

# Response Cache (Hash)
{prefix}:cache:{url_hash}
  - Fields: status_code, content_type, content, headers, cached_at
  - TTL: configurable

# Job Status (Hash)
{prefix}:job:{job_id}
  - Fields: name, status, total_urls, crawled_urls, etc.

# Active Workers (Set)
{prefix}:workers:active
  - Members: worker IDs

# ==================== Lua Scripts ====================

-- Atomic URL dequeue with domain limiting
KEYS: [queue_key, seen_key, domain_prefix]
ARGV: [max_per_domain]

local url = redis.call('ZPOPMIN', KEYS[1])
if not url then return nil end

local domain = extract_domain(url)
local domain_key = KEYS[3] .. ':' .. domain
local count = redis.call('INCR', domain_key)
redis.call('EXPIRE', domain_key, 1)

if count > tonumber(ARGV[1]) then
    -- Re-queue with delay
    redis.call('ZADD', KEYS[1], 'NX', score + 1000, url)
    return nil
end

return url
```

---

## 7. Algorithm Design

### 7.1 URL Prioritization Algorithm (SDS-ALG-001)

**Traces To:** SRS-URL-001

```go
// Priority calculation algorithm
func calculatePriority(entry *URLEntry, config *PriorityConfig) int {
    basePriority := entry.Priority
    if basePriority == 0 {
        basePriority = config.DefaultPriority
    }

    // Depth penalty: deeper pages get lower priority
    depthPenalty := entry.Depth * config.DepthPenaltyFactor

    // Domain boost: prioritize specific domains
    domainBoost := 0
    if contains(config.HighPriorityDomains, entry.Domain) {
        domainBoost = config.DomainBoostValue
    }

    // Path pattern boost
    pathBoost := 0
    for pattern, boost := range config.PathPatternBoosts {
        if matchesPattern(entry.URL, pattern) {
            pathBoost = max(pathBoost, boost)
        }
    }

    // Calculate final priority (1-10 range)
    priority := basePriority - depthPenalty + domainBoost + pathBoost
    return clamp(priority, 1, 10)
}

// Score for sorted set (higher = more urgent)
func calculateScore(entry *URLEntry) float64 {
    priority := float64(entry.Priority)
    // Use negative timestamp so newer URLs with same priority come later
    timestamp := float64(entry.DiscoveredAt.UnixNano())
    return priority * 1e18 - timestamp
}
```

### 7.2 Exponential Backoff Algorithm (SDS-ALG-002)

**Traces To:** SRS-MW-002

```go
// Exponential backoff with jitter
func exponentialBackoff(attempt int, config *RetryConfig) time.Duration {
    // Base calculation: delay = baseDelay * multiplier^attempt
    delay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(attempt))

    // Cap at maximum delay
    delay = math.Min(delay, float64(config.MaxDelay))

    // Add jitter: ±(jitter * delay)
    jitterRange := config.Jitter * delay
    jitter := (rand.Float64()*2 - 1) * jitterRange
    delay += jitter

    return time.Duration(delay)
}

// Example with default config:
// Attempt 0: 1s * 2^0 = 1s    ±0.1s  → [0.9s, 1.1s]
// Attempt 1: 1s * 2^1 = 2s    ±0.2s  → [1.8s, 2.2s]
// Attempt 2: 1s * 2^2 = 4s    ±0.4s  → [3.6s, 4.4s]
// Attempt 3: 1s * 2^3 = 8s    ±0.8s  → [7.2s, 8.8s]
// ... capped at MaxDelay (30s)
```

### 7.3 URL Canonicalization Algorithm (SDS-ALG-003)

**Traces To:** SRS-URL-002

```go
func canonicalize(rawURL string, opts CanonicalizeOptions) (string, error) {
    // Step 1: Parse URL
    u, err := url.Parse(rawURL)
    if err != nil {
        return "", err
    }

    // Step 2: Normalize scheme (lowercase)
    u.Scheme = strings.ToLower(u.Scheme)

    // Step 3: Normalize host (lowercase, remove www if configured)
    u.Host = strings.ToLower(u.Host)
    if opts.RemoveWWW {
        u.Host = strings.TrimPrefix(u.Host, "www.")
    }

    // Step 4: Remove default port
    if opts.RemoveDefaultPort {
        host, port, _ := net.SplitHostPort(u.Host)
        if (u.Scheme == "http" && port == "80") ||
           (u.Scheme == "https" && port == "443") {
            u.Host = host
        }
    }

    // Step 5: Normalize path
    u.Path = normalizePath(u.Path)
    if opts.RemoveTrailingSlash && u.Path != "/" {
        u.Path = strings.TrimSuffix(u.Path, "/")
    }

    // Step 6: Sort query parameters
    if opts.SortQueryParams {
        u.RawQuery = sortQueryParams(u.Query()).Encode()
    }

    // Step 7: Remove fragment
    if opts.RemoveFragment {
        u.Fragment = ""
    }

    return u.String(), nil
}

func normalizePath(path string) string {
    // Handle empty path
    if path == "" {
        return "/"
    }

    // Resolve . and ..
    segments := strings.Split(path, "/")
    result := make([]string, 0, len(segments))

    for _, seg := range segments {
        switch seg {
        case ".":
            // Skip current directory
        case "..":
            // Go up one level
            if len(result) > 0 {
                result = result[:len(result)-1]
            }
        default:
            result = append(result, seg)
        }
    }

    return "/" + strings.Join(result, "/")
}
```

### 7.4 Token Bucket Rate Limiting (SDS-ALG-004)

**Traces To:** SRS-MW-003

```go
type TokenBucket struct {
    capacity   float64       // Maximum tokens
    tokens     float64       // Current tokens
    rate       float64       // Tokens per second
    lastUpdate time.Time     // Last refill time
    mu         sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    tb.refill()

    if tb.tokens >= 1 {
        tb.tokens--
        return true
    }
    return false
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
    tb.mu.Lock()
    tb.refill()

    if tb.tokens >= 1 {
        tb.tokens--
        tb.mu.Unlock()
        return nil
    }

    // Calculate wait time
    waitTime := time.Duration((1 - tb.tokens) / tb.rate * float64(time.Second))
    tb.mu.Unlock()

    // Wait with context
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(waitTime):
        return tb.Wait(ctx) // Retry after wait
    }
}

func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastUpdate).Seconds()
    tb.tokens = math.Min(tb.capacity, tb.tokens + elapsed * tb.rate)
    tb.lastUpdate = now
}
```

---

## 8. Error Handling Design

### 8.1 Error Types (SDS-ERR-001)

**Traces To:** SRS-REL-001

```go
package errors

// Error codes
const (
    // Network errors (1xxx)
    ErrNetworkTimeout     = 1001
    ErrDNSFailure         = 1002
    ErrConnectionRefused  = 1003
    ErrSSLError           = 1004
    ErrConnectionReset    = 1005

    // Rate limiting errors (2xxx)
    ErrRateLimited        = 2001
    ErrBlocked            = 2002
    ErrRobotsDisallowed   = 2003

    // Parse errors (3xxx)
    ErrParseError         = 3001
    ErrEncodingError      = 3002
    ErrInvalidSelector    = 3003

    // Storage errors (4xxx)
    ErrStorageError       = 4001
    ErrQueueFull          = 4002
    ErrDuplicateURL       = 4003

    // Internal errors (5xxx)
    ErrInternal           = 5001
    ErrConfigInvalid      = 5002
    ErrPluginError        = 5003
)

// CrawlerError is the base error type
type CrawlerError struct {
    Code      int               `json:"code"`
    Message   string            `json:"message"`
    Cause     error             `json:"-"`
    URL       string            `json:"url,omitempty"`
    Retryable bool              `json:"retryable"`
    Context   map[string]any    `json:"context,omitempty"`
}

func (e *CrawlerError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *CrawlerError) Unwrap() error {
    return e.Cause
}

// Error constructors
func NewNetworkError(url string, cause error) *CrawlerError {
    return &CrawlerError{
        Code:      ErrNetworkTimeout,
        Message:   "network request failed",
        Cause:     cause,
        URL:       url,
        Retryable: true,
    }
}

func NewRateLimitError(url string) *CrawlerError {
    return &CrawlerError{
        Code:      ErrRateLimited,
        Message:   "rate limit exceeded",
        URL:       url,
        Retryable: true,
    }
}
```

### 8.2 Error Recovery Strategy (SDS-ERR-002)

**Traces To:** SRS-REL-001

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Error Recovery Strategy Matrix                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  Error Type          │ Recovery Action           │ Max Retries      │
│  ────────────────────┼───────────────────────────┼─────────────────│
│  Network Timeout     │ Retry with backoff        │ 3                │
│  DNS Failure         │ Retry after 5s delay      │ 2                │
│  Connection Refused  │ Retry with backoff        │ 3                │
│  SSL Error           │ Log and skip              │ 0                │
│  429 Rate Limited    │ Respect Retry-After       │ 5                │
│  5xx Server Error    │ Retry with backoff        │ 3                │
│  Robots Disallowed   │ Skip URL                  │ 0                │
│  Parse Error         │ Log and continue          │ 0                │
│  Browser Crash       │ Restart browser, retry    │ 2                │
│  OOM                 │ Reduce concurrency        │ N/A              │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 8.3 Graceful Shutdown Sequence (SDS-ERR-003)

**Traces To:** SRS-REL-002

```
┌─────────────────────────────────────────────────────────────────────┐
│                    Graceful Shutdown Sequence                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   Signal Received (SIGINT/SIGTERM)                                  │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 1. Stop accepting new URLs (disable frontier.Add)           │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 2. Cancel pending requests (context cancellation)            │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 3. Wait for in-flight requests (with timeout: 30s)          │   │
│   │    - Track via sync.WaitGroup                                │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 4. Persist frontier state (if persistent backend)           │   │
│   │    - Save pending URLs to Redis/PostgreSQL                   │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 5. Close browser instances                                   │   │
│   │    - Graceful browser.Close() for each                       │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 6. Flush storage pipelines                                   │   │
│   │    - StoreBatch remaining items                              │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 7. Close database connections                                │   │
│   │    - PostgreSQL pool close                                   │   │
│   │    - Redis client close                                      │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│   ┌─────────────────────────────────────────────────────────────┐   │
│   │ 8. Flush metrics (final push to Prometheus)                  │   │
│   └─────────────────────────────────────────────────────────────┘   │
│            │                                                         │
│            ▼                                                         │
│        Exit(0)                                                       │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 9. Traceability Matrix

### 9.1 SRS → SDS Forward Traceability

| SRS ID | SRS Title | SDS IDs |
|--------|-----------|---------|
| **SRS-CORE-001** | HTTP Client Initialization | SDS-ARCH-001, SDS-MOD-002 |
| **SRS-CORE-002** | Request Execution | SDS-MOD-001, SDS-MOD-002 |
| **SRS-CORE-003** | Concurrent Request Management | SDS-COMP-004, SDS-MOD-001 |
| **SRS-CORE-004** | Cookie Management | SDS-MOD-002 |
| **SRS-CORE-005** | Proxy Support | SDS-MOD-002, SDS-MOD-006 |
| **SRS-JS-001** | Browser Pool Management | SDS-MOD-003 |
| **SRS-JS-002** | Page Rendering | SDS-MOD-003 |
| **SRS-JS-003** | Resource Blocking | SDS-MOD-003 |
| **SRS-JS-004** | Screenshot Capture | SDS-MOD-003 |
| **SRS-URL-001** | URL Frontier | SDS-MOD-004, SDS-ALG-001 |
| **SRS-URL-002** | URL Canonicalization | SDS-MOD-004, SDS-ALG-003 |
| **SRS-URL-003** | URL Deduplication | SDS-MOD-004, SDS-DATA-003 |
| **SRS-URL-004** | URL Filtering | SDS-MOD-004 |
| **SRS-EXT-001** | CSS Selector Extraction | SDS-MOD-005 |
| **SRS-EXT-002** | XPath Extraction | SDS-MOD-005 |
| **SRS-EXT-003** | JSON Extraction | SDS-MOD-005 |
| **SRS-EXT-004** | Regex Extraction | SDS-MOD-005 |
| **SRS-EXT-005** | Encoding Detection | SDS-COMP-005 |
| **SRS-MW-001** | Middleware Chain | SDS-MOD-006 |
| **SRS-MW-002** | Retry Middleware | SDS-COMP-001, SDS-ALG-002 |
| **SRS-MW-003** | Rate Limit Middleware | SDS-COMP-002, SDS-ALG-004 |
| **SRS-MW-004** | Robots.txt Middleware | SDS-COMP-003 |
| **SRS-MW-005** | Proxy Rotation | SDS-MOD-006 |
| **SRS-MW-006** | User-Agent Rotation | SDS-MOD-006 |
| **SRS-MW-007** | Authentication | SDS-MOD-006 |
| **SRS-STOR-001** | Storage Plugin Interface | SDS-MOD-007 |
| **SRS-STOR-002** | PostgreSQL Plugin | SDS-MOD-007, SDS-DATA-002 |
| **SRS-STOR-003** | Redis Integration | SDS-MOD-007, SDS-DATA-003 |
| **SRS-STOR-004** | File Export | SDS-MOD-007 |
| **SRS-PY-001** | gRPC Service | SDS-MOD-008, SDS-IF-003 |
| **SRS-PY-002** | Sync Python Client | SDS-MOD-008, SDS-IF-002 |
| **SRS-PY-003** | Async Python Client | SDS-MOD-008, SDS-IF-002 |
| **SRS-PY-004** | Type Hints | SDS-IF-002 |
| **SRS-CLI-001** | CLI Framework | SDS-MOD-009 |
| **SRS-CLI-002** | Init Command | SDS-MOD-009 |
| **SRS-CLI-003** | Crawl Command | SDS-MOD-009 |
| **SRS-API-001** | Go SDK Interface | SDS-IF-001 |
| **SRS-API-002** | Python SDK Interface | SDS-IF-002 |
| **SRS-API-003** | gRPC Interface | SDS-IF-003 |
| **SRS-API-004** | Configuration Interface | SDS-IF-004 |
| **SRS-DATA-001** | Request Model | SDS-DATA-001 |
| **SRS-DATA-002** | Response Model | SDS-DATA-001 |
| **SRS-DATA-003** | Crawled Data Model | SDS-DATA-001 |
| **SRS-PERF-001** | Throughput | SDS-ARCH-001, SDS-COMP-004 |
| **SRS-PERF-002** | Memory Usage | SDS-MOD-003, SDS-MOD-004 |
| **SRS-PERF-003** | Latency | SDS-MOD-002 |
| **SRS-SEC-001** | TLS Configuration | SDS-MOD-002, SDS-IF-004 |
| **SRS-SEC-002** | Credential Protection | SDS-IF-004 |
| **SRS-SEC-003** | Log Sanitization | SDS-ERR-001 |
| **SRS-REL-001** | Error Recovery | SDS-ERR-001, SDS-ERR-002 |
| **SRS-REL-002** | Graceful Shutdown | SDS-ERR-003 |
| **SRS-OBS-001** | Prometheus Metrics | SDS-ARCH-002 |
| **SRS-OBS-002** | Structured Logging | SDS-ARCH-002 |

### 9.2 SDS → SRS Backward Traceability

| SDS ID | SDS Title | SRS IDs |
|--------|-----------|---------|
| SDS-ARCH-001 | Architecture Overview | SRS-CORE-001, SRS-CORE-002, SRS-PY-001, SRS-PERF-001 |
| SDS-ARCH-002 | Package Structure | SRS-API-001, SRS-OBS-001, SRS-OBS-002 |
| SDS-ARCH-003 | Communication Patterns | SRS-PY-001, SRS-CLI-001 |
| SDS-MOD-001 | Core Engine Module | SRS-CORE-001, SRS-CORE-002, SRS-CORE-003 |
| SDS-MOD-002 | HTTP Client Module | SRS-CORE-001, SRS-CORE-002, SRS-CORE-004, SRS-CORE-005, SRS-SEC-001, SRS-PERF-003 |
| SDS-MOD-003 | Browser Module | SRS-JS-001, SRS-JS-002, SRS-JS-003, SRS-JS-004, SRS-PERF-002 |
| SDS-MOD-004 | Frontier Module | SRS-URL-001, SRS-URL-002, SRS-URL-003, SRS-URL-004, SRS-PERF-002 |
| SDS-MOD-005 | Extractor Module | SRS-EXT-001, SRS-EXT-002, SRS-EXT-003, SRS-EXT-004, SRS-EXT-005 |
| SDS-MOD-006 | Middleware Module | SRS-MW-001 to SRS-MW-007 |
| SDS-MOD-007 | Storage Module | SRS-STOR-001 to SRS-STOR-004 |
| SDS-MOD-008 | Python Bindings Module | SRS-PY-001 to SRS-PY-004 |
| SDS-MOD-009 | CLI Module | SRS-CLI-001 to SRS-CLI-003 |
| SDS-COMP-001 | Retry Component | SRS-MW-002 |
| SDS-COMP-002 | Rate Limiter Component | SRS-MW-003 |
| SDS-COMP-003 | Robots.txt Parser | SRS-MW-004 |
| SDS-COMP-004 | Concurrency Manager | SRS-CORE-003, SRS-PERF-001 |
| SDS-COMP-005 | Encoding Detector | SRS-EXT-005 |
| SDS-IF-001 | Go SDK Public API | SRS-API-001 |
| SDS-IF-002 | Python SDK API | SRS-API-002, SRS-PY-002, SRS-PY-003, SRS-PY-004 |
| SDS-IF-003 | gRPC Interface | SRS-API-003, SRS-PY-001 |
| SDS-IF-004 | Configuration Schema | SRS-API-004, SRS-SEC-001, SRS-SEC-002 |
| SDS-DATA-001 | Core Data Structures | SRS-DATA-001, SRS-DATA-002, SRS-DATA-003 |
| SDS-DATA-002 | Database Schema | SRS-STOR-002 |
| SDS-DATA-003 | Redis Data Structures | SRS-STOR-003, SRS-URL-003 |
| SDS-ALG-001 | URL Prioritization | SRS-URL-001 |
| SDS-ALG-002 | Exponential Backoff | SRS-MW-002 |
| SDS-ALG-003 | URL Canonicalization | SRS-URL-002 |
| SDS-ALG-004 | Token Bucket Rate Limiting | SRS-MW-003 |
| SDS-ERR-001 | Error Types | SRS-REL-001, SRS-SEC-003 |
| SDS-ERR-002 | Error Recovery Strategy | SRS-REL-001 |
| SDS-ERR-003 | Graceful Shutdown | SRS-REL-002 |

### 9.3 Coverage Summary

| SRS Category | Total Requirements | Covered in SDS | Coverage |
|--------------|-------------------|----------------|----------|
| SRS-CORE | 5 | 5 | 100% |
| SRS-JS | 4 | 4 | 100% |
| SRS-URL | 4 | 4 | 100% |
| SRS-EXT | 5 | 5 | 100% |
| SRS-MW | 7 | 7 | 100% |
| SRS-STOR | 4 | 4 | 100% |
| SRS-PY | 4 | 4 | 100% |
| SRS-CLI | 3 | 3 | 100% |
| SRS-API | 4 | 4 | 100% |
| SRS-DATA | 4 | 4 | 100% |
| SRS-PERF | 3 | 3 | 100% |
| SRS-SEC | 3 | 3 | 100% |
| SRS-REL | 2 | 2 | 100% |
| SRS-OBS | 2 | 2 | 100% |
| **Total** | **54** | **54** | **100%** |

---

## 10. Appendix

### 10.1 Design Decisions

| Decision | Options Considered | Selected | Rationale |
|----------|-------------------|----------|-----------|
| Core Language | Go, Rust, C++ | **Go** | 우수한 동시성, 빠른 컴파일, 간결한 문법 |
| Python Binding | cgo, gRPC, REST | **gRPC** | 타입 안전성, 스트리밍 지원, 성능 |
| URL Queue | In-memory, Redis, PostgreSQL | **Pluggable** | 사용 사례별 유연성 |
| Browser Engine | Playwright, Puppeteer, chromedp | **chromedp** | Go 네이티브, 경량, 안정적 |
| Configuration | JSON, YAML, TOML | **YAML** | 가독성, 주석 지원, 널리 사용됨 |

### 10.2 Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-02-05 | Development Team | Initial SDS from SRS |

### 10.3 Document Approval

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Tech Lead | | | |
| Software Architect | | | |
| Senior Developer | | | |

---

*This SDS document maintains full bidirectional traceability with the SRS to ensure all technical requirements are properly addressed in the system design.*
