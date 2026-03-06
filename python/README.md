# Web Crawler Python SDK

Python client library for the Web Crawler gRPC server.

## Installation

```bash
pip install -e .
```

## Quick Start

### Synchronous Client

```python
from crawler import CrawlerClient

with CrawlerClient(host="localhost", port=50051) as client:
    # Simple crawl
    result = client.crawl("https://example.com")
    print(f"Status: {result.status_code}")
    print(f"Content length: {len(result.content)}")

    # Crawl multiple URLs
    results = client.crawl_many([
        "https://example.com",
        "https://example.org",
    ])
    for r in results:
        print(f"{r.url}: {r.status_code}")
```

### Async Client

```python
import asyncio
from crawler import AsyncCrawlerClient

async def main():
    async with AsyncCrawlerClient(host="localhost", port=50051) as client:
        result = await client.crawl("https://example.com")
        print(f"Status: {result.status_code}")

        # Stream results
        async for result in client.crawl_stream(["https://example.com"]):
            print(f"Streamed: {result.url}")

asyncio.run(main())
```

### Long-Running Crawler

```python
from crawler import CrawlerClient, CrawlConfig

with CrawlerClient() as client:
    # Start a managed crawler
    config = CrawlConfig(max_depth=3, max_pages=100)
    crawler_id = client.start(config)
    print(f"Crawler started: {crawler_id}")

    # Add more URLs
    client.add_urls(crawler_id, ["https://example.com/page2"])

    # Check stats
    stats = client.stats(crawler_id)
    print(f"Pages crawled: {stats.pages_crawled}")

    # Stop and get final stats
    final_stats = client.stop(crawler_id)
    print(f"Total crawled: {final_stats.pages_crawled}")
```

## API Reference

### CrawlerClient

| Method | Description |
|--------|-------------|
| `crawl(url, **kwargs)` | Crawl a single URL |
| `crawl_many(urls, **kwargs)` | Crawl multiple URLs |
| `start(config, crawler_id="")` | Start a managed crawler |
| `stop(crawler_id)` | Stop a crawler, get final stats |
| `add_urls(crawler_id, urls)` | Add URLs to running crawler |
| `stats(crawler_id)` | Get crawler statistics |
| `close()` | Close the connection |

### AsyncCrawlerClient

Same API as `CrawlerClient` but with `async`/`await` support, plus:

| Method | Description |
|--------|-------------|
| `crawl_stream(urls, **kwargs)` | Stream results as they arrive |

### Types

- `CrawlConfig(max_depth, max_pages, respect_robots_txt, urls)`
- `CrawlResult(url, status_code, content, crawled_at, duration, error)`
- `CrawlStats(pages_crawled, pages_failed, pages_queued)`

### Exceptions

- `CrawlerError` - Base exception
- `ConnectionError` - Server unreachable
- `CrawlError` - Crawl operation failed
- `NotFoundError` - Crawler not found
- `AlreadyRunningError` - Crawler already started

## Requirements

- Python 3.10+
- Running Web Crawler gRPC server

## Development

```bash
pip install -e ".[dev]"
pytest
mypy crawler/
ruff check crawler/
```
