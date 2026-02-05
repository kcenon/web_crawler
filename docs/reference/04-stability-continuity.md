# Stable and Continuous Crawling Guide

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Strategies for reliable, long-term web crawling operations

## Overview

Building a crawler that operates reliably over time requires careful attention to error handling, resource management, and defensive programming against anti-bot measures.

---

## 1. Distributed Architecture

### 1.1 System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    Distributed Crawler Architecture             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐        │
│   │   URL       │    │  Scheduler  │    │   Worker    │        │
│   │  Frontier   │───▶│   Service   │───▶│   Nodes     │        │
│   │  (Kafka)    │    │             │    │  (N nodes)  │        │
│   └─────────────┘    └─────────────┘    └──────┬──────┘        │
│         │                                       │               │
│         │            ┌─────────────┐           │               │
│         └───────────▶│    Redis    │◀──────────┘               │
│                      │  (Dedup/    │                           │
│                      │   Cache)    │                           │
│                      └─────────────┘                           │
│                             │                                   │
│                      ┌──────▼──────┐                           │
│                      │  Database   │                           │
│                      │ (PostgreSQL/│                           │
│                      │  MongoDB)   │                           │
│                      └─────────────┘                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Redis Role in Distribution

**Key Functions**:
- **Message Broker**: Manages URL queue distribution
- **Deduplication Filter**: Tracks visited URLs using Sets
- **Rate Limit Tracking**: Per-domain request counting
- **Crawl State Cache**: Quick lookup for crawl status

```python
import redis
from urllib.parse import urlparse
import hashlib

class RedisURLManager:
    def __init__(self, redis_url: str = 'redis://localhost:6379'):
        self.redis = redis.from_url(redis_url)
        self.visited_key = 'crawler:visited'
        self.queue_key = 'crawler:queue'

    def url_hash(self, url: str) -> str:
        """Generate consistent hash for URL"""
        return hashlib.sha256(url.encode()).hexdigest()[:16]

    def is_visited(self, url: str) -> bool:
        """Check if URL was already crawled"""
        return self.redis.sismember(self.visited_key, self.url_hash(url))

    def mark_visited(self, url: str):
        """Mark URL as visited"""
        self.redis.sadd(self.visited_key, self.url_hash(url))

    def add_to_queue(self, url: str, priority: int = 0):
        """Add URL to crawl queue with priority"""
        if not self.is_visited(url):
            self.redis.zadd(self.queue_key, {url: priority})

    def get_next_url(self) -> str | None:
        """Get next URL to crawl (highest priority first)"""
        result = self.redis.zpopmax(self.queue_key, count=1)
        return result[0][0] if result else None
```

### 1.3 Domain-Specific Rate Limiting

```python
import time

class DomainRateLimiter:
    def __init__(self, redis_client, requests_per_second: float = 1.0):
        self.redis = redis_client
        self.rps = requests_per_second
        self.interval = 1.0 / requests_per_second

    def can_request(self, domain: str) -> bool:
        """Check if we can make a request to this domain"""
        key = f"ratelimit:{domain}"
        last_request = self.redis.get(key)

        if last_request is None:
            return True

        elapsed = time.time() - float(last_request)
        return elapsed >= self.interval

    def record_request(self, domain: str):
        """Record that a request was made"""
        key = f"ratelimit:{domain}"
        self.redis.set(key, time.time(), ex=3600)  # 1 hour expiry

    async def wait_if_needed(self, domain: str):
        """Wait until we can make a request"""
        key = f"ratelimit:{domain}"
        last_request = self.redis.get(key)

        if last_request:
            elapsed = time.time() - float(last_request)
            if elapsed < self.interval:
                await asyncio.sleep(self.interval - elapsed)
```

---

## 2. Error Handling & Retry Strategies

### 2.1 HTTP Status Code Handling

| Status Code | Meaning | Action |
|-------------|---------|--------|
| **200** | Success | Process content |
| **301/302** | Redirect | Follow (with limit) |
| **403** | Forbidden | Check robots.txt, stop or rotate |
| **404** | Not Found | Mark as dead, don't retry |
| **429** | Rate Limited | Exponential backoff |
| **500-503** | Server Error | Retry with delay |
| **504** | Gateway Timeout | Retry |

### 2.2 Retry Strategy Implementation

```python
from dataclasses import dataclass
from enum import Enum
import asyncio
import random

class RetryDecision(Enum):
    RETRY = "retry"
    SKIP = "skip"
    FATAL = "fatal"

@dataclass
class RetryConfig:
    max_retries: int = 3
    base_delay: float = 1.0
    max_delay: float = 60.0
    exponential_base: float = 2.0
    jitter: float = 0.1

class RetryHandler:
    def __init__(self, config: RetryConfig = None):
        self.config = config or RetryConfig()

    def should_retry(self, status_code: int, attempt: int) -> RetryDecision:
        """Determine if request should be retried"""
        if attempt >= self.config.max_retries:
            return RetryDecision.SKIP

        # Never retry these
        if status_code in [400, 401, 403, 404, 410]:
            return RetryDecision.SKIP

        # Always retry these
        if status_code in [429, 500, 502, 503, 504, 520, 521, 522, 523, 524]:
            return RetryDecision.RETRY

        # Unknown error
        if status_code >= 400:
            return RetryDecision.SKIP

        return RetryDecision.FATAL

    def calculate_delay(self, attempt: int, status_code: int = None) -> float:
        """Calculate delay before retry with exponential backoff"""
        delay = self.config.base_delay * (self.config.exponential_base ** attempt)

        # Extra delay for rate limiting
        if status_code == 429:
            delay *= 2

        # Add jitter
        jitter_range = delay * self.config.jitter
        delay += random.uniform(-jitter_range, jitter_range)

        return min(delay, self.config.max_delay)

    async def execute_with_retry(self, request_func, url: str):
        """Execute request with automatic retry"""
        for attempt in range(self.config.max_retries + 1):
            try:
                response = await request_func(url)
                return response
            except Exception as e:
                status_code = getattr(e, 'status_code', 500)
                decision = self.should_retry(status_code, attempt)

                if decision == RetryDecision.SKIP:
                    raise
                elif decision == RetryDecision.RETRY:
                    delay = self.calculate_delay(attempt, status_code)
                    await asyncio.sleep(delay)
                else:
                    raise

        raise MaxRetriesExceeded(url)
```

### 2.3 Crawlee Error Handling

```python
from crawlee.playwright_crawler import PlaywrightCrawler
from crawlee.errors import RequestHandlerError

crawler = PlaywrightCrawler(
    max_request_retries=3,
    request_handler_timeout=timedelta(seconds=60),
)

@crawler.failed_request_handler
async def handle_failed_request(context, error):
    """Handle permanently failed requests"""
    url = context.request.url
    error_type = type(error).__name__

    # Log for analysis
    logger.error(f"Failed: {url} - {error_type}: {error}")

    # Store for later review
    await context.push_data({
        'url': url,
        'error': str(error),
        'error_type': error_type,
        'timestamp': datetime.utcnow().isoformat(),
    }, dataset_name='failed_requests')
```

---

## 3. Anti-Bot Detection Evasion

### 3.1 Detection Mechanisms (2025)

Modern anti-bot systems analyze:

| Layer | Detection Method |
|-------|-----------------|
| **Network** | TLS fingerprint (JA3/JA4), IP reputation |
| **HTTP** | Header order, timing patterns |
| **Browser** | Canvas/WebGL/Audio fingerprint |
| **Behavior** | Mouse movement, scroll patterns |
| **Session** | Cookie consistency, state tracking |

### 3.2 Evasion Strategies

#### TLS Fingerprint Management

```python
# Use curl_cffi for realistic TLS fingerprints
from curl_cffi import requests

# Impersonate Chrome browser
response = requests.get(
    url,
    impersonate="chrome110",
    headers=headers
)
```

#### Browser Fingerprint Randomization

```python
from playwright.async_api import async_playwright

async def create_stealth_context(browser):
    """Create browser context with fingerprint evasion"""
    context = await browser.new_context(
        viewport={'width': random.randint(1200, 1920),
                  'height': random.randint(800, 1080)},
        locale=random.choice(['en-US', 'en-GB', 'ko-KR']),
        timezone_id=random.choice(['Asia/Seoul', 'America/New_York']),
        color_scheme=random.choice(['light', 'dark']),
        device_scale_factor=random.choice([1, 1.25, 1.5, 2]),
    )

    # Inject stealth scripts
    await context.add_init_script("""
        // Override navigator properties
        Object.defineProperty(navigator, 'webdriver', {
            get: () => undefined
        });

        // Randomize canvas fingerprint
        const originalToDataURL = HTMLCanvasElement.prototype.toDataURL;
        HTMLCanvasElement.prototype.toDataURL = function(type) {
            if (type === 'image/png' && this.width > 0) {
                const ctx = this.getContext('2d');
                const noise = Math.random() * 0.01;
                // Add subtle noise
            }
            return originalToDataURL.apply(this, arguments);
        };
    """)

    return context
```

### 3.3 Proxy Rotation Strategies

#### Session-Based Rotation

```python
class SessionProxyManager:
    def __init__(self, proxies: list[str]):
        self.proxies = proxies
        self.sessions = {}  # domain -> (proxy, start_time, request_count)
        self.session_duration = 300  # 5 minutes
        self.max_requests_per_session = 50

    def get_proxy_for_domain(self, domain: str) -> str:
        """Get or create session proxy for domain"""
        current_time = time.time()

        if domain in self.sessions:
            proxy, start_time, count = self.sessions[domain]

            # Check if session is still valid
            elapsed = current_time - start_time
            if elapsed < self.session_duration and count < self.max_requests_per_session:
                self.sessions[domain] = (proxy, start_time, count + 1)
                return proxy

        # Create new session
        proxy = random.choice(self.proxies)
        self.sessions[domain] = (proxy, current_time, 1)
        return proxy

    def rotate_proxy(self, domain: str):
        """Force rotation for domain"""
        if domain in self.sessions:
            del self.sessions[domain]
```

#### Health-Based Rotation

```python
class HealthBasedProxyManager:
    def __init__(self, proxies: list[str]):
        self.proxies = {p: {'success': 0, 'fail': 0, 'blocked': False}
                       for p in proxies}

    def get_healthy_proxy(self) -> str:
        """Get proxy with best success rate"""
        healthy = [p for p, stats in self.proxies.items()
                   if not stats['blocked']]

        if not healthy:
            # Reset all if all blocked
            for stats in self.proxies.values():
                stats['blocked'] = False
            healthy = list(self.proxies.keys())

        # Weight by success rate
        weights = []
        for proxy in healthy:
            stats = self.proxies[proxy]
            total = stats['success'] + stats['fail']
            rate = stats['success'] / total if total > 0 else 0.5
            weights.append(rate + 0.1)  # Minimum weight

        return random.choices(healthy, weights=weights)[0]

    def record_result(self, proxy: str, success: bool, blocked: bool = False):
        """Record request result for proxy"""
        if success:
            self.proxies[proxy]['success'] += 1
        else:
            self.proxies[proxy]['fail'] += 1

        if blocked:
            self.proxies[proxy]['blocked'] = True
```

---

## 4. Monitoring & Logging

### 4.1 Key Metrics to Track

| Category | Metrics |
|----------|---------|
| **Performance** | Requests/second, response time, queue size |
| **Success** | Success rate, error rate by type |
| **Resources** | Memory usage, CPU, network bandwidth |
| **Business** | Pages crawled, data extracted, coverage |

### 4.2 Structured Logging

```python
import structlog
from datetime import datetime

logger = structlog.get_logger()

class CrawlLogger:
    def __init__(self):
        self.metrics = {
            'requests_total': 0,
            'requests_success': 0,
            'requests_failed': 0,
            'bytes_downloaded': 0,
        }

    def log_request(self, url: str, status: int, duration: float,
                    size: int, error: str = None):
        """Log individual request with context"""
        self.metrics['requests_total'] += 1

        if 200 <= status < 300:
            self.metrics['requests_success'] += 1
            self.metrics['bytes_downloaded'] += size

            logger.info(
                "request_success",
                url=url,
                status=status,
                duration_ms=round(duration * 1000, 2),
                size_bytes=size,
            )
        else:
            self.metrics['requests_failed'] += 1

            logger.warning(
                "request_failed",
                url=url,
                status=status,
                duration_ms=round(duration * 1000, 2),
                error=error,
            )

    def log_metrics(self):
        """Log aggregated metrics"""
        success_rate = (self.metrics['requests_success'] /
                       max(self.metrics['requests_total'], 1) * 100)

        logger.info(
            "crawler_metrics",
            requests_total=self.metrics['requests_total'],
            success_rate=round(success_rate, 2),
            bytes_downloaded=self.metrics['bytes_downloaded'],
        )
```

### 4.3 Alerting Thresholds

```python
class CrawlHealthMonitor:
    def __init__(self):
        self.thresholds = {
            'error_rate': 0.10,  # Alert if >10% errors
            'avg_response_time': 5.0,  # Alert if >5s average
            'queue_growth_rate': 100,  # Alert if queue growing too fast
            'memory_usage': 0.85,  # Alert if >85% memory
        }
        self.window = []
        self.window_size = 100

    def record(self, success: bool, response_time: float):
        """Record request result"""
        self.window.append({
            'success': success,
            'response_time': response_time,
            'timestamp': time.time()
        })

        # Keep window size limited
        if len(self.window) > self.window_size:
            self.window.pop(0)

        self.check_health()

    def check_health(self):
        """Check if crawler is healthy"""
        if len(self.window) < 10:
            return

        # Calculate metrics
        error_rate = sum(1 for r in self.window if not r['success']) / len(self.window)
        avg_response = sum(r['response_time'] for r in self.window) / len(self.window)

        # Check thresholds
        if error_rate > self.thresholds['error_rate']:
            self.alert(f"High error rate: {error_rate:.1%}")

        if avg_response > self.thresholds['avg_response_time']:
            self.alert(f"Slow responses: {avg_response:.1f}s average")

    def alert(self, message: str):
        """Send alert"""
        logger.error("health_alert", message=message)
        # Integrate with alerting system (Slack, PagerDuty, etc.)
```

---

## 5. State Persistence & Recovery

### 5.1 Checkpoint System

```python
import json
import pickle
from pathlib import Path

class CrawlCheckpoint:
    def __init__(self, checkpoint_dir: str = './checkpoints'):
        self.checkpoint_dir = Path(checkpoint_dir)
        self.checkpoint_dir.mkdir(exist_ok=True)

    def save(self, state: dict, name: str = 'latest'):
        """Save crawler state"""
        checkpoint_path = self.checkpoint_dir / f'{name}.pkl'

        with open(checkpoint_path, 'wb') as f:
            pickle.dump({
                'state': state,
                'timestamp': datetime.utcnow().isoformat(),
            }, f)

        logger.info("checkpoint_saved", path=str(checkpoint_path))

    def load(self, name: str = 'latest') -> dict | None:
        """Load crawler state"""
        checkpoint_path = self.checkpoint_dir / f'{name}.pkl'

        if not checkpoint_path.exists():
            return None

        with open(checkpoint_path, 'rb') as f:
            data = pickle.load(f)

        logger.info(
            "checkpoint_loaded",
            path=str(checkpoint_path),
            saved_at=data['timestamp']
        )

        return data['state']

    def save_periodic(self, state: dict, interval: int = 300):
        """Save checkpoint every N seconds"""
        timestamp = int(time.time())
        if timestamp % interval == 0:
            self.save(state, f'checkpoint_{timestamp}')
            self.save(state, 'latest')
```

### 5.2 Graceful Shutdown

```python
import signal
import asyncio

class GracefulCrawler:
    def __init__(self):
        self.running = True
        self.checkpoint = CrawlCheckpoint()
        self.state = {'queue': [], 'visited': set()}

        # Register signal handlers
        signal.signal(signal.SIGINT, self.handle_shutdown)
        signal.signal(signal.SIGTERM, self.handle_shutdown)

    def handle_shutdown(self, signum, frame):
        """Handle shutdown signal"""
        logger.info("shutdown_initiated", signal=signum)
        self.running = False

    async def run(self):
        """Main crawl loop with graceful shutdown"""
        try:
            while self.running and self.state['queue']:
                url = self.state['queue'].pop(0)
                await self.crawl_url(url)

                # Periodic checkpoint
                if len(self.state['visited']) % 100 == 0:
                    self.checkpoint.save(self.state)

        finally:
            # Save final state
            logger.info("saving_final_state")
            self.checkpoint.save(self.state)
            logger.info("crawler_stopped")
```

---

## 6. Resource Management

### 6.1 Memory Management

```python
import gc
import sys

class MemoryManager:
    def __init__(self, max_memory_mb: int = 4096):
        self.max_memory = max_memory_mb * 1024 * 1024

    def get_memory_usage(self) -> int:
        """Get current memory usage in bytes"""
        import psutil
        process = psutil.Process()
        return process.memory_info().rss

    def check_memory(self):
        """Check and manage memory"""
        usage = self.get_memory_usage()
        usage_percent = usage / self.max_memory

        if usage_percent > 0.9:
            logger.warning("memory_critical", usage_percent=f"{usage_percent:.1%}")
            self.force_cleanup()
        elif usage_percent > 0.7:
            logger.info("memory_high", usage_percent=f"{usage_percent:.1%}")
            gc.collect()

    def force_cleanup(self):
        """Aggressive memory cleanup"""
        gc.collect()
        # Clear caches, reduce batch sizes, etc.
```

### 6.2 Connection Pool Management

```python
import aiohttp

class ConnectionPoolManager:
    def __init__(self, max_connections: int = 100):
        self.connector = aiohttp.TCPConnector(
            limit=max_connections,
            limit_per_host=10,
            ttl_dns_cache=300,
            enable_cleanup_closed=True,
        )
        self.session = None

    async def get_session(self) -> aiohttp.ClientSession:
        """Get or create session"""
        if self.session is None or self.session.closed:
            self.session = aiohttp.ClientSession(
                connector=self.connector,
                timeout=aiohttp.ClientTimeout(total=30),
            )
        return self.session

    async def close(self):
        """Clean up connections"""
        if self.session:
            await self.session.close()
        await self.connector.close()
```

---

## References

- [Building Distributed Web Crawler with Python and Redis](https://medium.com/pythoneers/building-a-distributed-web-crawler-with-python-asyncio-and-redis-queues-8613a6f061d8)
- [Distributed Web Crawling Guide](https://brightdata.com/blog/web-data/distributed-web-crawling)
- [Proxy Strategy in 2025](https://scrapingant.com/blog/proxy-strategy-in-2025-beating-anti-bot-systems-without)
- [Bypass Bot Detection 2026](https://www.zenrows.com/blog/bypass-bot-detection)
- [Crawlee Error Handling](https://crawlee.dev/python/docs/guides/error-handling)

---

*Reliable crawling requires continuous monitoring and adaptation to changing conditions.*
