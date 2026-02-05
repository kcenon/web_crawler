# Additional Considerations for Web Crawling

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Important topics not explicitly requested but essential for production crawlers

## Overview

This document covers critical aspects that are often overlooked but essential for building robust, production-ready web crawlers.

---

## 1. Security Considerations

### 1.1 Credential Management

**Never hardcode sensitive data**:

```python
# ❌ WRONG - Never do this
PROXY_PASSWORD = "my_secret_password"
API_KEY = "sk-1234567890abcdef"

# ✅ CORRECT - Use environment variables
import os
PROXY_PASSWORD = os.environ.get('PROXY_PASSWORD')
API_KEY = os.environ.get('CRAWLER_API_KEY')

# ✅ BETTER - Use secret management
from aws_secrets import get_secret
credentials = get_secret('crawler/credentials')
```

### 1.2 Secrets Management Options

| Tool | Best For | Features |
|------|----------|----------|
| **AWS Secrets Manager** | AWS infrastructure | Rotation, audit |
| **HashiCorp Vault** | Multi-cloud | Dynamic secrets |
| **Azure Key Vault** | Azure infrastructure | Integration |
| **dotenv + .gitignore** | Development | Simple, local |

### 1.3 Secure Configuration

```python
# config.py
from pydantic_settings import BaseSettings
from pydantic import SecretStr

class CrawlerSettings(BaseSettings):
    proxy_username: str
    proxy_password: SecretStr
    database_url: SecretStr
    api_key: SecretStr

    class Config:
        env_file = '.env'
        env_file_encoding = 'utf-8'

settings = CrawlerSettings()

# Usage - password is masked in logs
print(settings.proxy_password)  # Output: SecretStr('**********')
print(settings.proxy_password.get_secret_value())  # Actual value
```

### 1.4 Network Security

```python
# Verify SSL certificates (default, but ensure it's enabled)
import requests
response = requests.get(url, verify=True)

# Use secure connections
DATABASE_URL = "postgresql+psycopg2://user:pass@host:5432/db?sslmode=require"
REDIS_URL = "rediss://user:pass@host:6379"  # Note: rediss:// for SSL
```

---

## 2. Testing Strategy

### 2.1 Test Types for Crawlers

| Type | Purpose | Tools |
|------|---------|-------|
| **Unit Tests** | Individual functions | pytest, unittest |
| **Integration Tests** | Database, APIs | pytest, testcontainers |
| **Mock Tests** | HTTP responses | responses, httpretty |
| **End-to-End** | Full crawl flow | Real test sites |

### 2.2 Mock HTTP Responses

```python
import pytest
import responses
from crawler import fetch_page

@responses.activate
def test_successful_fetch():
    """Test successful page fetch"""
    responses.add(
        responses.GET,
        "https://example.com/page",
        body="<html><title>Test</title></html>",
        status=200,
        headers={"Content-Type": "text/html"}
    )

    result = fetch_page("https://example.com/page")

    assert result.status_code == 200
    assert "Test" in result.content

@responses.activate
def test_retry_on_500():
    """Test retry behavior on server error"""
    responses.add(responses.GET, "https://example.com/page", status=500)
    responses.add(responses.GET, "https://example.com/page", status=500)
    responses.add(
        responses.GET,
        "https://example.com/page",
        body="<html>Success</html>",
        status=200
    )

    result = fetch_page("https://example.com/page", max_retries=3)

    assert result.status_code == 200
    assert len(responses.calls) == 3
```

### 2.3 Testing Parsers

```python
import pytest
from bs4 import BeautifulSoup
from parsers import ProductParser

@pytest.fixture
def sample_product_html():
    return """
    <html>
        <div class="product">
            <h1 class="title">Test Product</h1>
            <span class="price">$99.99</span>
            <div class="stock">In Stock</div>
        </div>
    </html>
    """

def test_product_parser(sample_product_html):
    """Test product data extraction"""
    parser = ProductParser()
    result = parser.parse(sample_product_html)

    assert result['name'] == 'Test Product'
    assert result['price'] == 99.99
    assert result['in_stock'] == True

def test_missing_price():
    """Test handling of missing price"""
    html = "<html><h1 class='title'>Product</h1></html>"
    parser = ProductParser()

    result = parser.parse(html)

    assert result['price'] is None
    assert result['name'] == 'Product'
```

### 2.4 Integration Testing with Testcontainers

```python
import pytest
from testcontainers.postgres import PostgresContainer
from testcontainers.redis import RedisContainer

@pytest.fixture(scope="module")
def postgres_db():
    """Spin up PostgreSQL for testing"""
    with PostgresContainer("postgres:15") as postgres:
        yield postgres.get_connection_url()

@pytest.fixture(scope="module")
def redis_cache():
    """Spin up Redis for testing"""
    with RedisContainer("redis:7") as redis:
        yield redis.get_connection_url()

def test_database_storage(postgres_db):
    """Test full database operations"""
    from storage import DatabaseManager

    db = DatabaseManager(postgres_db)
    db.save_page(url="https://test.com", status_code=200, content="test")

    result = db.get_page("https://test.com")
    assert result['status_code'] == 200
```

---

## 3. Cost Optimization

### 3.1 Cost Factors

| Component | Cost Driver | Optimization |
|-----------|-------------|--------------|
| **Proxies** | Request volume | Batch requests, caching |
| **Compute** | CPU/Memory | Efficient parsing, async I/O |
| **Storage** | Data volume | Compression, deduplication |
| **Bandwidth** | Downloaded data | Block images/media, compression |

### 3.2 Proxy Cost Reduction

```python
class CostAwareCrawler:
    def __init__(self):
        self.cache = {}
        self.cache_ttl = 3600  # 1 hour

    async def fetch_with_cache(self, url: str) -> dict:
        """Fetch with local caching to reduce proxy usage"""
        cache_key = hashlib.md5(url.encode()).hexdigest()

        # Check cache first
        if cache_key in self.cache:
            cached = self.cache[cache_key]
            if time.time() - cached['time'] < self.cache_ttl:
                return cached['data']

        # Fetch if not cached
        response = await self.fetch(url)
        self.cache[cache_key] = {
            'data': response,
            'time': time.time()
        }

        return response

    def should_use_proxy(self, domain: str) -> bool:
        """Determine if proxy is needed for domain"""
        # Use direct connection for friendly sites
        friendly_domains = ['wikipedia.org', 'github.com']
        return domain not in friendly_domains
```

### 3.3 Bandwidth Optimization

```python
from playwright.async_api import async_playwright

async def create_bandwidth_optimized_page(browser):
    """Create page that blocks unnecessary resources"""
    context = await browser.new_context()
    page = await context.new_page()

    # Block heavy resources
    await page.route("**/*", lambda route: (
        route.abort() if route.request.resource_type in [
            'image', 'media', 'font', 'stylesheet'
        ] else route.continue_()
    ))

    return page

# Scrapy settings for bandwidth optimization
HTTPCACHE_ENABLED = True
HTTPCACHE_EXPIRATION_SECS = 86400  # 24 hours
HTTPCACHE_DIR = 'httpcache'
```

### 3.4 Compute Optimization

```python
import asyncio
from concurrent.futures import ProcessPoolExecutor

class OptimizedParser:
    def __init__(self, max_workers: int = 4):
        self.executor = ProcessPoolExecutor(max_workers=max_workers)

    async def parse_batch(self, html_pages: list[str]) -> list[dict]:
        """Parse multiple pages in parallel using process pool"""
        loop = asyncio.get_event_loop()

        # CPU-bound parsing in separate processes
        tasks = [
            loop.run_in_executor(self.executor, self.parse_single, html)
            for html in html_pages
        ]

        return await asyncio.gather(*tasks)

    def parse_single(self, html: str) -> dict:
        """Parse single page (runs in separate process)"""
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(html, 'lxml')  # lxml is faster
        return self.extract_data(soup)
```

---

## 4. Common Pitfalls and Solutions

### 4.1 Memory Leaks

**Problem**: Memory grows unbounded during long crawls.

```python
# ❌ Memory leak - storing all responses
class BadCrawler:
    def __init__(self):
        self.all_responses = []  # Grows forever!

    def crawl(self, url):
        response = requests.get(url)
        self.all_responses.append(response)  # Memory leak!

# ✅ Fixed - process and discard
class GoodCrawler:
    def crawl(self, url):
        response = requests.get(url)
        data = self.extract_data(response)
        self.save_to_db(data)
        # Response is garbage collected
```

### 4.2 Infinite Loops

**Problem**: Crawler gets stuck in URL loops.

```python
# ✅ Solution - track visited URLs with depth limit
class SafeCrawler:
    def __init__(self, max_depth: int = 5):
        self.visited = set()
        self.max_depth = max_depth

    def crawl(self, url: str, depth: int = 0):
        # Prevent infinite loops
        if depth > self.max_depth:
            return

        normalized_url = self.normalize_url(url)
        if normalized_url in self.visited:
            return

        self.visited.add(normalized_url)

        response = self.fetch(url)
        links = self.extract_links(response)

        for link in links:
            self.crawl(link, depth + 1)
```

### 4.3 Session/Cookie Issues

**Problem**: Site requires login but session expires.

```python
class SessionManager:
    def __init__(self, login_url: str, credentials: dict):
        self.session = requests.Session()
        self.login_url = login_url
        self.credentials = credentials
        self.last_login = None
        self.session_duration = 3600  # 1 hour

    def ensure_logged_in(self):
        """Re-login if session expired"""
        if self.last_login and time.time() - self.last_login < self.session_duration:
            return

        response = self.session.post(self.login_url, data=self.credentials)
        if response.status_code == 200:
            self.last_login = time.time()
        else:
            raise LoginError("Failed to login")

    def get(self, url: str):
        """Get with automatic re-login"""
        self.ensure_logged_in()
        return self.session.get(url)
```

### 4.4 Character Encoding Issues

**Problem**: Garbled text from different encodings.

```python
import chardet

def safe_decode(content: bytes) -> str:
    """Safely decode content with encoding detection"""
    # Try to detect encoding
    detected = chardet.detect(content)
    encoding = detected['encoding'] or 'utf-8'

    try:
        return content.decode(encoding)
    except UnicodeDecodeError:
        # Fallback encodings
        for fallback in ['utf-8', 'latin-1', 'cp949', 'euc-kr']:
            try:
                return content.decode(fallback)
            except UnicodeDecodeError:
                continue

    # Last resort - ignore errors
    return content.decode('utf-8', errors='ignore')
```

### 4.5 Dynamic Content Not Loading

**Problem**: JavaScript content not rendered.

```python
# ✅ Solution - wait for specific elements
async def wait_for_content(page, selector: str, timeout: int = 30000):
    """Wait for dynamic content to load"""
    try:
        await page.wait_for_selector(selector, timeout=timeout)
    except TimeoutError:
        # Try scrolling to trigger lazy loading
        await page.evaluate("window.scrollTo(0, document.body.scrollHeight)")
        await asyncio.sleep(1)
        await page.wait_for_selector(selector, timeout=5000)

    return await page.content()
```

---

## 5. Performance Benchmarking

### 5.1 Metrics to Track

```python
import time
from dataclasses import dataclass, field
from statistics import mean, median, stdev

@dataclass
class CrawlMetrics:
    requests_total: int = 0
    requests_success: int = 0
    requests_failed: int = 0
    bytes_downloaded: int = 0
    response_times: list = field(default_factory=list)
    start_time: float = field(default_factory=time.time)

    @property
    def elapsed_time(self) -> float:
        return time.time() - self.start_time

    @property
    def requests_per_second(self) -> float:
        return self.requests_total / max(self.elapsed_time, 1)

    @property
    def success_rate(self) -> float:
        return self.requests_success / max(self.requests_total, 1) * 100

    @property
    def avg_response_time(self) -> float:
        return mean(self.response_times) if self.response_times else 0

    def report(self) -> dict:
        return {
            'total_requests': self.requests_total,
            'success_rate': f"{self.success_rate:.1f}%",
            'requests_per_second': f"{self.requests_per_second:.1f}",
            'avg_response_time_ms': f"{self.avg_response_time * 1000:.0f}",
            'total_bytes': f"{self.bytes_downloaded / 1024 / 1024:.1f} MB",
            'elapsed_time': f"{self.elapsed_time:.0f}s",
        }
```

### 5.2 Load Testing

```python
import asyncio
import aiohttp

async def benchmark_crawler(urls: list[str], concurrency: int = 10):
    """Benchmark crawler performance"""
    metrics = CrawlMetrics()

    async def fetch_one(session, url):
        start = time.time()
        try:
            async with session.get(url) as response:
                content = await response.read()
                metrics.requests_success += 1
                metrics.bytes_downloaded += len(content)
        except Exception:
            metrics.requests_failed += 1
        finally:
            metrics.requests_total += 1
            metrics.response_times.append(time.time() - start)

    connector = aiohttp.TCPConnector(limit=concurrency)
    async with aiohttp.ClientSession(connector=connector) as session:
        tasks = [fetch_one(session, url) for url in urls]
        await asyncio.gather(*tasks)

    return metrics.report()

# Usage
results = asyncio.run(benchmark_crawler(test_urls, concurrency=20))
print(results)
```

---

## 6. Deployment Considerations

### 6.1 Containerization

```dockerfile
# Dockerfile
FROM python:3.11-slim

# Install system dependencies
RUN apt-get update && apt-get install -y \
    libxml2-dev \
    libxslt1-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Playwright browsers
RUN pip install playwright && playwright install chromium --with-deps

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

CMD ["python", "-m", "crawler.main"]
```

### 6.2 Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-crawler
spec:
  replicas: 3
  selector:
    matchLabels:
      app: web-crawler
  template:
    metadata:
      labels:
        app: web-crawler
    spec:
      containers:
      - name: crawler
        image: my-crawler:latest
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        env:
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: crawler-secrets
              key: redis-url
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: crawler-secrets
              key: database-url
```

### 6.3 Monitoring Stack

```yaml
# docker-compose.yml for monitoring
version: '3.8'
services:
  prometheus:
    image: prom/prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  grafana-data:
```

---

## 7. Documentation Requirements

### 7.1 What to Document

| Document | Contents |
|----------|----------|
| **README** | Setup, usage, configuration |
| **Architecture** | System design, data flow |
| **API Docs** | Internal APIs, data formats |
| **Runbook** | Operations, troubleshooting |
| **Data Dictionary** | Field definitions, schemas |

### 7.2 Runbook Template

```markdown
# Crawler Runbook

## Starting the Crawler
1. Ensure Redis is running: `docker ps | grep redis`
2. Start crawler: `python -m crawler.main`
3. Monitor logs: `tail -f logs/crawler.log`

## Common Issues

### High Error Rate
- Check: `grep ERROR logs/crawler.log | tail -20`
- Likely causes: Proxy issues, rate limiting
- Resolution: Reduce concurrency, rotate proxies

### Memory Growing
- Check: `docker stats crawler`
- Likely causes: Response accumulation, cache leak
- Resolution: Restart crawler, check for memory leaks

## Scaling Up
1. Add more replicas: `kubectl scale deployment crawler --replicas=5`
2. Monitor queue: `redis-cli llen crawl_queue`
```

---

## Summary Checklist

### Before Production

- [ ] Security audit completed
- [ ] All credentials in secret management
- [ ] Unit and integration tests passing
- [ ] Load testing completed
- [ ] Monitoring and alerting configured
- [ ] Runbook documented
- [ ] Cost estimates reviewed

### Ongoing Operations

- [ ] Regular security updates
- [ ] Monitor error rates and costs
- [ ] Review and update TOS compliance
- [ ] Performance optimization reviews
- [ ] Backup verification

---

*Production crawlers require attention to many details beyond basic functionality.*
