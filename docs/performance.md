# Performance Baseline

Measured on Apple M4 Max (arm64, 14 cores), Go 1.24, `benchtime=5s`.

## HTTP Client Throughput (`pkg/client`)

| Benchmark | Before | After (v0.2) | Change |
|-----------|--------|-------------|--------|
| `BenchmarkDo` (serial) | 35,978 ns/op, 88 allocs | 34,702 ns/op, 88 allocs | −4% ns/op |
| `BenchmarkDo_Parallel` (14 goroutines) | 11,236 ns/op, 8,558 B/op | 11,630 ns/op, 8,136 B/op | −5% B/op |
| Parallel throughput | ~89,000 req/s | ~89,000 req/s | — |

**Optimizations applied (PR #66)**:
- Response body buffer pool (`sync.Pool[*bytes.Buffer]`) — reduces GC pressure for large responses; `Content-Length` pre-allocation avoids buffer growth
- `MaxIdleConnsPerHost` 10 → 20 — prevents connection starvation when worker count > 10

> Serial and parallel micro-benchmarks show modest improvement because the test payload is only 23 bytes. The buffer pool benefit is more pronounced for real-world pages (4–100 KB range) where GC pressure reduction matters.



## Engine Throughput (`pkg/crawler`)

| Benchmark | ops/s | URLs/op | ns/op | Allocs/op |
|-----------|-------|---------|-------|-----------|
| `BenchmarkEngine_Throughput` (10 workers, batch=50) | ~180/s batches | 50 | 5,578,821 | 5,059 |
| **Effective req/s** | **~8,991 req/s** | | | |

Effective throughput: `50 urls/op ÷ 5.578ms/op ≈ 8,991 req/s` against a local httptest server.

**Target**: 5,000 req/s (PRD NFR-101) ✅ **Exceeded on local loopback**

> Note: Real-world throughput depends on network latency, server response time, and concurrency settings. Local benchmarks eliminate network latency and represent the ceiling of engine-side processing overhead.

Run the benchmark:

```bash
go test -bench=BenchmarkEngine_Throughput -benchmem -benchtime=5s ./pkg/crawler/
```

Scaling by worker count:

```bash
go test -bench=BenchmarkEngine_WorkerScaling -benchmem -benchtime=3s ./pkg/crawler/
```

## Middleware Chain Overhead (`pkg/middleware`)

| Layers | ns/op | Allocs/op |
|--------|-------|-----------|
| 0 (pass-through) | 22.11 | 2 |
| 1 | 36.13 | 3 |
| 3 | 63.01 | 5 |
| 5 | 89.96 | 7 |
| 10 | 162.6 | 12 |

Each middleware layer adds ~14–16 ns overhead.

Run the benchmark:

```bash
go test -bench=BenchmarkChain -benchmem -benchtime=3s ./pkg/middleware/
```

## Frontier Throughput (`pkg/frontier`)

| Benchmark | ns/op | Allocs/op | ops/s |
|-----------|-------|-----------|-------|
| `BenchmarkFrontier_Add` | 468.8 | 5 | ~2.1M |
| `BenchmarkFrontier_AddNext` (round-trip) | 534.4 | 5 | ~1.9M |

The in-memory frontier supports ~1.9M URL enqueue+dequeue cycles per second, well above crawler engine requirements.

Run the benchmark:

```bash
go test -bench=BenchmarkFrontier -benchmem -benchtime=3s ./pkg/frontier/
```

## CLI Benchmark Command

Use `crawler benchmark` to measure throughput and latency against any target URL:

```bash
# 30-second load test with 20 concurrent workers
crawler benchmark https://example.com --concurrency 20 --duration 30s

# Fixed request count
crawler benchmark https://example.com --concurrency 10 --requests 1000
```

Example output:

```json
{
  "url": "https://example.com",
  "workers": 20,
  "total_requests_sent": 600,
  "successful": 598,
  "failed": 2,
  "wall_time": "30.123s",
  "throughput_req_per_sec": 19.85,
  "latency_p50_ms": "48.20 ms",
  "latency_p95_ms": "215.40 ms",
  "latency_p99_ms": "488.10 ms",
  "mem_alloc_mb": 12.34
}
```

## Performance Targets (PRD NFR-101 – NFR-105)

| Metric | Target | Baseline (local) | Status |
|--------|--------|-----------------|--------|
| Throughput | 5,000 req/s | ~8,991 req/s | ✅ |
| Memory baseline | < 100 MB | TBD (real workload) | — |
| Latency P50 | < 200 ms | N/A (network-dependent) | — |
| Latency P99 | < 1 s | N/A (network-dependent) | — |
| CPU utilization | < 70% | N/A (profiling required) | — |

> Latency and memory targets require measurement against a real target with network latency. Use `crawler benchmark` against a staging environment to validate these.

---

## Performance Tuning Guide

This section provides actionable recommendations for configuring the crawler to
meet or exceed the PRD performance targets in real-world deployments.

### Quick-Start: High-Throughput Configuration

The snippet below is tuned for broad, multi-domain crawling at 5,000+ req/s
when network round-trip latency is 50–200 ms.

```go
import (
    "github.com/kcenon/web_crawler/pkg/client"
    "github.com/kcenon/web_crawler/pkg/crawler"
    "github.com/kcenon/web_crawler/pkg/frontier"
    "time"
)

// Estimated total URLs for the crawl run (reduces GC pressure).
const estimatedURLs = 100_000

f := frontier.NewMemoryFrontier(frontier.Config{
    CrawlDelay:      500 * time.Millisecond, // be polite; set 0 for benchmarking
    InitialCapacity: estimatedURLs,
})

c, err := crawler.NewBuilder().
    WithWorkerCount(50).
    WithConfig(crawler.Config{
        MaxDepth: 5,
        Concurrency: crawler.ConcurrencyConfig{
            GlobalMaxConcurrency:    200,
            PerDomainMaxConcurrency: 10,
        },
        Client: client.Config{
            Timeout: 20 * time.Second,
            Transport: client.TransportConfig{
                MaxIdleConns:          500,
                MaxIdleConnsPerHost:   50,
                MaxConnsPerHost:       50,
                IdleConnTimeout:       90 * time.Second,
                DialTimeout:           5 * time.Second,
                TLSHandshakeTimeout:   5 * time.Second,
                ResponseHeaderTimeout: 10 * time.Second,
            },
        },
    }).
    Build()
```

---

### Worker Count Tuning

Workers are Go goroutines — they are cheap. The optimal count depends on
average request latency, not CPU count.

**Formula**:

```
workers ≥ target_req_per_sec × avg_latency_seconds × safety_factor(1.5–2)
```

| Target req/s | Avg latency | Min workers | Recommended |
|--------------|-------------|-------------|-------------|
| 1,000 | 100 ms | 100 | 150–200 |
| 5,000 | 100 ms | 500 | 600–800 |
| 5,000 | 200 ms | 1,000 | 1,200–1,500 |

**Rules of thumb**:
- Start with `WorkerCount = 2 × target_req_per_sec × avg_latency_s`
- If CPU is the bottleneck (>80 % utilisation), lower the worker count
- If workers are idle (low CPU, low throughput), raise the worker count
- Never set `GlobalMaxConcurrency` below `WorkerCount`

---

### HTTP Transport Settings

The `client.TransportConfig` controls connection pooling, which is the most
common source of throughput regression.

| Field | Default | High-Throughput Value | Notes |
|-------|---------|----------------------|-------|
| `MaxIdleConns` | 100 | 500–1,000 | Total idle connections across all hosts |
| `MaxIdleConnsPerHost` | 20 | 50–100 | Must be ≥ concurrent workers per domain |
| `MaxConnsPerHost` | 0 | 50–100 | 0 = unlimited; set equal to `MaxIdleConnsPerHost` |
| `IdleConnTimeout` | 90 s | 90 s | Keep long to avoid TCP re-establishment |
| `DialTimeout` | 10 s | 5 s | Fail fast on unresponsive hosts |
| `ResponseHeaderTimeout` | 15 s | 10 s | Detect hung servers early |
| `TLSHandshakeTimeout` | 10 s | 5 s | Fail fast on TLS issues |

**Warning**: Setting `MaxIdleConnsPerHost` below `WorkerCount` causes
connection starvation and forces new TCP dials, which adds 20–200 ms per
affected request. The default was raised from Go's standard 2 to 20 in
PR #67 specifically to prevent this.

---

### Concurrency Limits

```go
Concurrency: crawler.ConcurrencyConfig{
    GlobalMaxConcurrency:    200,   // total in-flight requests
    PerDomainMaxConcurrency: 10,    // per target domain
    // DisablePerDomainLimits: true, // only for single-domain crawls
}
```

- Set `GlobalMaxConcurrency` ≈ `WorkerCount × 1.5` to provide a buffer
- Set `PerDomainMaxConcurrency` to match the server's connection capacity
  (start at 5–10; check server logs for 429/503 responses)
- For single-domain, high-performance crawls, set `DisablePerDomainLimits: true`
  and use only `GlobalMaxConcurrency`

---

### Frontier Configuration

```go
frontier.Config{
    CrawlDelay:      500 * time.Millisecond, // per-domain politeness delay
    InitialCapacity: 100_000,               // pre-allocate for expected URL count
}
```

**`InitialCapacity`**: Set this to an estimate of the total URLs to be crawled.
When the dedup map grows beyond its capacity, Go allocates a new backing array
2× as large and copies all entries. For a 100 k-URL crawl with zero initial
capacity, this happens ~17 times. With `InitialCapacity: 100_000` it happens
zero times.

| Total URLs | Rehash cycles (no hint) | With InitialCapacity |
|------------|------------------------|----------------------|
| 1,000 | ~10 | 0 |
| 100,000 | ~17 | 0 |
| 1,000,000 | ~20 | 0 |

**`CrawlDelay`**: Set to `0` only for benchmarking or crawling your own
infrastructure. For public sites, `500 ms–2 s` is a common polite default.

---

### Memory Optimisation

**Response body buffer pool** (built-in): `pkg/client` reuses `bytes.Buffer`
objects via `sync.Pool`, avoiding per-request heap allocation for response
bodies up to 1 MiB. No user configuration needed.

**GC tuning**: For sustained high-throughput crawls, consider raising `GOGC`:

```bash
GOGC=200 crawler run --config config.yaml
```

A higher `GOGC` reduces GC frequency by allowing the heap to grow to 2× its
live size before triggering a collection. This trades memory for lower GC
pause overhead. Start with `GOGC=200` and monitor heap size with `go tool pprof`.

**Estimated memory** per 100 k deduplicated URLs:
- Dedup map entries: ~100 k × ~50 bytes (avg URL) ≈ ~5 MB
- Priority queue entries: ~100 k × ~80 bytes (`URLEntry`) ≈ ~8 MB
- HTTP keep-alive connections: 100 × ~32 KB ≈ ~3 MB
- **Total frontier overhead**: ~16 MB for 100 k URLs

---

### Profiling Guide

Use `go tool pprof` to identify bottlenecks in a running crawl:

```bash
# 1. Enable pprof endpoint (add to your server setup)
import _ "net/http/pprof"
go http.ListenAndServe(":6060", nil)

# 2. Capture a 30-second CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# 3. Capture a heap snapshot
go tool pprof http://localhost:6060/debug/pprof/heap

# 4. In the pprof shell:
(pprof) top20        # top 20 functions by CPU/memory
(pprof) web          # open flame graph in browser (requires graphviz)
(pprof) list <func>  # annotated source for a specific function
```

**Common bottleneck patterns**:

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| High `runtime.mallocgc` | Too many allocations | Raise `GOGC`, check hot paths |
| High `net/http.(*Transport).roundTrip` | Connection pool exhaustion | Increase `MaxIdleConnsPerHost` |
| High `sync.(*Mutex).Lock` in frontier | Lock contention | Ensure frontier is not the bottleneck (it supports 1.9M ops/s) |
| High `net.(*netFD).Read` | Waiting on network I/O | Normal for I/O-bound crawls; add more workers |

---

### Interpreting CI Benchmark Comments

After every PR, the `Benchmarks` CI workflow posts a `benchstat` comparison table:

```
goos: linux
goarch: amd64
pkg: github.com/kcenon/web_crawler/pkg/frontier

              │  baseline  │              pr              │
              │   sec/op   │   sec/op     vs base         │
Frontier/Add    468.8n ± 2%  421.3n ± 1%  -10.12% (p=0.008 n=3)
Frontier/Next   534.4n ± 1%  534.1n ± 2%   ~      (p=0.700 n=3)
```

| Symbol | Meaning |
|--------|---------|
| `~` | No statistically significant change (p > 0.05) |
| `-N%` | Improvement (lower is better for `sec/op`) |
| `+N%` | Regression (higher is worse for `sec/op`) |
| `(p=0.008 n=3)` | p-value and number of benchmark runs |

A change with `p > 0.05` is not statistically significant regardless of
the percentage shown. Focus on regressions with `p < 0.05` and `+N% > 5%`.
