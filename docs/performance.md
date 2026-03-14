# Performance Baseline

Measured on Apple M4 Max (arm64, 14 cores), Go 1.24, `benchtime=5s`.

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
