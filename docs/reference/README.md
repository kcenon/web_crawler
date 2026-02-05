# Web Crawler SDK Reference Documentation

> **Created**: 2026-02-05
> **Updated**: 2026-02-05
> **Purpose**: Comprehensive reference materials for building a production-ready web crawler SDK
> **Strategy**: Go Core + Python Bindings (Strategy C)

## SDK Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        Web Crawler SDK Architecture                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌──────────────────────────────────────────────────────────────────┐  │
│   │                      Python SDK (User-Facing)                     │  │
│   │  • Pythonic API  • Type hints  • Async support  • Rich ecosystem │  │
│   └────────────────────────────┬─────────────────────────────────────┘  │
│                                │ CGO / gRPC                              │
│   ┌────────────────────────────▼─────────────────────────────────────┐  │
│   │                        Go Core Engine                             │  │
│   │  • HTTP Client  • Scheduler  • URL Frontier  • Rate Limiter      │  │
│   │  • Parser Engine  • Middleware Chain  • Plugin System            │  │
│   └──────────────────────────────────────────────────────────────────┘  │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Document Index

### Core Documentation (01-13)

| # | Document | Description |
|---|----------|-------------|
| 01 | [Legal Considerations](./01-legal-considerations.md) | Copyright, CFAA, TOS, Korean law, GDPR compliance |
| 02 | [Technical Stack](./02-technical-stack.md) | **Go core + Python bindings**, Colly, chromedp, framework selection |
| 03 | [Policy Guidelines](./03-policy-guidelines.md) | robots.txt, crawl-delay, ethics, User-Agent policy |
| 04 | [Stability & Continuity](./04-stability-continuity.md) | Distributed architecture, error handling, anti-bot evasion |
| 05 | [Data Storage](./05-data-storage.md) | PostgreSQL, MongoDB, Redis, data pipelines |
| 06 | [Additional Considerations](./06-additional-considerations.md) | Security, testing, cost optimization, deployment |
| 07 | [URL Frontier & Scheduling](./07-url-frontier-scheduling.md) | URL prioritization, recrawl scheduling, politeness |
| 08 | [SPA & JavaScript Rendering](./08-spa-javascript-rendering.md) | SPA crawling, API discovery, headless browsers |
| 09 | [Data Quality](./09-data-quality.md) | Validation, cleaning, deduplication, profiling |
| 10 | [Special Content Types](./10-special-content-types.md) | PDF extraction, OCR, video metadata, Office docs |
| 11 | [Internationalization](./11-internationalization.md) | Encoding detection, multilingual, CJK, RTL languages |
| 12 | [Site-Specific Strategies](./12-site-specific-strategies.md) | E-commerce, news, job boards, real estate patterns |
| 13 | [Testing Environment](./13-testing-environment.md) | Mock servers, unit/integration/E2E testing, CI/CD |

### SDK Development (14-16)

| # | Document | Description |
|---|----------|-------------|
| 14 | [SDK Architecture](./14-sdk-architecture.md) | Go core design, interfaces, plugin system, middleware |
| 15 | [Go-Python Binding](./15-go-python-binding.md) | CGO, gRPC, PyO3 integration strategies |
| 16 | [Developer Experience](./16-developer-experience.md) | CLI tools, debugging, project templates, IDE support |

---

## Quick Reference

### Technology Stack (Strategy C: Go Core + Python Bindings)

| Layer | Technology | Purpose |
|-------|------------|---------|
| **Core Engine** | Go | High-performance HTTP, scheduling, concurrency |
| **Browser Automation** | chromedp (Go) | JavaScript rendering, SPA support |
| **Python Bindings** | CGO + gRPC | Pythonic API for data scientists |
| **Data Storage** | PostgreSQL + Redis | Persistence and caching |
| **Message Queue** | Kafka / NATS | Distributed task distribution |

### Scale-Based Architecture

| Scale | Architecture | Performance Target |
|-------|--------------|-------------------|
| **Small** (< 100K pages/day) | Single Go binary | 500 req/s |
| **Medium** (100K - 10M pages/day) | Go + Redis + PostgreSQL | 5,000 req/s |
| **Large** (> 10M pages/day) | Distributed Go + Kafka + K8s | 50,000+ req/s |

### Why Go Core + Python Bindings?

| Benefit | Description |
|---------|-------------|
| **5x Performance** | Go outperforms Python by 5x+ in HTTP operations |
| **Single Binary** | No dependency hell, easy container deployment |
| **True Concurrency** | Goroutines handle 10,000+ concurrent requests |
| **Memory Efficiency** | 1/5 memory usage compared to Python |
| **Python Accessibility** | Data scientists use familiar Python API |
| **Cost Reduction** | Same throughput with 80% fewer servers |

### Legal Compliance Checklist

```
□ Review target site's robots.txt
□ Check Terms of Service
□ Set identifiable User-Agent
□ Implement rate limiting
□ Avoid personal data unless necessary
□ Document compliance efforts
```

### Key Metrics to Monitor

| Metric | Target | Go Advantage |
|--------|--------|--------------|
| Success Rate | > 95% | Stable goroutine management |
| Avg Response Time | < 500ms | 5x faster than Python |
| Error Rate | < 5% | Better error handling |
| Memory Usage | < 50% | Efficient memory model |
| Concurrent Requests | 10,000+ | Native goroutine support |

---

## Reading Order

### For SDK Developers
1. **14-sdk-architecture.md** - Understand Go core design
2. **15-go-python-binding.md** - Learn binding strategies
3. **16-developer-experience.md** - Build great DX
4. **02-technical-stack.md** - Technology deep dive

### For SDK Users (Python)
1. **01-legal-considerations.md** - Understand legal boundaries
2. **03-policy-guidelines.md** - Learn ethical crawling
3. **02-technical-stack.md** - Understand the stack
4. **05-data-storage.md** - Design data architecture

### For Scaling & Operations
1. **04-stability-continuity.md** - Distributed architecture
2. **07-url-frontier-scheduling.md** - URL management at scale
3. **06-additional-considerations.md** - Cost optimization

---

## Related Resources

### Go Ecosystem
- [Colly Documentation](http://go-colly.org/docs/)
- [chromedp Documentation](https://github.com/chromedp/chromedp)
- [Go net/http Package](https://pkg.go.dev/net/http)
- [GoQuery (jQuery-like HTML parsing)](https://github.com/PuerkitoBio/goquery)

### Python Bindings
- [CGO Documentation](https://pkg.go.dev/cmd/cgo)
- [gRPC Go](https://grpc.io/docs/languages/go/)
- [gRPC Python](https://grpc.io/docs/languages/python/)
- [PyO3 (Rust-Python, for reference)](https://pyo3.rs/)

### Browser Automation
- [chromedp Examples](https://github.com/chromedp/examples)
- [Rod (Alternative Go browser driver)](https://go-rod.github.io/)

### Legal Resources
- [robots.txt Specification](https://developers.google.com/crawling/docs/robots-txt)
- [CFAA Overview](https://www.law.cornell.edu/uscode/text/18/1030)

### Korean Legal Resources
- [Korean Copyright Act (저작권법)](https://www.law.go.kr/)
- [Personal Information Protection Act (개인정보보호법)](https://www.pipc.go.kr/)

---

## Project Structure

```
web-crawler-sdk/
├── cmd/
│   └── crawler/              # CLI entry point
├── internal/
│   ├── engine/               # Core crawling engine
│   ├── scheduler/            # URL scheduling
│   ├── frontier/             # URL frontier management
│   ├── middleware/           # Middleware chain
│   └── storage/              # Storage adapters
├── pkg/
│   ├── crawler/              # Public Go API
│   ├── config/               # Configuration
│   └── plugin/               # Plugin interfaces
├── bindings/
│   ├── python/               # Python SDK (gRPC client)
│   └── grpc/                 # gRPC service definitions
├── python/
│   └── crawler_sdk/          # Python package
├── docs/
│   └── reference/            # This documentation
└── examples/
    ├── go/                   # Go usage examples
    └── python/               # Python usage examples
```

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0.0 | 2026-02-05 | Initial release (01-06) |
| 1.1.0 | 2026-02-05 | Added advanced topics (07-09) |
| 1.2.0 | 2026-02-05 | Added special content, i18n, site patterns, testing (10-13) |
| **2.0.0** | **2026-02-05** | **Strategy C: Go Core + Python Bindings (14-16)** |

---

## Contributing

This SDK follows the Go + Python hybrid architecture. When contributing:

1. **Go Code**: Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
2. **Python Code**: Follow [PEP 8](https://pep8.org/) and type hints (PEP 484)
3. **Documentation**: All docs in English, code comments in English
4. **Testing**: Go tests with `go test`, Python tests with `pytest`

---

*These documents are living references. Update as laws, technologies, and best practices evolve.*
