# Software Requirements Specification (SRS)
# Web Crawler SDK

> **Version**: 1.0.0
> **Created**: 2026-02-05
> **Last Updated**: 2026-02-05
> **Status**: Draft
> **Parent Document**: [PRD.md](./PRD.md)
> **Traceability**: Full bidirectional traceability with PRD

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [System Overview](#2-system-overview)
3. [Functional Requirements Specification](#3-functional-requirements-specification)
4. [Non-Functional Requirements Specification](#4-non-functional-requirements-specification)
5. [Interface Requirements](#5-interface-requirements)
6. [Data Requirements](#6-data-requirements)
7. [System Constraints](#7-system-constraints)
8. [Traceability Matrix](#8-traceability-matrix)
9. [Appendix](#9-appendix)

---

## 1. Introduction

### 1.1 Purpose

본 문서는 Web Crawler SDK의 소프트웨어 요구사항 명세서(SRS)입니다. PRD에서 정의된 비즈니스 요구사항을 구현 가능한 기술 명세로 변환하며, 개발팀이 시스템을 구현하는 데 필요한 상세한 기술 요구사항을 제공합니다.

### 1.2 Scope

**포함 범위:**
- Go 코어 엔진의 상세 기술 명세
- Python 바인딩 인터페이스 명세
- 미들웨어 시스템 명세
- 플러그인 아키텍처 명세
- API 및 CLI 인터페이스 명세
- 데이터 모델 및 저장소 명세

**제외 범위:**
- 배포 인프라 상세 구성
- 운영 매뉴얼
- 사용자 튜토리얼

### 1.3 Document Conventions

#### 요구사항 ID 체계

| Prefix | Category | Example |
|--------|----------|---------|
| **SRS-CORE** | Core Engine | SRS-CORE-001 |
| **SRS-JS** | JavaScript Rendering | SRS-JS-001 |
| **SRS-URL** | URL Management | SRS-URL-001 |
| **SRS-EXT** | Data Extraction | SRS-EXT-001 |
| **SRS-MW** | Middleware | SRS-MW-001 |
| **SRS-STOR** | Storage | SRS-STOR-001 |
| **SRS-PY** | Python Bindings | SRS-PY-001 |
| **SRS-CLI** | CLI Tools | SRS-CLI-001 |
| **SRS-API** | API/Interface | SRS-API-001 |
| **SRS-DATA** | Data Model | SRS-DATA-001 |
| **SRS-PERF** | Performance | SRS-PERF-001 |
| **SRS-SEC** | Security | SRS-SEC-001 |
| **SRS-REL** | Reliability | SRS-REL-001 |

#### 우선순위 정의

| Priority | Definition | SLA |
|----------|------------|-----|
| **P0** | Must Have | MVP에 필수 포함 |
| **P1** | Should Have | v1.0 릴리스 전 구현 |
| **P2** | Nice to Have | 향후 버전에서 구현 |

#### 상태 정의

| Status | Definition |
|--------|------------|
| **Draft** | 초안 작성 중 |
| **Review** | 검토 중 |
| **Approved** | 승인됨 |
| **Implemented** | 구현 완료 |
| **Verified** | 검증 완료 |

### 1.4 References

| Document | Version | Location |
|----------|---------|----------|
| PRD | 1.0.0 | `docs/PRD.md` |
| SDK Architecture | 1.0.0 | `docs/reference/14-sdk-architecture.md` |
| Go-Python Binding | 1.0.0 | `docs/reference/15-go-python-binding.md` |
| Developer Experience | 1.0.0 | `docs/reference/16-developer-experience.md` |

### 1.5 Glossary

| Term | Definition |
|------|------------|
| **Crawler** | 웹 페이지를 자동으로 탐색하고 데이터를 수집하는 소프트웨어 |
| **Frontier** | 크롤링 대기 중인 URL을 관리하는 우선순위 큐 |
| **Middleware** | 요청/응답 처리 파이프라인에 삽입되는 처리 컴포넌트 |
| **Spider** | 특정 웹사이트나 도메인을 크롤링하는 로직 단위 |
| **Pipeline** | 추출된 데이터를 처리하고 저장하는 일련의 프로세스 |
| **gRPC** | Google이 개발한 고성능 원격 프로시저 호출 프레임워크 |
| **Politeness** | 대상 서버에 과부하를 주지 않도록 요청을 조절하는 정책 |

---

## 2. System Overview

### 2.1 System Context

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              External Systems                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │ Target Web  │  │   Proxy     │  │   robots    │  │ External    │        │
│  │   Sites     │  │  Servers    │  │    .txt     │  │   APIs      │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │                │
└─────────┼────────────────┼────────────────┼────────────────┼────────────────┘
          │                │                │                │
          ▼                ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Web Crawler SDK                                    │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                         Core Engine (Go)                               │  │
│  │  HTTP Client │ Browser │ Scheduler │ Parser │ Middleware │ Plugins    │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                    │                                         │
│  ┌──────────────────┐  ┌──────────▼──────────┐  ┌──────────────────┐       │
│  │   gRPC Server    │◄─┤   Internal APIs     ├─►│  Plugin System   │       │
│  └────────┬─────────┘  └─────────────────────┘  └────────┬─────────┘       │
│           │                                              │                  │
└───────────┼──────────────────────────────────────────────┼──────────────────┘
            │                                              │
            ▼                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Client Applications                             │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐              │
│  │  Python SDK     │  │   Go SDK        │  │   CLI Tool      │              │
│  │  (Data Sci)     │  │  (Backend Dev)  │  │   (DevOps)      │              │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘              │
└─────────────────────────────────────────────────────────────────────────────┘
            │                                              │
            ▼                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Infrastructure                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │    Redis    │  │ PostgreSQL  │  │    Kafka    │  │ Prometheus  │        │
│  │ (Cache/Q)   │  │ (Storage)   │  │ (Dist.Q)    │  │ (Metrics)   │        │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘        │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 System Components

| Component | Description | Technology |
|-----------|-------------|------------|
| **Core Engine** | HTTP 요청, 스케줄링, 파싱의 핵심 로직 | Go 1.21+ |
| **HTTP Client** | HTTP/1.1, HTTP/2 요청 처리 | net/http, Colly |
| **Browser Engine** | JavaScript 렌더링 | chromedp |
| **Scheduler** | URL 스케줄링 및 Politeness 관리 | Go (custom) |
| **Parser** | HTML/JSON/XML 파싱 | GoQuery, gjson |
| **Middleware Chain** | 요청/응답 처리 파이프라인 | Go (custom) |
| **Plugin System** | 확장 가능한 플러그인 아키텍처 | Go interfaces |
| **gRPC Server** | Python 바인딩용 RPC 서버 | gRPC-Go |
| **Python SDK** | Python 클라이언트 라이브러리 | gRPC-Python |
| **CLI Tool** | 명령줄 인터페이스 | Cobra |

### 2.3 Deployment Configurations

#### 2.3.1 Standalone Mode
```
┌─────────────────────────────┐
│      Single Binary          │
│  ┌───────────────────────┐  │
│  │  Crawler Engine       │  │
│  │  + In-Memory Queue    │  │
│  │  + Local Storage      │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

#### 2.3.2 Server Mode
```
┌─────────────────────────────┐     ┌─────────────────────┐
│    gRPC Server              │◄────┤  Python Client      │
│  ┌───────────────────────┐  │     └─────────────────────┘
│  │  Crawler Engine       │  │     ┌─────────────────────┐
│  │  + Redis Queue        │  │◄────┤  Go Client          │
│  │  + PostgreSQL         │  │     └─────────────────────┘
│  └───────────────────────┘  │
└─────────────────────────────┘
```

#### 2.3.3 Distributed Mode
```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  Worker 1   │  │  Worker 2   │  │  Worker N   │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         │                │                │                  │
│         └────────────────┼────────────────┘                  │
│                          ▼                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                    Kafka Cluster                       │  │
│  └───────────────────────────────────────────────────────┘  │
│         │                │                │                  │
│         ▼                ▼                ▼                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │   Redis     │  │ PostgreSQL  │  │ Prometheus  │         │
│  │  Cluster    │  │  Cluster    │  │             │         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Functional Requirements Specification

### 3.1 Core Engine Requirements (SRS-CORE)

#### SRS-CORE-001: HTTP Client Initialization
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CORE-001 |
| **Title** | HTTP Client Initialization |
| **Traces To** | PRD: FR-101, FR-102, FR-103 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 설정 기반으로 HTTP 클라이언트를 초기화해야 한다.

**Input:**
```go
type HTTPClientConfig struct {
    MaxIdleConns        int           // 최대 유휴 연결 수 (default: 100)
    MaxIdleConnsPerHost int           // 호스트당 최대 유휴 연결 (default: 10)
    IdleConnTimeout     time.Duration // 유휴 연결 타임아웃 (default: 90s)
    TLSHandshakeTimeout time.Duration // TLS 핸드셰이크 타임아웃 (default: 10s)
    DisableCompression  bool          // 압축 비활성화 (default: false)
    DisableKeepAlives   bool          // Keep-Alive 비활성화 (default: false)
    HTTP2Enabled        bool          // HTTP/2 활성화 (default: true)
}
```

**Process:**
1. 설정값 유효성 검증
2. Transport 생성 및 구성
3. HTTP/2 지원 여부에 따른 프로토콜 설정
4. 연결 풀 초기화

**Output:**
```go
type HTTPClient interface {
    Do(req *Request) (*Response, error)
    Close() error
}
```

**Acceptance Criteria:**
- [ ] HTTP/1.1 요청 성공
- [ ] HTTP/2 요청 성공
- [ ] 연결 풀 재사용 확인
- [ ] 타임아웃 동작 확인

---

#### SRS-CORE-002: Request Execution
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CORE-002 |
| **Title** | HTTP Request Execution |
| **Traces To** | PRD: FR-101, FR-104, FR-105 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 HTTP 요청을 실행하고 응답을 반환해야 한다.

**Input:**
```go
type Request struct {
    URL       string              // 요청 URL (required)
    Method    string              // HTTP 메서드 (default: GET)
    Headers   map[string]string   // 커스텀 헤더
    Body      []byte              // 요청 본문
    Timeout   time.Duration       // 요청 타임아웃
    Metadata  map[string]any      // 사용자 메타데이터
}
```

**Process:**
1. URL 유효성 검증
2. 요청 헤더 구성 (User-Agent, Accept 등)
3. 미들웨어 체인 통과 (PreRequest)
4. HTTP 요청 실행
5. 리다이렉트 처리 (설정에 따라)
6. 미들웨어 체인 통과 (PostResponse)
7. 응답 반환

**Output:**
```go
type Response struct {
    Request     *Request          // 원본 요청
    StatusCode  int               // HTTP 상태 코드
    Headers     http.Header       // 응답 헤더
    Body        []byte            // 응답 본문
    ContentType string            // Content-Type
    FetchTime   time.Duration     // 요청 소요 시간
    FinalURL    string            // 최종 URL (리다이렉트 후)
}
```

**Acceptance Criteria:**
- [ ] GET, POST, PUT, DELETE 메서드 지원
- [ ] 커스텀 헤더 전송 확인
- [ ] 리다이렉트 처리 (최대 10회)
- [ ] 타임아웃 발생 시 에러 반환

---

#### SRS-CORE-003: Concurrent Request Management
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CORE-003 |
| **Title** | Concurrent Request Management |
| **Traces To** | PRD: FR-102, NFR-101, NFR-105 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 동시 요청 수를 제한하고 관리해야 한다.

**Input:**
```go
type ConcurrencyConfig struct {
    MaxConcurrency     int  // 전역 최대 동시 요청 (default: 100)
    MaxPerDomain       int  // 도메인당 최대 동시 요청 (default: 5)
    EnableDomainLimit  bool // 도메인별 제한 활성화 (default: true)
}
```

**Process:**
1. 세마포어 기반 동시성 제어
2. 도메인별 동시성 추적
3. 요청 전 세마포어 획득
4. 요청 완료 후 세마포어 해제

**Technical Design:**
```go
type ConcurrencyManager struct {
    globalSem   *semaphore.Weighted
    domainSems  map[string]*semaphore.Weighted
    mu          sync.RWMutex
}

func (m *ConcurrencyManager) Acquire(ctx context.Context, domain string) error
func (m *ConcurrencyManager) Release(domain string)
```

**Acceptance Criteria:**
- [ ] 전역 동시성 제한 동작
- [ ] 도메인별 동시성 제한 동작
- [ ] 제한 초과 시 대기
- [ ] Context 취소 시 즉시 반환

---

#### SRS-CORE-004: Cookie Management
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CORE-004 |
| **Title** | Cookie Management |
| **Traces To** | PRD: FR-106 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 쿠키를 자동으로 관리하고 세션을 유지해야 한다.

**Input:**
```go
type CookieConfig struct {
    Enabled     bool   // 쿠키 관리 활성화 (default: true)
    PersistPath string // 쿠키 저장 경로 (optional)
}
```

**Process:**
1. 응답에서 Set-Cookie 헤더 파싱
2. 쿠키 저장 (도메인/경로별)
3. 요청 시 해당 쿠키 첨부
4. 만료된 쿠키 자동 제거

**Technical Design:**
```go
type CookieJar interface {
    SetCookies(u *url.URL, cookies []*http.Cookie)
    Cookies(u *url.URL) []*http.Cookie
    Clear()
    Save(path string) error
    Load(path string) error
}
```

**Acceptance Criteria:**
- [ ] 쿠키 자동 저장 및 전송
- [ ] 도메인 범위 쿠키 처리
- [ ] Secure/HttpOnly 속성 준수
- [ ] 만료 쿠키 자동 제거

---

#### SRS-CORE-005: Proxy Support
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CORE-005 |
| **Title** | Proxy Support |
| **Traces To** | PRD: FR-107 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 HTTP 및 SOCKS5 프록시를 지원해야 한다.

**Input:**
```go
type ProxyConfig struct {
    URL      string // 프록시 URL (http://, socks5://)
    Username string // 인증 사용자명
    Password string // 인증 비밀번호
}
```

**Process:**
1. 프록시 URL 파싱
2. 프로토콜별 Dialer 구성 (HTTP/SOCKS5)
3. 인증 정보 설정
4. Transport에 프록시 적용

**Acceptance Criteria:**
- [ ] HTTP 프록시 연결
- [ ] SOCKS5 프록시 연결
- [ ] 프록시 인증 지원
- [ ] 프록시 실패 시 에러 반환

---

### 3.2 JavaScript Rendering Requirements (SRS-JS)

#### SRS-JS-001: Browser Pool Management
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-JS-001 |
| **Title** | Browser Pool Management |
| **Traces To** | PRD: FR-201, FR-204 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 chromedp 브라우저 인스턴스 풀을 관리해야 한다.

**Input:**
```go
type BrowserPoolConfig struct {
    PoolSize     int           // 풀 크기 (default: 5)
    Headless     bool          // Headless 모드 (default: true)
    PageTimeout  time.Duration // 페이지 타임아웃 (default: 30s)
    IdleTimeout  time.Duration // 유휴 타임아웃 (default: 5m)
    UserDataDir  string        // 사용자 데이터 디렉토리
}
```

**Process:**
1. 브라우저 풀 초기화 (Lazy Loading)
2. 요청 시 유휴 브라우저 할당
3. 작업 완료 후 브라우저 반환
4. 오류 발생 시 브라우저 재생성
5. 유휴 타임아웃 시 브라우저 정리

**Technical Design:**
```go
type BrowserPool struct {
    pool     chan *Browser
    config   BrowserPoolConfig
    ctx      context.Context
    cancel   context.CancelFunc
}

func (p *BrowserPool) Acquire(ctx context.Context) (*Browser, error)
func (p *BrowserPool) Release(b *Browser)
func (p *BrowserPool) Close() error
```

**Acceptance Criteria:**
- [ ] 풀 크기 제한 동작
- [ ] 브라우저 재사용
- [ ] 오류 브라우저 자동 교체
- [ ] Graceful shutdown

---

#### SRS-JS-002: Page Rendering
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-JS-002 |
| **Title** | Page Rendering with Wait Conditions |
| **Traces To** | PRD: FR-201, FR-202 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 JavaScript를 실행하고 지정된 조건까지 대기해야 한다.

**Input:**
```go
type RenderOptions struct {
    WaitFor     WaitCondition // 대기 조건
    WaitTimeout time.Duration // 대기 타임아웃
    Viewport    *Viewport     // 뷰포트 크기
    UserAgent   string        // User-Agent 오버라이드
}

type WaitCondition interface {
    // Selector: 특정 요소가 나타날 때까지 대기
    // Idle: 네트워크 유휴 상태까지 대기
    // Custom: 사용자 정의 JavaScript 조건
}
```

**Process:**
1. 브라우저 풀에서 인스턴스 획득
2. 페이지 생성 및 뷰포트 설정
3. URL 네비게이션
4. 대기 조건 실행
5. HTML 추출
6. 브라우저 반환

**Technical Design:**
```go
type WaitForSelector struct {
    Selector string
}

type WaitForNetworkIdle struct {
    IdleTime time.Duration
}

type WaitForFunction struct {
    Function string // JavaScript 함수
}
```

**Acceptance Criteria:**
- [ ] 기본 JavaScript 렌더링
- [ ] CSS Selector 대기 조건
- [ ] 네트워크 유휴 대기
- [ ] 커스텀 JavaScript 대기

---

#### SRS-JS-003: Resource Blocking
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-JS-003 |
| **Title** | Resource Blocking |
| **Traces To** | PRD: FR-206 |
| **Priority** | P2 |
| **Status** | Draft |

**Description:**
시스템은 불필요한 리소스 로딩을 차단할 수 있어야 한다.

**Input:**
```go
type ResourceBlockConfig struct {
    BlockImages bool     // 이미지 차단
    BlockMedia  bool     // 미디어 차단
    BlockFonts  bool     // 폰트 차단
    BlockTypes  []string // 추가 차단 타입
}
```

**Process:**
1. 네트워크 인터셉터 설정
2. 요청 타입 확인
3. 차단 대상이면 요청 중단
4. 허용 대상이면 요청 진행

**Acceptance Criteria:**
- [ ] 이미지 차단
- [ ] CSS/JS 이외 리소스 차단
- [ ] 차단으로 인한 페이지 로드 시간 감소

---

#### SRS-JS-004: Screenshot Capture
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-JS-004 |
| **Title** | Screenshot Capture |
| **Traces To** | PRD: FR-205 |
| **Priority** | P2 |
| **Status** | Draft |

**Description:**
시스템은 렌더링된 페이지의 스크린샷을 캡처할 수 있어야 한다.

**Input:**
```go
type ScreenshotOptions struct {
    Format   string // png, jpeg
    Quality  int    // JPEG 품질 (1-100)
    FullPage bool   // 전체 페이지 캡처
    Selector string // 특정 요소만 캡처
}
```

**Output:**
```go
type Screenshot struct {
    Data   []byte
    Format string
    Width  int
    Height int
}
```

**Acceptance Criteria:**
- [ ] PNG 형식 캡처
- [ ] JPEG 형식 캡처
- [ ] 전체 페이지 캡처
- [ ] 특정 요소 캡처

---

### 3.3 URL Management Requirements (SRS-URL)

#### SRS-URL-001: URL Frontier Implementation
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-URL-001 |
| **Title** | URL Frontier with Priority Queue |
| **Traces To** | PRD: FR-301, FR-304 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 우선순위 기반 URL 큐(Frontier)를 구현해야 한다.

**Input:**
```go
type FrontierConfig struct {
    MaxSize        int           // 최대 큐 크기
    DefaultPriority int          // 기본 우선순위 (1-10)
    EnableBackQueue bool         // 도메인별 Back Queue 활성화
}

type URLEntry struct {
    URL       string
    Priority  int       // 1 (lowest) - 10 (highest)
    Depth     int       // 크롤링 깊이
    Domain    string    // 추출된 도메인
    DiscoveredAt time.Time
    Metadata  map[string]any
}
```

**Process:**
1. URL 추가 요청 수신
2. URL 정규화
3. 중복 검사
4. 우선순위 계산
5. 도메인별 Back Queue에 삽입
6. Front Queue에서 다음 URL 선택

**Technical Design:**
```go
type Frontier interface {
    Add(ctx context.Context, entry *URLEntry) error
    AddBatch(ctx context.Context, entries []*URLEntry) error
    Next(ctx context.Context) (*URLEntry, error)
    Size() int64
    Close() error
}

// Dual Queue Architecture
// Front Queues: 우선순위 기반 (Heap)
// Back Queues: 도메인별 (FIFO per domain)
```

**Acceptance Criteria:**
- [ ] 우선순위 기반 URL 반환
- [ ] 도메인별 Politeness 유지
- [ ] 큐 크기 제한 동작
- [ ] 빈 큐에서 블로킹

---

#### SRS-URL-002: URL Canonicalization
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-URL-002 |
| **Title** | URL Canonicalization |
| **Traces To** | PRD: FR-302 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 URL을 정규화하여 일관된 형식으로 변환해야 한다.

**Input:**
```go
type CanonicalizeOptions struct {
    RemoveFragment     bool // # 이후 제거
    RemoveDefaultPort  bool // 기본 포트 제거
    LowercaseHost      bool // 호스트 소문자
    SortQueryParams    bool // 쿼리 파라미터 정렬
    RemoveTrailingSlash bool // 후행 슬래시 제거
    DecodePercent      bool // 퍼센트 인코딩 디코드
}
```

**Process:**
1. URL 파싱
2. 스킴 정규화 (http/https)
3. 호스트 소문자 변환
4. 기본 포트 제거
5. 경로 정규화 (../, ./ 처리)
6. 쿼리 파라미터 정렬
7. Fragment 제거

**Examples:**
```
Input:  HTTP://Example.COM:80/path/../page?b=2&a=1#section
Output: http://example.com/page?a=1&b=2
```

**Acceptance Criteria:**
- [ ] 스킴 소문자 변환
- [ ] 호스트 소문자 변환
- [ ] 기본 포트 제거
- [ ] 경로 정규화
- [ ] 쿼리 파라미터 정렬

---

#### SRS-URL-003: URL Deduplication
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-URL-003 |
| **Title** | URL Deduplication |
| **Traces To** | PRD: FR-303 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 중복 URL을 감지하고 제거해야 한다.

**Input:**
```go
type DeduplicationConfig struct {
    Backend    string // memory, redis, bloom
    Capacity   int    // Bloom filter 용량
    ErrorRate  float64 // Bloom filter 오류율
    RedisURL   string // Redis URL (backend=redis)
}
```

**Process:**
1. URL 정규화
2. URL 해시 생성 (SHA-256)
3. 해시 존재 여부 확인
4. 신규 URL이면 해시 저장 및 통과
5. 중복 URL이면 필터링

**Technical Design:**
```go
type Deduplicator interface {
    IsSeen(url string) (bool, error)
    MarkSeen(url string) error
    Clear() error
    Size() int64
}

// Implementations
type MemoryDedup struct { seen sync.Map }
type RedisDedup struct { client *redis.Client }
type BloomDedup struct { filter *bloom.BloomFilter }
```

**Acceptance Criteria:**
- [ ] 메모리 기반 중복 검사
- [ ] Redis 기반 중복 검사
- [ ] Bloom Filter 기반 중복 검사
- [ ] 정규화 후 중복 검사

---

#### SRS-URL-004: URL Filtering
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-URL-004 |
| **Title** | URL Filtering Rules |
| **Traces To** | PRD: FR-306 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 규칙 기반으로 URL을 필터링해야 한다.

**Input:**
```go
type FilterConfig struct {
    AllowedDomains    []string // 허용 도메인
    DisallowedDomains []string // 차단 도메인
    AllowedPaths      []string // 허용 경로 패턴 (glob)
    DisallowedPaths   []string // 차단 경로 패턴 (glob)
    MaxDepth          int      // 최대 크롤링 깊이
    AllowedSchemes    []string // 허용 스킴 (http, https)
}
```

**Process:**
1. 스킴 검사
2. 도메인 허용/차단 검사
3. 경로 패턴 매칭
4. 깊이 검사
5. 통과/차단 결정

**Acceptance Criteria:**
- [ ] 도메인 기반 필터링
- [ ] 경로 패턴 필터링
- [ ] 깊이 기반 필터링
- [ ] 복합 조건 필터링

---

### 3.4 Data Extraction Requirements (SRS-EXT)

#### SRS-EXT-001: CSS Selector Extraction
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-EXT-001 |
| **Title** | CSS Selector Based Extraction |
| **Traces To** | PRD: FR-401 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 CSS Selector를 사용하여 HTML 요소를 추출해야 한다.

**Input:**
```go
type CSSExtractOptions struct {
    Selector string // CSS Selector
    Attr     string // 추출할 속성 (없으면 텍스트)
    All      bool   // 모든 매칭 요소
}
```

**Process:**
1. HTML 문서 파싱
2. CSS Selector 실행
3. 요소 추출 (단일/다중)
4. 속성 또는 텍스트 반환

**Technical Design:**
```go
type HTMLElement struct {
    Tag        string
    Text       string
    HTML       string
    Attributes map[string]string
}

func (e *HTMLElement) CSS(selector string) *HTMLElement
func (e *HTMLElement) CSSAll(selector string) []*HTMLElement
func (e *HTMLElement) Attr(name string) string
func (e *HTMLElement) Text() string
```

**Acceptance Criteria:**
- [ ] 기본 Selector 지원 (tag, .class, #id)
- [ ] 복합 Selector 지원 (div.class > a)
- [ ] 속성 Selector 지원 ([attr=value])
- [ ] 다중 요소 추출

---

#### SRS-EXT-002: XPath Extraction
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-EXT-002 |
| **Title** | XPath Based Extraction |
| **Traces To** | PRD: FR-402 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 XPath를 사용하여 XML/HTML 요소를 추출해야 한다.

**Input:**
```go
type XPathExtractOptions struct {
    Expression string // XPath 표현식
}
```

**Process:**
1. HTML/XML 문서 파싱
2. XPath 표현식 컴파일
3. 노드 매칭
4. 결과 반환 (문자열/노드)

**Acceptance Criteria:**
- [ ] 기본 XPath 지원
- [ ] Predicate 지원
- [ ] 함수 지원 (contains, text())
- [ ] 축(axis) 지원

---

#### SRS-EXT-003: JSON Extraction
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-EXT-003 |
| **Title** | JSON Path Extraction |
| **Traces To** | PRD: FR-404 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 JSON 데이터에서 값을 추출해야 한다.

**Input:**
```go
type JSONExtractOptions struct {
    Path string // JSONPath 또는 gjson path
}
```

**Process:**
1. JSON 유효성 검증
2. 경로 표현식 파싱
3. 값 추출
4. 타입 변환 (필요시)

**Technical Design:**
```go
func (r *Response) JSON(path string) gjson.Result
func (r *Response) JSONArray(path string) []gjson.Result
func (r *Response) JSONMap() map[string]any
```

**Acceptance Criteria:**
- [ ] 기본 경로 접근 (data.items)
- [ ] 배열 인덱스 (items.0)
- [ ] 와일드카드 (items.*.name)
- [ ] 조건 필터 (items.#(price>100))

---

#### SRS-EXT-004: Regex Extraction
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-EXT-004 |
| **Title** | Regular Expression Extraction |
| **Traces To** | PRD: FR-403 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 정규표현식을 사용하여 텍스트를 추출해야 한다.

**Input:**
```go
type RegexExtractOptions struct {
    Pattern string // 정규표현식
    Group   int    // 캡처 그룹 인덱스
}
```

**Process:**
1. 정규표현식 컴파일
2. 텍스트 매칭
3. 캡처 그룹 추출
4. 결과 반환

**Acceptance Criteria:**
- [ ] 기본 패턴 매칭
- [ ] 캡처 그룹 추출
- [ ] 명명된 그룹 지원
- [ ] 다중 매칭

---

#### SRS-EXT-005: Encoding Detection
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-EXT-005 |
| **Title** | Automatic Encoding Detection |
| **Traces To** | PRD: FR-406 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 응답의 문자 인코딩을 자동으로 감지해야 한다.

**Process:**
1. Content-Type 헤더의 charset 확인
2. BOM (Byte Order Mark) 검사
3. HTML meta 태그 검사
4. 통계적 인코딩 감지
5. UTF-8로 변환

**Supported Encodings:**
- UTF-8, UTF-16, UTF-32
- ISO-8859-1, ISO-8859-15
- Windows-1252
- EUC-KR, EUC-JP
- Shift_JIS
- GB2312, GBK, GB18030
- Big5

**Acceptance Criteria:**
- [ ] UTF-8 자동 감지
- [ ] 한국어 인코딩 감지 (EUC-KR)
- [ ] 일본어 인코딩 감지
- [ ] 중국어 인코딩 감지

---

### 3.5 Middleware Requirements (SRS-MW)

#### SRS-MW-001: Middleware Chain Architecture
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-001 |
| **Title** | Middleware Chain Architecture |
| **Traces To** | PRD: FR-501, FR-502, FR-505 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 체인 패턴 기반의 미들웨어 시스템을 제공해야 한다.

**Technical Design:**
```go
type Middleware interface {
    ProcessRequest(ctx context.Context, req *Request) error
    ProcessResponse(ctx context.Context, resp *Response) error
    Name() string
    Priority() int // 낮을수록 먼저 실행
}

type Chain struct {
    middlewares []Middleware
}

func (c *Chain) Add(m Middleware)
func (c *Chain) ProcessRequest(ctx context.Context, req *Request) error
func (c *Chain) ProcessResponse(ctx context.Context, resp *Response) error
```

**Execution Order:**
```
Request:  Middleware1 → Middleware2 → Middleware3 → HTTP Client
Response: Middleware3 → Middleware2 → Middleware1 → Caller
```

**Acceptance Criteria:**
- [ ] 우선순위 기반 실행 순서
- [ ] 요청/응답 양방향 처리
- [ ] 미들웨어 동적 추가/제거
- [ ] 에러 발생 시 체인 중단

---

#### SRS-MW-002: Retry Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-002 |
| **Title** | Retry Middleware with Exponential Backoff |
| **Traces To** | PRD: FR-501 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 실패한 요청을 자동으로 재시도해야 한다.

**Input:**
```go
type RetryConfig struct {
    MaxRetries    int           // 최대 재시도 횟수 (default: 3)
    BaseDelay     time.Duration // 기본 지연 (default: 1s)
    MaxDelay      time.Duration // 최대 지연 (default: 30s)
    Multiplier    float64       // 지연 배수 (default: 2.0)
    Jitter        float64       // 지터 비율 (default: 0.1)
    RetryOnStatus []int         // 재시도 상태 코드
}
```

**Process:**
1. 응답 상태 코드 확인
2. 재시도 대상 여부 판단
3. 재시도 횟수 초과 시 에러 반환
4. Exponential Backoff 계산
5. Jitter 적용
6. 지연 후 재요청

**Formula:**
```
delay = min(BaseDelay * Multiplier^attempt, MaxDelay) * (1 ± Jitter)
```

**Retry Status Codes (Default):**
- 429 (Too Many Requests)
- 500 (Internal Server Error)
- 502 (Bad Gateway)
- 503 (Service Unavailable)
- 504 (Gateway Timeout)

**Acceptance Criteria:**
- [ ] 설정된 횟수만큼 재시도
- [ ] Exponential Backoff 적용
- [ ] Jitter 적용
- [ ] 최대 지연 제한

---

#### SRS-MW-003: Rate Limit Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-003 |
| **Title** | Rate Limit Middleware |
| **Traces To** | PRD: FR-502, NFR-501, NFR-502 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 요청 빈도를 제한해야 한다.

**Input:**
```go
type RateLimitConfig struct {
    RequestsPerSecond float64 // 초당 요청 수 (global)
    BurstSize         int     // 버스트 크기
    PerDomain         bool    // 도메인별 적용
    RespectCrawlDelay bool    // robots.txt Crawl-delay 준수
}
```

**Process:**
1. 도메인 추출
2. Rate Limiter 조회/생성
3. 토큰 획득 대기
4. 요청 진행

**Technical Design:**
```go
type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func (r *RateLimiter) Wait(ctx context.Context, domain string) error
func (r *RateLimiter) SetLimit(domain string, rps float64)
```

**Acceptance Criteria:**
- [ ] 전역 Rate Limiting
- [ ] 도메인별 Rate Limiting
- [ ] robots.txt Crawl-delay 준수
- [ ] 429 응답 시 자동 조절

---

#### SRS-MW-004: Robots.txt Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-004 |
| **Title** | Robots.txt Compliance Middleware |
| **Traces To** | PRD: FR-505, NFR-501 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 robots.txt 규칙을 자동으로 준수해야 한다.

**Input:**
```go
type RobotsConfig struct {
    Enabled    bool          // 활성화 (default: true)
    UserAgent  string        // User-Agent 이름
    CacheTTL   time.Duration // 캐시 TTL (default: 24h)
}
```

**Process:**
1. 대상 URL의 robots.txt 조회 (캐시 확인)
2. robots.txt 파싱
3. User-Agent 매칭
4. Allow/Disallow 규칙 적용
5. Crawl-delay 추출 및 적용

**Technical Design:**
```go
type RobotsChecker interface {
    IsAllowed(userAgent, url string) (bool, error)
    GetCrawlDelay(userAgent, domain string) (time.Duration, error)
}

type RobotsCache struct {
    cache map[string]*RobotsData
    ttl   time.Duration
}
```

**Acceptance Criteria:**
- [ ] robots.txt 자동 조회
- [ ] Allow/Disallow 규칙 적용
- [ ] Crawl-delay 적용
- [ ] 캐싱 동작

---

#### SRS-MW-005: Proxy Rotation Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-005 |
| **Title** | Proxy Rotation Middleware |
| **Traces To** | PRD: FR-503 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 여러 프록시를 순환하며 사용해야 한다.

**Input:**
```go
type ProxyRotationConfig struct {
    Proxies       []string // 프록시 URL 목록
    RotationMode  string   // round-robin, random, health-based
    HealthCheck   bool     // 헬스체크 활성화
    FailThreshold int      // 실패 임계값
}
```

**Process:**
1. 다음 프록시 선택 (회전 모드에 따라)
2. 프록시 건강 상태 확인
3. 요청에 프록시 적용
4. 응답 결과에 따라 건강 상태 업데이트

**Acceptance Criteria:**
- [ ] Round-Robin 회전
- [ ] Random 회전
- [ ] 건강 기반 회전
- [ ] 실패 프록시 자동 제외

---

#### SRS-MW-006: User-Agent Rotation Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-006 |
| **Title** | User-Agent Rotation Middleware |
| **Traces To** | PRD: FR-504 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 다양한 User-Agent를 순환하며 사용해야 한다.

**Input:**
```go
type UserAgentConfig struct {
    UserAgents    []string // User-Agent 목록
    RotationMode  string   // round-robin, random, per-domain
}
```

**Process:**
1. 회전 모드에 따라 User-Agent 선택
2. 요청 헤더에 User-Agent 설정

**Default User-Agents:**
```go
var DefaultUserAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36...",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36...",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36...",
}
```

**Acceptance Criteria:**
- [ ] User-Agent 순환
- [ ] 도메인별 고정 옵션
- [ ] 커스텀 User-Agent 추가

---

#### SRS-MW-007: Authentication Middleware
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-MW-007 |
| **Title** | Authentication Middleware |
| **Traces To** | PRD: FR-506 |
| **Priority** | P1 |
| **Status** | Draft |

**Description:**
시스템은 다양한 인증 방식을 지원해야 한다.

**Input:**
```go
type AuthConfig struct {
    Type     string // basic, bearer, oauth2
    Username string // Basic Auth
    Password string // Basic Auth
    Token    string // Bearer Token
    OAuth2   *OAuth2Config
}

type OAuth2Config struct {
    ClientID     string
    ClientSecret string
    TokenURL     string
    Scopes       []string
}
```

**Process:**
1. 인증 타입 확인
2. 인증 헤더 생성
3. 요청에 헤더 추가
4. (OAuth2) 토큰 갱신 필요시 자동 갱신

**Acceptance Criteria:**
- [ ] Basic Authentication
- [ ] Bearer Token
- [ ] OAuth2 Client Credentials
- [ ] 토큰 자동 갱신

---

### 3.6 Storage Requirements (SRS-STOR)

#### SRS-STOR-001: Storage Plugin Interface
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-STOR-001 |
| **Title** | Storage Plugin Interface |
| **Traces To** | PRD: FR-601, FR-602, FR-603 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 플러그인 기반 저장소 시스템을 제공해야 한다.

**Technical Design:**
```go
type StoragePlugin interface {
    Plugin
    Store(ctx context.Context, data *CrawledData) error
    StoreBatch(ctx context.Context, data []*CrawledData) error
    Query(ctx context.Context, query *StorageQuery) ([]*CrawledData, error)
    Delete(ctx context.Context, query *StorageQuery) (int64, error)
}

type CrawledData struct {
    ID          string
    URL         string
    StatusCode  int
    ContentType string
    Content     []byte
    ExtractedData map[string]any
    Metadata    map[string]any
    CrawledAt   time.Time
}

type StorageQuery struct {
    URLs        []string
    Domains     []string
    DateRange   *DateRange
    Limit       int
    Offset      int
}
```

**Acceptance Criteria:**
- [ ] 플러그인 인터페이스 정의
- [ ] 단일/배치 저장
- [ ] 쿼리 지원
- [ ] 삭제 지원

---

#### SRS-STOR-002: PostgreSQL Storage Plugin
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-STOR-002 |
| **Title** | PostgreSQL Storage Plugin |
| **Traces To** | PRD: FR-601 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 PostgreSQL 저장소 플러그인을 제공해야 한다.

**Schema:**
```sql
CREATE TABLE crawled_pages (
    id          BIGSERIAL PRIMARY KEY,
    url         TEXT UNIQUE NOT NULL,
    url_hash    TEXT NOT NULL,
    status_code INTEGER,
    content_type TEXT,
    content     BYTEA,
    extracted   JSONB,
    metadata    JSONB,
    crawled_at  TIMESTAMP DEFAULT NOW(),
    created_at  TIMESTAMP DEFAULT NOW(),
    updated_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_crawled_pages_url_hash ON crawled_pages(url_hash);
CREATE INDEX idx_crawled_pages_crawled_at ON crawled_pages(crawled_at);
CREATE INDEX idx_crawled_pages_domain ON crawled_pages((metadata->>'domain'));
```

**Configuration:**
```go
type PostgresConfig struct {
    URL             string        // PostgreSQL URL
    MaxConnections  int           // 최대 연결 수
    TableName       string        // 테이블 이름
    BatchSize       int           // 배치 크기
    EnableUpsert    bool          // UPSERT 활성화
}
```

**Acceptance Criteria:**
- [ ] 연결 풀 관리
- [ ] UPSERT 동작
- [ ] 배치 삽입
- [ ] JSONB 쿼리

---

#### SRS-STOR-003: Redis Integration
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-STOR-003 |
| **Title** | Redis Integration |
| **Traces To** | PRD: FR-603 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 Redis를 캐시 및 큐로 사용해야 한다.

**Use Cases:**
1. **URL Frontier**: Sorted Set 기반 우선순위 큐
2. **Deduplication**: Set 기반 중복 검사
3. **Rate Limiting**: Token Bucket 구현
4. **Cache**: 응답 캐싱

**Configuration:**
```go
type RedisConfig struct {
    URL          string        // Redis URL
    Password     string        // 비밀번호
    DB           int           // 데이터베이스 번호
    PoolSize     int           // 연결 풀 크기
    KeyPrefix    string        // 키 접두사
}
```

**Key Patterns:**
```
{prefix}:frontier:queue       - URL 큐 (Sorted Set)
{prefix}:frontier:seen        - 방문 URL (Set)
{prefix}:ratelimit:{domain}   - Rate Limit 상태 (String)
{prefix}:cache:{url_hash}     - 응답 캐시 (Hash)
```

**Acceptance Criteria:**
- [ ] URL Frontier 구현
- [ ] 중복 제거 구현
- [ ] Rate Limiting 구현
- [ ] 응답 캐싱

---

#### SRS-STOR-004: File Export
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-STOR-004 |
| **Title** | JSON/CSV File Export |
| **Traces To** | PRD: FR-604 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 크롤링 결과를 JSON/CSV 파일로 내보낼 수 있어야 한다.

**Configuration:**
```go
type FileExportConfig struct {
    Format      string // json, jsonl, csv
    OutputPath  string // 출력 경로
    MaxFileSize int64  // 최대 파일 크기
    Compression string // none, gzip
}
```

**Process:**
1. 데이터 직렬화
2. 파일 크기 확인 (롤오버)
3. 압축 적용 (선택)
4. 파일 쓰기

**Acceptance Criteria:**
- [ ] JSON 내보내기
- [ ] JSON Lines 내보내기
- [ ] CSV 내보내기
- [ ] gzip 압축

---

### 3.7 Python Bindings Requirements (SRS-PY)

#### SRS-PY-001: gRPC Service Definition
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PY-001 |
| **Title** | gRPC Service Definition |
| **Traces To** | PRD: FR-701 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 gRPC 기반 Python 바인딩을 위한 서비스를 정의해야 한다.

**Protocol Buffer Definition:**
```protobuf
syntax = "proto3";
package crawler.v1;

service CrawlerService {
    // Single URL crawling
    rpc Crawl(CrawlRequest) returns (CrawlResponse);

    // Batch URL crawling
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);

    // Start a crawl job
    rpc StartJob(StartJobRequest) returns (JobResponse);

    // Get job status
    rpc GetJobStatus(JobStatusRequest) returns (JobResponse);

    // Stop a job
    rpc StopJob(StopJobRequest) returns (JobResponse);

    // Stream crawled data
    rpc StreamResults(StreamRequest) returns (stream CrawlResponse);
}

message CrawlRequest {
    string url = 1;
    CrawlOptions options = 2;
}

message CrawlResponse {
    string url = 1;
    int32 status_code = 2;
    bytes content = 3;
    string content_type = 4;
    map<string, string> headers = 5;
    int64 fetch_time_ms = 6;
    string error = 7;
}

message CrawlOptions {
    bool render_js = 1;
    int32 timeout_ms = 2;
    map<string, string> headers = 3;
    string proxy = 4;
    int32 max_retries = 5;
}

message CrawlBatchRequest {
    repeated string urls = 1;
    CrawlOptions options = 2;
    int32 concurrency = 3;
}
```

**Acceptance Criteria:**
- [ ] Proto 파일 정의
- [ ] Go 서버 구현
- [ ] Python 클라이언트 생성
- [ ] 양방향 스트리밍

---

#### SRS-PY-002: Synchronous Python Client
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PY-002 |
| **Title** | Synchronous Python Client |
| **Traces To** | PRD: FR-701, FR-702 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 동기 방식의 Python 클라이언트를 제공해야 한다.

**Interface:**
```python
from typing import Optional, Dict, List, Iterator
from dataclasses import dataclass

@dataclass
class CrawlResult:
    url: str
    status_code: int
    content: bytes
    text: str
    content_type: str
    headers: Dict[str, str]
    fetch_time_ms: int
    error: Optional[str]
    success: bool

@dataclass
class CrawlOptions:
    render_js: bool = False
    timeout_ms: int = 30000
    headers: Optional[Dict[str, str]] = None
    proxy: Optional[str] = None
    max_retries: int = 3

class CrawlerClient:
    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        *,
        secure: bool = False,
        credentials: Optional[grpc.ChannelCredentials] = None,
    ) -> None: ...

    def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
    ) -> CrawlResult: ...

    def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> Iterator[CrawlResult]: ...

    def close(self) -> None: ...

    def __enter__(self) -> "CrawlerClient": ...
    def __exit__(self, *args) -> None: ...
```

**Acceptance Criteria:**
- [ ] 단일 URL 크롤링
- [ ] 배치 크롤링
- [ ] Context Manager 지원
- [ ] 에러 처리

---

#### SRS-PY-003: Asynchronous Python Client
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PY-003 |
| **Title** | Asynchronous Python Client |
| **Traces To** | PRD: FR-702 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 비동기 방식의 Python 클라이언트를 제공해야 한다.

**Interface:**
```python
from typing import AsyncIterator

class AsyncCrawlerClient:
    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        *,
        secure: bool = False,
    ) -> None: ...

    async def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
    ) -> CrawlResult: ...

    async def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> AsyncIterator[CrawlResult]: ...

    async def close(self) -> None: ...

    async def __aenter__(self) -> "AsyncCrawlerClient": ...
    async def __aexit__(self, *args) -> None: ...
```

**Usage Example:**
```python
async def main():
    async with AsyncCrawlerClient() as client:
        result = await client.crawl("https://example.com")
        print(result.text)

        async for result in client.crawl_batch(urls):
            print(f"{result.url}: {result.status_code}")
```

**Acceptance Criteria:**
- [ ] asyncio 호환
- [ ] 비동기 스트리밍
- [ ] 동시 요청 처리
- [ ] Graceful shutdown

---

#### SRS-PY-004: Type Hints Support
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PY-004 |
| **Title** | Complete Type Hints Support |
| **Traces To** | PRD: FR-703 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 완전한 Python 타입 힌트를 제공해야 한다.

**Requirements:**
1. 모든 공개 API에 타입 힌트
2. `py.typed` 마커 파일 포함
3. mypy strict 모드 통과
4. stub 파일 (.pyi) 제공

**Acceptance Criteria:**
- [ ] 모든 함수/메서드 타입 힌트
- [ ] Generic 타입 지원
- [ ] mypy 검증 통과
- [ ] IDE 자동완성 지원

---

### 3.8 CLI Requirements (SRS-CLI)

#### SRS-CLI-001: CLI Framework
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CLI-001 |
| **Title** | CLI Framework Setup |
| **Traces To** | PRD: FR-801-806 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 Cobra 기반 CLI 프레임워크를 제공해야 한다.

**Command Structure:**
```
crawler
├── init <project-name>        # 프로젝트 초기화
├── run [spider]               # 크롤러 실행
├── crawl <url>                # 단일 URL 크롤링
├── server                     # gRPC 서버 시작
│   ├── start                  # 서버 시작
│   └── stop                   # 서버 중지
├── job                        # 작업 관리
│   ├── start                  # 작업 시작
│   ├── status <job-id>        # 작업 상태
│   ├── stop <job-id>          # 작업 중지
│   └── list                   # 작업 목록
├── test [spider]              # 테스트 실행
├── benchmark                  # 벤치마크 실행
├── config                     # 설정 관리
│   ├── show                   # 현재 설정 표시
│   └── validate               # 설정 유효성 검사
└── version                    # 버전 정보
```

**Global Flags:**
```
--config, -c    설정 파일 경로
--verbose, -v   상세 출력
--quiet, -q     최소 출력
--debug         디버그 모드
--no-color      색상 비활성화
```

**Acceptance Criteria:**
- [ ] 모든 명령어 구현
- [ ] 도움말 메시지
- [ ] 자동완성 스크립트
- [ ] 설정 파일 로드

---

#### SRS-CLI-002: Init Command
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CLI-002 |
| **Title** | Project Initialization Command |
| **Traces To** | PRD: FR-801 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 새 크롤러 프로젝트를 초기화하는 명령을 제공해야 한다.

**Usage:**
```bash
crawler init my-project [--template basic|ecommerce|news|api] [--language go|python]
```

**Generated Structure (Go):**
```
my-project/
├── main.go
├── crawler.yaml
├── spiders/
│   └── example_spider.go
├── items/
│   └── items.go
├── pipelines/
│   └── json_pipeline.go
├── go.mod
└── README.md
```

**Generated Structure (Python):**
```
my-project/
├── main.py
├── crawler.yaml
├── spiders/
│   └── example_spider.py
├── items/
│   └── items.py
├── pipelines/
│   └── json_pipeline.py
├── requirements.txt
└── README.md
```

**Acceptance Criteria:**
- [ ] Go 프로젝트 생성
- [ ] Python 프로젝트 생성
- [ ] 템플릿 선택
- [ ] 설정 파일 생성

---

#### SRS-CLI-003: Crawl Command
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-CLI-003 |
| **Title** | Single URL Crawl Command |
| **Traces To** | PRD: FR-803 |
| **Priority** | P0 |
| **Status** | Draft |

**Description:**
시스템은 단일 URL을 빠르게 크롤링하는 명령을 제공해야 한다.

**Usage:**
```bash
crawler crawl <url> [flags]

Flags:
  --render-js          JavaScript 렌더링
  --timeout <duration> 타임아웃 (default: 30s)
  --header, -H         커스텀 헤더 (반복 가능)
  --proxy              프록시 URL
  --json               JSON 출력
  --show-headers       응답 헤더 표시
  --output, -o         출력 파일
```

**Output Example:**
```
URL:          https://example.com
Status:       200
Content-Type: text/html; charset=utf-8
Size:         1256 bytes
Fetch Time:   234ms

Content Preview:
<!DOCTYPE html>
<html>
<head><title>Example Domain</title>...
```

**Acceptance Criteria:**
- [ ] 기본 크롤링
- [ ] JavaScript 렌더링 옵션
- [ ] JSON 출력 포맷
- [ ] 파일 저장

---

---

## 4. Non-Functional Requirements Specification

### 4.1 Performance Requirements (SRS-PERF)

#### SRS-PERF-001: Throughput
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PERF-001 |
| **Title** | Request Throughput |
| **Traces To** | PRD: NFR-101 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 단일 노드에서 5,000 req/s 이상의 처리량을 달성해야 한다.

**Conditions:**
- 하드웨어: 8 vCPU, 16GB RAM
- 네트워크: 1Gbps
- 대상: 정적 HTML (평균 10KB)
- 동시성: 100 goroutines

**Measurement:**
- 벤치마크 도구: wrk, hey
- 측정 기간: 60초
- 측정 지표: 초당 완료된 요청 수

**Acceptance Criteria:**
- [ ] 5,000 req/s 달성
- [ ] P99 latency < 1s
- [ ] 에러율 < 1%

---

#### SRS-PERF-002: Memory Usage
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PERF-002 |
| **Title** | Memory Efficiency |
| **Traces To** | PRD: NFR-103 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 기본 설정에서 100MB 미만의 메모리를 사용해야 한다.

**Conditions:**
- 동시성: 10 goroutines
- URL 큐: 1,000 URLs
- 브라우저: 비활성화

**Measurement:**
- 도구: Go runtime.MemStats, pprof
- 측정 지표: HeapAlloc, HeapInuse

**Acceptance Criteria:**
- [ ] 기본 설정 100MB 미만
- [ ] 메모리 누수 없음 (24시간 테스트)
- [ ] 브라우저 모드 500MB 미만

---

#### SRS-PERF-003: Latency
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-PERF-003 |
| **Title** | Response Latency |
| **Traces To** | PRD: NFR-102 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템의 평균 응답 시간은 500ms 미만이어야 한다.

**Latency Budget:**
| Component | Budget |
|-----------|--------|
| DNS Resolution | 50ms |
| TCP Connection | 50ms |
| TLS Handshake | 100ms |
| Request Send | 10ms |
| Server Processing | 200ms |
| Response Receive | 50ms |
| SDK Processing | 40ms |
| **Total** | **500ms** |

**Acceptance Criteria:**
- [ ] 평균 latency < 500ms
- [ ] P95 latency < 800ms
- [ ] P99 latency < 1000ms

---

### 4.2 Security Requirements (SRS-SEC)

#### SRS-SEC-001: TLS Configuration
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-SEC-001 |
| **Title** | TLS Security Configuration |
| **Traces To** | PRD: NFR-401 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 TLS 1.2 이상을 사용해야 한다.

**Configuration:**
```go
type TLSConfig struct {
    MinVersion uint16   // tls.VersionTLS12
    MaxVersion uint16   // tls.VersionTLS13
    CipherSuites []uint16 // 허용된 암호화 스위트
    InsecureSkipVerify bool // 인증서 검증 스킵 (개발용)
}
```

**Allowed Cipher Suites:**
- TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
- TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
- TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256

**Acceptance Criteria:**
- [ ] TLS 1.2+ 강제
- [ ] 약한 암호화 비활성화
- [ ] 인증서 검증 기본 활성화

---

#### SRS-SEC-002: Credential Protection
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-SEC-002 |
| **Title** | Credential Protection |
| **Traces To** | PRD: NFR-402 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 인증 정보를 안전하게 관리해야 한다.

**Requirements:**
1. 인증 정보는 환경 변수 또는 시크릿 매니저에서 로드
2. 설정 파일에 평문 비밀번호 금지
3. 메모리에서 사용 후 즉시 제거 (가능한 경우)

**Configuration:**
```yaml
# crawler.yaml - 환경 변수 참조
auth:
  basic:
    username: ${CRAWLER_USERNAME}
    password: ${CRAWLER_PASSWORD}
  bearer:
    token: ${CRAWLER_API_TOKEN}
```

**Acceptance Criteria:**
- [ ] 환경 변수 지원
- [ ] AWS Secrets Manager 지원
- [ ] HashiCorp Vault 지원

---

#### SRS-SEC-003: Log Sanitization
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-SEC-003 |
| **Title** | Sensitive Data Log Sanitization |
| **Traces To** | PRD: NFR-403 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 로그에서 민감한 정보를 자동으로 마스킹해야 한다.

**Sanitized Fields:**
- Authorization 헤더
- Cookie 헤더
- 비밀번호 파라미터
- API 키
- 토큰

**Masking Pattern:**
```
Before: Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
After:  Authorization: Bearer [REDACTED]
```

**Acceptance Criteria:**
- [ ] Authorization 헤더 마스킹
- [ ] Cookie 값 마스킹
- [ ] URL 쿼리 파라미터 마스킹

---

### 4.3 Reliability Requirements (SRS-REL)

#### SRS-REL-001: Error Recovery
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-REL-001 |
| **Title** | Error Recovery Mechanism |
| **Traces To** | PRD: NFR-303 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 오류 발생 시 자동으로 복구해야 한다.

**Recovery Scenarios:**
| Error Type | Recovery Action |
|------------|-----------------|
| Network timeout | Retry with backoff |
| DNS failure | Retry after delay |
| 5xx response | Retry with backoff |
| Connection reset | Retry immediately |
| Browser crash | Restart browser |
| OOM (Out of Memory) | Reduce concurrency |

**Acceptance Criteria:**
- [ ] 자동 재시도
- [ ] 브라우저 자동 복구
- [ ] 리소스 압박 시 자동 조절

---

#### SRS-REL-002: Graceful Shutdown
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-REL-002 |
| **Title** | Graceful Shutdown |
| **Traces To** | PRD: NFR-301 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 종료 신호 수신 시 정상적으로 종료해야 한다.

**Shutdown Sequence:**
1. 새 요청 수신 중단
2. 진행 중인 요청 완료 대기 (타임아웃 적용)
3. URL 큐 상태 저장
4. 브라우저 인스턴스 정리
5. 데이터베이스 연결 종료
6. 메트릭 최종 플러시
7. 종료

**Signals:**
- SIGINT (Ctrl+C)
- SIGTERM (kill)

**Acceptance Criteria:**
- [ ] SIGINT/SIGTERM 처리
- [ ] 진행 중인 작업 완료
- [ ] 상태 저장
- [ ] 30초 내 종료

---

### 4.4 Observability Requirements (SRS-OBS)

#### SRS-OBS-001: Prometheus Metrics
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-OBS-001 |
| **Title** | Prometheus Metrics Export |
| **Traces To** | PRD: NFR-604 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 Prometheus 형식으로 메트릭을 노출해야 한다.

**Metrics:**
```prometheus
# Counter: 총 요청 수
crawler_requests_total{status="success|error", domain="example.com"}

# Counter: HTTP 상태 코드별 응답
crawler_responses_total{status_code="200|404|500", domain="example.com"}

# Histogram: 요청 지연 시간
crawler_request_duration_seconds{domain="example.com"}

# Gauge: 현재 큐 크기
crawler_queue_size{type="frontier|pending"}

# Gauge: 활성 워커 수
crawler_active_workers

# Counter: 바이트 수신량
crawler_bytes_received_total

# Gauge: 메모리 사용량
crawler_memory_bytes{type="heap|stack"}
```

**Endpoint:** `/metrics`

**Acceptance Criteria:**
- [ ] Prometheus 형식 출력
- [ ] 레이블 기반 집계
- [ ] 히스토그램 버킷

---

#### SRS-OBS-002: Structured Logging
| Attribute | Value |
|-----------|-------|
| **ID** | SRS-OBS-002 |
| **Title** | Structured JSON Logging |
| **Traces To** | PRD: NFR-603 |
| **Priority** | P0 |
| **Status** | Draft |

**Requirement:**
시스템은 구조화된 JSON 형식으로 로그를 출력해야 한다.

**Log Format:**
```json
{
  "timestamp": "2026-02-05T10:30:00.000Z",
  "level": "info",
  "message": "Request completed",
  "fields": {
    "url": "https://example.com",
    "status_code": 200,
    "duration_ms": 234,
    "bytes": 12560
  },
  "trace_id": "abc123",
  "span_id": "def456"
}
```

**Log Levels:**
- DEBUG: 상세 디버깅 정보
- INFO: 일반 작업 정보
- WARN: 경고 (정상 동작은 유지)
- ERROR: 오류 발생

**Acceptance Criteria:**
- [ ] JSON 형식 출력
- [ ] 로그 레벨 필터링
- [ ] 트레이스 ID 포함

---

---

## 5. Interface Requirements

### 5.1 Go SDK Interface (SRS-API-001)

**Traces To:** PRD: SO1, FR-100

```go
package crawler

// Crawler is the main interface for crawling operations
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
    OnXML(xpath string, callback XMLCallback)

    // Statistics
    Stats() *CrawlStats
}

// Builder pattern for configuration
type CrawlerBuilder interface {
    WithConfig(cfg *Config) CrawlerBuilder
    WithMiddleware(m Middleware) CrawlerBuilder
    WithPlugin(p Plugin) CrawlerBuilder
    Build() (Crawler, error)
}

// Factory function
func New(opts ...Option) (Crawler, error)
func NewBuilder() CrawlerBuilder
```

### 5.2 Python SDK Interface (SRS-API-002)

**Traces To:** PRD: FR-700

```python
# Type-safe Python interface
from typing import Protocol, runtime_checkable

@runtime_checkable
class CrawlerProtocol(Protocol):
    def crawl(self, url: str, options: Optional[CrawlOptions] = None) -> CrawlResult: ...
    def crawl_batch(self, urls: List[str], options: Optional[CrawlOptions] = None) -> Iterator[CrawlResult]: ...
    def close(self) -> None: ...

# Synchronous implementation
class CrawlerClient(CrawlerProtocol):
    ...

# Asynchronous implementation
class AsyncCrawlerClient:
    async def crawl(self, url: str, options: Optional[CrawlOptions] = None) -> CrawlResult: ...
    async def crawl_batch(self, urls: List[str], options: Optional[CrawlOptions] = None) -> AsyncIterator[CrawlResult]: ...
    async def close(self) -> None: ...
```

### 5.3 gRPC Interface (SRS-API-003)

**Traces To:** PRD: FR-701

```protobuf
// Full service definition
service CrawlerService {
    rpc Crawl(CrawlRequest) returns (CrawlResponse);
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);
    rpc StartJob(StartJobRequest) returns (JobResponse);
    rpc GetJobStatus(JobStatusRequest) returns (JobResponse);
    rpc StopJob(StopJobRequest) returns (JobResponse);
    rpc StreamResults(StreamRequest) returns (stream CrawlResponse);
    rpc GetStats(StatsRequest) returns (StatsResponse);
}
```

### 5.4 Configuration Interface (SRS-API-004)

**Traces To:** PRD: Technical Architecture

```yaml
# crawler.yaml schema
crawler:
  name: string                    # 크롤러 이름
  max_concurrency: int            # 최대 동시 요청 (default: 100)
  max_depth: int                  # 최대 크롤링 깊이 (default: 3)
  requests_per_second: float      # 초당 요청 수 (default: 10.0)
  allowed_domains: [string]       # 허용 도메인
  respect_robots_txt: bool        # robots.txt 준수 (default: true)
  user_agent: string              # User-Agent

http:
  timeout: duration               # 요청 타임아웃 (default: 30s)
  max_retries: int                # 최대 재시도 (default: 3)
  follow_redirects: bool          # 리다이렉트 따라가기 (default: true)
  max_redirects: int              # 최대 리다이렉트 (default: 10)

browser:
  enabled: bool                   # 브라우저 활성화 (default: false)
  headless: bool                  # Headless 모드 (default: true)
  pool_size: int                  # 풀 크기 (default: 5)
  page_timeout: duration          # 페이지 타임아웃 (default: 30s)

storage:
  type: string                    # memory, redis, postgres
  redis_url: string               # Redis URL
  postgres_url: string            # PostgreSQL URL

middleware:
  retry:
    enabled: bool
    max_retries: int
    base_delay: duration
  rate_limit:
    enabled: bool
    requests_per_second: float
    per_domain: bool
  robots:
    enabled: bool
    cache_ttl: duration

logging:
  level: string                   # debug, info, warn, error
  format: string                  # json, text
  output: string                  # stdout, file path

server:
  grpc_port: int                  # gRPC 포트 (default: 50051)
  http_port: int                  # HTTP 포트 (default: 8080)
  metrics_path: string            # 메트릭 경로 (default: /metrics)
```

---

## 6. Data Requirements

### 6.1 Data Models (SRS-DATA)

#### SRS-DATA-001: Request Model

```go
type Request struct {
    // Identity
    ID        string    `json:"id"`
    URL       string    `json:"url"`
    Method    string    `json:"method"`

    // Headers
    Headers   map[string]string `json:"headers"`

    // Body
    Body      []byte    `json:"body,omitempty"`

    // Crawl context
    Depth     int       `json:"depth"`
    Priority  int       `json:"priority"`
    ParentURL string    `json:"parent_url,omitempty"`

    // Options
    Options   *RequestOptions `json:"options,omitempty"`

    // Metadata
    Metadata  map[string]any `json:"metadata,omitempty"`

    // Timestamps
    CreatedAt time.Time `json:"created_at"`
}
```

#### SRS-DATA-002: Response Model

```go
type Response struct {
    // Request reference
    Request   *Request  `json:"request"`

    // HTTP response
    StatusCode  int               `json:"status_code"`
    Headers     map[string]string `json:"headers"`
    Body        []byte            `json:"body"`
    ContentType string            `json:"content_type"`

    // Performance
    FetchTime   time.Duration `json:"fetch_time"`

    // Final state
    FinalURL    string `json:"final_url"`
    Cached      bool   `json:"cached"`

    // Timestamps
    ReceivedAt time.Time `json:"received_at"`
}
```

#### SRS-DATA-003: Crawled Data Model

```go
type CrawledData struct {
    // Identity
    ID          string `json:"id"`
    URL         string `json:"url"`
    URLHash     string `json:"url_hash"`

    // Response
    StatusCode  int    `json:"status_code"`
    ContentType string `json:"content_type"`
    Content     []byte `json:"content,omitempty"`

    // Extracted
    ExtractedData map[string]any `json:"extracted_data,omitempty"`

    // Metadata
    Domain    string         `json:"domain"`
    Depth     int            `json:"depth"`
    Metadata  map[string]any `json:"metadata,omitempty"`

    // Timestamps
    CrawledAt time.Time `json:"crawled_at"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### SRS-DATA-004: Job Model

```go
type Job struct {
    // Identity
    ID        string    `json:"id"`
    Name      string    `json:"name"`

    // Configuration
    SeedURLs  []string  `json:"seed_urls"`
    Config    *Config   `json:"config"`

    // Status
    Status    JobStatus `json:"status"`
    Progress  *JobProgress `json:"progress"`

    // Error
    Error     string    `json:"error,omitempty"`

    // Timestamps
    CreatedAt   time.Time  `json:"created_at"`
    StartedAt   *time.Time `json:"started_at,omitempty"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type JobStatus string
const (
    JobStatusPending   JobStatus = "pending"
    JobStatusRunning   JobStatus = "running"
    JobStatusPaused    JobStatus = "paused"
    JobStatusCompleted JobStatus = "completed"
    JobStatusFailed    JobStatus = "failed"
    JobStatusCancelled JobStatus = "cancelled"
)

type JobProgress struct {
    TotalURLs     int64 `json:"total_urls"`
    CrawledURLs   int64 `json:"crawled_urls"`
    SuccessURLs   int64 `json:"success_urls"`
    FailedURLs    int64 `json:"failed_urls"`
    PendingURLs   int64 `json:"pending_urls"`
}
```

### 6.2 Database Schema

#### PostgreSQL Schema

```sql
-- Crawled pages table
CREATE TABLE crawled_pages (
    id          BIGSERIAL PRIMARY KEY,
    url         TEXT UNIQUE NOT NULL,
    url_hash    VARCHAR(64) NOT NULL,
    domain      VARCHAR(255) NOT NULL,
    status_code INTEGER,
    content_type VARCHAR(255),
    content     BYTEA,
    extracted   JSONB DEFAULT '{}',
    metadata    JSONB DEFAULT '{}',
    depth       INTEGER DEFAULT 0,
    crawled_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_crawled_pages_url_hash ON crawled_pages(url_hash);
CREATE INDEX idx_crawled_pages_domain ON crawled_pages(domain);
CREATE INDEX idx_crawled_pages_crawled_at ON crawled_pages(crawled_at DESC);
CREATE INDEX idx_crawled_pages_status ON crawled_pages(status_code);
CREATE INDEX idx_crawled_pages_extracted ON crawled_pages USING GIN(extracted);

-- Jobs table
CREATE TABLE crawl_jobs (
    id          VARCHAR(36) PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    seed_urls   TEXT[] NOT NULL,
    config      JSONB NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress    JSONB DEFAULT '{}',
    error       TEXT,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at  TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_crawl_jobs_status ON crawl_jobs(status);
CREATE INDEX idx_crawl_jobs_created_at ON crawl_jobs(created_at DESC);

-- URL frontier table (for persistence)
CREATE TABLE url_frontier (
    id          BIGSERIAL PRIMARY KEY,
    job_id      VARCHAR(36) NOT NULL REFERENCES crawl_jobs(id),
    url         TEXT NOT NULL,
    url_hash    VARCHAR(64) NOT NULL,
    priority    INTEGER DEFAULT 5,
    depth       INTEGER DEFAULT 0,
    status      VARCHAR(20) DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(job_id, url_hash)
);

CREATE INDEX idx_url_frontier_job_status ON url_frontier(job_id, status);
CREATE INDEX idx_url_frontier_scheduled ON url_frontier(scheduled_at) WHERE status = 'pending';
```

---

## 7. System Constraints

### 7.1 Technical Constraints

| Constraint | Description |
|------------|-------------|
| **Language** | Core engine: Go 1.21+, Bindings: Python 3.9+ |
| **Platform** | Linux (primary), macOS, Windows (limited) |
| **Memory** | Minimum 512MB, Recommended 4GB+ |
| **Network** | IPv4/IPv6 지원, HTTP/1.1, HTTP/2 |
| **Dependencies** | Chrome/Chromium (JS rendering 시) |

### 7.2 Regulatory Constraints

| Constraint | Description |
|------------|-------------|
| **robots.txt** | 기본적으로 준수, 비활성화 옵션 제공 |
| **Rate Limiting** | 서버 과부하 방지 필수 |
| **User-Agent** | 식별 가능한 User-Agent 권장 |
| **Personal Data** | GDPR/개인정보보호법 준수 |

### 7.3 Operational Constraints

| Constraint | Description |
|------------|-------------|
| **Logging** | 30일 로그 보관 |
| **Metrics** | 7일 메트릭 보관 (기본) |
| **Backup** | 일일 설정 백업 권장 |

---

## 8. Traceability Matrix

### 8.1 PRD → SRS Forward Traceability

| PRD ID | PRD Description | SRS IDs |
|--------|-----------------|---------|
| **FR-101** | HTTP/1.1 및 HTTP/2 요청 지원 | SRS-CORE-001, SRS-CORE-002 |
| **FR-102** | 동시 요청 수 제한 | SRS-CORE-003 |
| **FR-103** | 요청별 타임아웃 설정 | SRS-CORE-002 |
| **FR-104** | 자동 리다이렉트 처리 | SRS-CORE-002 |
| **FR-105** | 커스텀 헤더 설정 | SRS-CORE-002 |
| **FR-106** | 쿠키 관리 | SRS-CORE-004 |
| **FR-107** | 프록시 지원 | SRS-CORE-005 |
| **FR-108** | TLS 인증서 검증 | SRS-SEC-001 |
| **FR-201** | chromedp 기반 JavaScript 렌더링 | SRS-JS-001, SRS-JS-002 |
| **FR-202** | 페이지 로드 대기 조건 | SRS-JS-002 |
| **FR-203** | Headless/Headful 모드 전환 | SRS-JS-001 |
| **FR-204** | 브라우저 풀 관리 | SRS-JS-001 |
| **FR-205** | 스크린샷 캡처 | SRS-JS-004 |
| **FR-206** | 리소스 차단 | SRS-JS-003 |
| **FR-301** | URL Frontier 우선순위 큐 | SRS-URL-001 |
| **FR-302** | URL 정규화 | SRS-URL-002 |
| **FR-303** | URL 중복 제거 | SRS-URL-003 |
| **FR-304** | 도메인별 큐 관리 | SRS-URL-001 |
| **FR-305** | Recrawl 스케줄링 | SRS-URL-001 |
| **FR-306** | URL 필터링 규칙 | SRS-URL-004 |
| **FR-401** | CSS Selector 기반 추출 | SRS-EXT-001 |
| **FR-402** | XPath 기반 추출 | SRS-EXT-002 |
| **FR-403** | 정규표현식 지원 | SRS-EXT-004 |
| **FR-404** | JSON 파싱 | SRS-EXT-003 |
| **FR-405** | XML/RSS 파싱 | SRS-EXT-002 |
| **FR-406** | 자동 인코딩 감지 | SRS-EXT-005 |
| **FR-501** | Retry with Exponential Backoff | SRS-MW-002 |
| **FR-502** | Rate Limiting | SRS-MW-003 |
| **FR-503** | Proxy Rotation | SRS-MW-005 |
| **FR-504** | User-Agent Rotation | SRS-MW-006 |
| **FR-505** | robots.txt 자동 준수 | SRS-MW-004 |
| **FR-506** | 인증 | SRS-MW-007 |
| **FR-507** | 캐싱 미들웨어 | SRS-STOR-003 |
| **FR-601** | PostgreSQL 저장소 | SRS-STOR-002 |
| **FR-602** | MongoDB 저장소 | SRS-STOR-001 |
| **FR-603** | Redis 캐시/큐 | SRS-STOR-003 |
| **FR-604** | JSON/CSV 내보내기 | SRS-STOR-004 |
| **FR-605** | S3 내보내기 | SRS-STOR-004 |
| **FR-701** | gRPC Python 클라이언트 | SRS-PY-001 |
| **FR-702** | 동기/비동기 API | SRS-PY-002, SRS-PY-003 |
| **FR-703** | Type hints | SRS-PY-004 |
| **FR-704** | Context Manager | SRS-PY-002, SRS-PY-003 |
| **FR-705** | Pydantic 통합 | SRS-PY-004 |
| **FR-801** | crawler init | SRS-CLI-002 |
| **FR-802** | crawler run | SRS-CLI-001 |
| **FR-803** | crawler crawl | SRS-CLI-003 |
| **FR-804** | crawler server | SRS-CLI-001 |
| **FR-805** | crawler test | SRS-CLI-001 |
| **FR-806** | crawler benchmark | SRS-CLI-001 |
| **NFR-101** | 처리량 5,000 req/s | SRS-PERF-001 |
| **NFR-102** | 응답 시간 < 500ms | SRS-PERF-003 |
| **NFR-103** | 메모리 < 100MB | SRS-PERF-002 |
| **NFR-104** | CPU 사용률 < 70% | SRS-PERF-001 |
| **NFR-105** | 동시 연결 10,000+ | SRS-CORE-003 |
| **NFR-301** | 가용성 99.9% | SRS-REL-002 |
| **NFR-302** | 성공률 > 95% | SRS-REL-001 |
| **NFR-303** | 장애 복구 < 5분 | SRS-REL-001 |
| **NFR-401** | TLS 1.2+ | SRS-SEC-001 |
| **NFR-402** | 인증 정보 보호 | SRS-SEC-002 |
| **NFR-403** | 로그 민감 정보 마스킹 | SRS-SEC-003 |
| **NFR-501** | robots.txt 준수 | SRS-MW-004 |
| **NFR-502** | Crawl-delay 준수 | SRS-MW-003, SRS-MW-004 |
| **NFR-603** | 구조화된 로깅 | SRS-OBS-002 |
| **NFR-604** | Prometheus 메트릭 | SRS-OBS-001 |

### 8.2 SRS → PRD Backward Traceability

| SRS ID | SRS Title | PRD IDs |
|--------|-----------|---------|
| SRS-CORE-001 | HTTP Client Initialization | FR-101, FR-102, FR-103 |
| SRS-CORE-002 | Request Execution | FR-101, FR-104, FR-105 |
| SRS-CORE-003 | Concurrent Request Management | FR-102, NFR-101, NFR-105 |
| SRS-CORE-004 | Cookie Management | FR-106 |
| SRS-CORE-005 | Proxy Support | FR-107 |
| SRS-JS-001 | Browser Pool Management | FR-201, FR-204 |
| SRS-JS-002 | Page Rendering | FR-201, FR-202 |
| SRS-JS-003 | Resource Blocking | FR-206 |
| SRS-JS-004 | Screenshot Capture | FR-205 |
| SRS-URL-001 | URL Frontier | FR-301, FR-304, FR-305 |
| SRS-URL-002 | URL Canonicalization | FR-302 |
| SRS-URL-003 | URL Deduplication | FR-303 |
| SRS-URL-004 | URL Filtering | FR-306 |
| SRS-EXT-001 | CSS Selector Extraction | FR-401 |
| SRS-EXT-002 | XPath Extraction | FR-402, FR-405 |
| SRS-EXT-003 | JSON Extraction | FR-404 |
| SRS-EXT-004 | Regex Extraction | FR-403 |
| SRS-EXT-005 | Encoding Detection | FR-406 |
| SRS-MW-001 | Middleware Chain | FR-501, FR-502, FR-505 |
| SRS-MW-002 | Retry Middleware | FR-501 |
| SRS-MW-003 | Rate Limit Middleware | FR-502, NFR-501, NFR-502 |
| SRS-MW-004 | Robots.txt Middleware | FR-505, NFR-501 |
| SRS-MW-005 | Proxy Rotation | FR-503 |
| SRS-MW-006 | User-Agent Rotation | FR-504 |
| SRS-MW-007 | Authentication | FR-506 |
| SRS-STOR-001 | Storage Plugin Interface | FR-601, FR-602, FR-603 |
| SRS-STOR-002 | PostgreSQL Plugin | FR-601 |
| SRS-STOR-003 | Redis Integration | FR-603, FR-507 |
| SRS-STOR-004 | File Export | FR-604, FR-605 |
| SRS-PY-001 | gRPC Service | FR-701 |
| SRS-PY-002 | Sync Python Client | FR-701, FR-702, FR-704 |
| SRS-PY-003 | Async Python Client | FR-702, FR-704 |
| SRS-PY-004 | Type Hints | FR-703, FR-705 |
| SRS-CLI-001 | CLI Framework | FR-801-806 |
| SRS-CLI-002 | Init Command | FR-801 |
| SRS-CLI-003 | Crawl Command | FR-803 |
| SRS-PERF-001 | Throughput | NFR-101, NFR-104 |
| SRS-PERF-002 | Memory Usage | NFR-103 |
| SRS-PERF-003 | Latency | NFR-102 |
| SRS-SEC-001 | TLS Configuration | NFR-401, FR-108 |
| SRS-SEC-002 | Credential Protection | NFR-402 |
| SRS-SEC-003 | Log Sanitization | NFR-403 |
| SRS-REL-001 | Error Recovery | NFR-302, NFR-303 |
| SRS-REL-002 | Graceful Shutdown | NFR-301 |
| SRS-OBS-001 | Prometheus Metrics | NFR-604 |
| SRS-OBS-002 | Structured Logging | NFR-603 |

### 8.3 Coverage Summary

| PRD Category | Total Requirements | Covered in SRS | Coverage |
|--------------|-------------------|----------------|----------|
| FR-100 (Core Engine) | 8 | 8 | 100% |
| FR-200 (JS Rendering) | 6 | 6 | 100% |
| FR-300 (URL Management) | 6 | 6 | 100% |
| FR-400 (Data Extraction) | 6 | 6 | 100% |
| FR-500 (Middleware) | 7 | 7 | 100% |
| FR-600 (Storage) | 5 | 5 | 100% |
| FR-700 (Python Bindings) | 5 | 5 | 100% |
| FR-800 (CLI) | 6 | 6 | 100% |
| NFR-100 (Performance) | 5 | 5 | 100% |
| NFR-200 (Scalability) | 3 | 3 | 100% |
| NFR-300 (Reliability) | 4 | 4 | 100% |
| NFR-400 (Security) | 4 | 4 | 100% |
| NFR-500 (Compliance) | 4 | 4 | 100% |
| NFR-600 (Maintainability) | 4 | 4 | 100% |
| **Total** | **73** | **73** | **100%** |

---

## 9. Appendix

### 9.1 Error Codes

| Code | Name | Description |
|------|------|-------------|
| E1001 | NETWORK_TIMEOUT | 네트워크 타임아웃 |
| E1002 | DNS_FAILURE | DNS 조회 실패 |
| E1003 | CONNECTION_REFUSED | 연결 거부됨 |
| E1004 | SSL_ERROR | SSL/TLS 오류 |
| E2001 | RATE_LIMITED | 요청 빈도 제한 |
| E2002 | BLOCKED | 요청 차단됨 |
| E2003 | ROBOTS_DISALLOWED | robots.txt에 의해 차단 |
| E3001 | PARSE_ERROR | 파싱 오류 |
| E3002 | ENCODING_ERROR | 인코딩 오류 |
| E4001 | STORAGE_ERROR | 저장소 오류 |
| E4002 | QUEUE_FULL | 큐 용량 초과 |

### 9.2 Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0.0 | 2026-02-05 | Development Team | Initial SRS from PRD |

### 9.3 Document Approval

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Tech Lead | | | |
| Software Architect | | | |
| QA Lead | | | |

---

*This SRS document maintains full bidirectional traceability with the PRD to ensure all business requirements are properly addressed in the technical specification.*
