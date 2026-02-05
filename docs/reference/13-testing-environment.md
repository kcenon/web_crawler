# Web Crawler Testing Environment

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Testing infrastructure, mock servers, and quality assurance for web crawlers

## Overview

Reliable web crawlers require comprehensive testing. This guide covers setting up test environments, mock servers, and automated testing strategies.

---

## 1. Testing Pyramid for Crawlers

### 1.1 Test Levels

```
                    ┌─────────────┐
                   │   E2E Tests  │  ← Real websites (few)
                  │  Integration  │
                 └───────────────┘
              ┌─────────────────────┐
             │   Integration Tests  │  ← Mock servers
            │    (Component Tests)  │
           └───────────────────────┘
        ┌─────────────────────────────┐
       │        Unit Tests             │  ← Pure functions
      │    (Parsers, Validators)       │
     └─────────────────────────────────┘
```

### 1.2 What to Test at Each Level

| Level | What to Test | Tools |
|-------|--------------|-------|
| **Unit** | Parsers, validators, transformers | pytest, unittest |
| **Integration** | HTTP handling, retries, pipelines | pytest-httpbin, responses |
| **E2E** | Full crawl workflow | Real test sites, staging |

---

## 2. Unit Testing Parsers

### 2.1 Parser Test Structure

```python
import pytest
from parsers import ProductParser

class TestProductParser:
    """Test product data extraction"""

    @pytest.fixture
    def parser(self):
        return ProductParser()

    @pytest.fixture
    def sample_html(self):
        return '''
        <html>
            <div class="product">
                <h1 class="title">Test Product</h1>
                <span class="price">$99.99</span>
                <div class="description">Product description here</div>
                <span class="stock in-stock">In Stock</span>
            </div>
        </html>
        '''

    def test_extract_title(self, parser, sample_html):
        """Test title extraction"""
        result = parser.parse(sample_html)
        assert result['title'] == 'Test Product'

    def test_extract_price(self, parser, sample_html):
        """Test price extraction and parsing"""
        result = parser.parse(sample_html)
        assert result['price'] == 99.99
        assert result['currency'] == 'USD'

    def test_extract_stock_status(self, parser, sample_html):
        """Test stock status detection"""
        result = parser.parse(sample_html)
        assert result['in_stock'] is True

    def test_missing_price(self, parser):
        """Test handling of missing price"""
        html = '<div class="product"><h1>No Price Product</h1></div>'
        result = parser.parse(html)
        assert result['price'] is None

    def test_out_of_stock_detection(self, parser):
        """Test out of stock detection"""
        html = '''
        <div class="product">
            <h1>Product</h1>
            <span class="stock out-of-stock">Sold Out</span>
        </div>
        '''
        result = parser.parse(html)
        assert result['in_stock'] is False

    @pytest.mark.parametrize('price_text,expected', [
        ('$99.99', 99.99),
        ('$1,234.56', 1234.56),
        ('₩99,000', 99000),
        ('€ 49,99', 49.99),
        ('Free', 0),
    ])
    def test_price_parsing_formats(self, parser, price_text, expected):
        """Test various price formats"""
        result = parser.parse_price(price_text)
        assert result == expected
```

### 2.2 Fixtures for Common HTML Patterns

```python
# conftest.py
import pytest

@pytest.fixture
def ecommerce_product_page():
    """Standard e-commerce product page HTML"""
    return '''
    <!DOCTYPE html>
    <html>
    <head>
        <script type="application/ld+json">
        {
            "@context": "https://schema.org",
            "@type": "Product",
            "name": "Test Widget",
            "offers": {
                "@type": "Offer",
                "price": "29.99",
                "priceCurrency": "USD"
            }
        }
        </script>
    </head>
    <body>
        <h1 itemprop="name">Test Widget</h1>
        <span itemprop="price" content="29.99">$29.99</span>
    </body>
    </html>
    '''

@pytest.fixture
def news_article_page():
    """Standard news article HTML"""
    return '''
    <!DOCTYPE html>
    <html>
    <body>
        <article>
            <h1 class="headline">Breaking News Title</h1>
            <span class="author">John Doe</span>
            <time datetime="2026-02-05T10:00:00Z">Feb 5, 2026</time>
            <div class="content">
                <p>First paragraph of the article.</p>
                <p>Second paragraph with more details.</p>
            </div>
        </article>
    </body>
    </html>
    '''

@pytest.fixture
def html_with_encoding_issues():
    """HTML with various encoding scenarios"""
    return {
        'utf8': b'<html><body>\xc3\xa9\xc3\xa0\xc3\xbc</body></html>',  # éàü
        'latin1': b'<html><body>\xe9\xe0\xfc</body></html>',
        'korean': '<html><body>한글 테스트</body></html>'.encode('euc-kr'),
    }
```

---

## 3. Mock HTTP Servers

### 3.1 pytest-httpbin

```python
# pytest-httpbin provides a local httpbin server
import pytest

def test_basic_request(httpbin):
    """Test with local httpbin server"""
    import requests

    response = requests.get(httpbin.url + '/get')
    assert response.status_code == 200

    data = response.json()
    assert 'headers' in data

def test_post_request(httpbin):
    """Test POST handling"""
    import requests

    response = requests.post(
        httpbin.url + '/post',
        json={'key': 'value'}
    )
    assert response.status_code == 200
    assert response.json()['json'] == {'key': 'value'}

def test_status_codes(httpbin):
    """Test various status codes"""
    import requests

    for status in [200, 301, 404, 500]:
        response = requests.get(httpbin.url + f'/status/{status}')
        assert response.status_code == status

def test_delayed_response(httpbin):
    """Test handling of slow responses"""
    import requests

    response = requests.get(
        httpbin.url + '/delay/2',
        timeout=5
    )
    assert response.status_code == 200
```

### 3.2 responses Library

```python
import responses
import requests

class TestWithResponses:
    """Mock HTTP responses for testing"""

    @responses.activate
    def test_successful_scrape(self):
        """Test successful page fetch"""
        responses.add(
            responses.GET,
            'https://example.com/product/123',
            body='<html><h1>Product Name</h1><span class="price">$99</span></html>',
            status=200,
            content_type='text/html',
        )

        response = requests.get('https://example.com/product/123')
        assert response.status_code == 200
        assert 'Product Name' in response.text

    @responses.activate
    def test_retry_on_error(self):
        """Test retry behavior"""
        # First two requests fail
        responses.add(responses.GET, 'https://example.com/page', status=500)
        responses.add(responses.GET, 'https://example.com/page', status=500)
        # Third succeeds
        responses.add(
            responses.GET,
            'https://example.com/page',
            body='Success',
            status=200,
        )

        from crawler import fetch_with_retry
        result = fetch_with_retry('https://example.com/page', max_retries=3)

        assert result == 'Success'
        assert len(responses.calls) == 3

    @responses.activate
    def test_rate_limiting(self):
        """Test rate limit handling"""
        responses.add(
            responses.GET,
            'https://example.com/api',
            status=429,
            headers={'Retry-After': '1'},
        )
        responses.add(
            responses.GET,
            'https://example.com/api',
            body='{"data": "success"}',
            status=200,
        )

        from crawler import fetch_with_rate_limit_handling
        result = fetch_with_rate_limit_handling('https://example.com/api')

        assert result['data'] == 'success'

    @responses.activate
    def test_redirect_handling(self):
        """Test redirect following"""
        responses.add(
            responses.GET,
            'https://example.com/old-url',
            status=301,
            headers={'Location': 'https://example.com/new-url'},
        )
        responses.add(
            responses.GET,
            'https://example.com/new-url',
            body='Final content',
            status=200,
        )

        response = requests.get(
            'https://example.com/old-url',
            allow_redirects=True
        )

        assert response.status_code == 200
        assert response.text == 'Final content'
```

### 3.3 Custom Mock Server

```python
from flask import Flask, request, jsonify
import threading

class MockWebServer:
    """Custom mock server for complex scenarios"""

    def __init__(self, port: int = 5000):
        self.app = Flask(__name__)
        self.port = port
        self.server_thread = None
        self._setup_routes()

    def _setup_routes(self):
        """Setup mock routes"""

        @self.app.route('/product/<int:product_id>')
        def product_page(product_id):
            return f'''
            <html>
                <h1>Product {product_id}</h1>
                <span class="price">${product_id * 10}.99</span>
            </html>
            '''

        @self.app.route('/api/products')
        def api_products():
            page = request.args.get('page', 1, type=int)
            return jsonify({
                'products': [
                    {'id': i, 'name': f'Product {i}'}
                    for i in range((page-1)*10, page*10)
                ],
                'page': page,
                'has_next': page < 5,
            })

        @self.app.route('/slow-page')
        def slow_page():
            import time
            time.sleep(2)
            return '<html>Slow content</html>'

        @self.app.route('/random-error')
        def random_error():
            import random
            if random.random() < 0.3:
                return 'Server Error', 500
            return '<html>Success</html>'

        @self.app.route('/robots.txt')
        def robots():
            return '''
            User-agent: *
            Disallow: /private/
            Crawl-delay: 1
            '''

    def start(self):
        """Start server in background thread"""
        self.server_thread = threading.Thread(
            target=lambda: self.app.run(port=self.port, threaded=True)
        )
        self.server_thread.daemon = True
        self.server_thread.start()

    def stop(self):
        """Stop server"""
        # Flask doesn't have clean shutdown, handled by daemon thread
        pass

    @property
    def base_url(self) -> str:
        return f'http://localhost:{self.port}'


# Usage in tests
@pytest.fixture(scope='module')
def mock_server():
    """Provide mock server for tests"""
    server = MockWebServer(port=5001)
    server.start()
    import time
    time.sleep(0.5)  # Wait for server to start
    yield server
    server.stop()

def test_with_mock_server(mock_server):
    """Test crawler with mock server"""
    import requests

    response = requests.get(f'{mock_server.base_url}/product/42')
    assert response.status_code == 200
    assert 'Product 42' in response.text
```

---

## 4. Integration Testing

### 4.1 Scrapy Test Runner

```python
from scrapy.crawler import CrawlerProcess
from scrapy.utils.project import get_project_settings
import pytest

class TestScrapySpider:
    """Integration tests for Scrapy spiders"""

    @pytest.fixture
    def crawler_process(self):
        """Create crawler process for testing"""
        settings = get_project_settings()
        settings.update({
            'LOG_LEVEL': 'ERROR',
            'ROBOTSTXT_OBEY': False,
            'HTTPCACHE_ENABLED': False,
        })
        return CrawlerProcess(settings)

    def test_spider_output(self, crawler_process, mock_server, tmp_path):
        """Test spider produces expected output"""
        from spiders.product_spider import ProductSpider

        output_file = tmp_path / 'output.json'

        crawler_process.crawl(
            ProductSpider,
            start_urls=[f'{mock_server.base_url}/product/1'],
            output_file=str(output_file),
        )
        crawler_process.start()

        # Verify output
        import json
        with open(output_file) as f:
            data = json.load(f)

        assert len(data) > 0
        assert 'title' in data[0]
        assert 'price' in data[0]
```

### 4.2 Pipeline Testing

```python
from scrapy import Item, Field
from pipelines import ValidationPipeline, DatabasePipeline

class ProductItem(Item):
    name = Field()
    price = Field()
    url = Field()

class TestPipelines:
    """Test Scrapy pipelines"""

    @pytest.fixture
    def validation_pipeline(self):
        return ValidationPipeline()

    @pytest.fixture
    def valid_item(self):
        item = ProductItem()
        item['name'] = 'Test Product'
        item['price'] = 99.99
        item['url'] = 'https://example.com/product'
        return item

    def test_validation_passes(self, validation_pipeline, valid_item):
        """Test valid item passes validation"""
        result = validation_pipeline.process_item(valid_item, None)
        assert result is not None

    def test_validation_fails_missing_name(self, validation_pipeline):
        """Test item without name fails validation"""
        item = ProductItem()
        item['price'] = 99.99

        with pytest.raises(DropItem):
            validation_pipeline.process_item(item, None)

    def test_validation_fails_invalid_price(self, validation_pipeline):
        """Test item with invalid price fails"""
        item = ProductItem()
        item['name'] = 'Product'
        item['price'] = -10  # Invalid

        with pytest.raises(DropItem):
            validation_pipeline.process_item(item, None)
```

---

## 5. Recording and Replaying HTTP

### 5.1 VCR.py for Response Recording

```python
import vcr

class TestWithRecordedResponses:
    """Test with recorded HTTP responses"""

    @vcr.use_cassette('tests/cassettes/product_page.yaml')
    def test_parse_real_product(self):
        """Test with recorded real response"""
        import requests
        from parsers import ProductParser

        # First run: records actual response
        # Subsequent runs: replays from cassette
        response = requests.get('https://real-store.com/product/123')
        parser = ProductParser()
        result = parser.parse(response.text)

        assert result['name'] is not None
        assert result['price'] > 0

    @vcr.use_cassette(
        'tests/cassettes/api_response.yaml',
        record_mode='none'  # Don't record new requests
    )
    def test_api_parsing(self):
        """Test API response parsing"""
        import requests

        response = requests.get('https://api.example.com/products')
        data = response.json()

        assert 'products' in data


# VCR configuration
# conftest.py
@pytest.fixture(scope='module')
def vcr_config():
    return {
        'filter_headers': ['authorization', 'cookie'],
        'filter_query_parameters': ['api_key'],
        'record_mode': 'once',  # Record once, then replay
    }
```

### 5.2 Cassette Management

```python
import vcr
import os
from pathlib import Path

class CassetteManager:
    """Manage VCR cassettes for tests"""

    def __init__(self, cassette_dir: str = 'tests/cassettes'):
        self.cassette_dir = Path(cassette_dir)
        self.cassette_dir.mkdir(exist_ok=True)

    def get_cassette_path(self, name: str) -> str:
        """Get path for cassette file"""
        return str(self.cassette_dir / f'{name}.yaml')

    def clear_cassette(self, name: str):
        """Remove cassette to force re-recording"""
        path = self.cassette_dir / f'{name}.yaml'
        if path.exists():
            path.unlink()

    def clear_all(self):
        """Clear all cassettes"""
        for cassette in self.cassette_dir.glob('*.yaml'):
            cassette.unlink()

    @staticmethod
    def cassette(name: str, **kwargs):
        """Decorator for using cassettes"""
        def decorator(func):
            cassette_path = f'tests/cassettes/{name}.yaml'
            return vcr.use_cassette(cassette_path, **kwargs)(func)
        return decorator


# Usage
@CassetteManager.cassette('my_test', record_mode='new_episodes')
def test_something():
    pass
```

---

## 6. End-to-End Testing

### 6.1 Test Site Setup

```python
# Local test site with known structure
class TestSiteGenerator:
    """Generate static test site for E2E testing"""

    def __init__(self, output_dir: str):
        self.output_dir = Path(output_dir)

    def generate(self, num_products: int = 100):
        """Generate test site structure"""
        self.output_dir.mkdir(exist_ok=True)

        # Generate index
        self._generate_index(num_products)

        # Generate product pages
        for i in range(num_products):
            self._generate_product_page(i)

        # Generate robots.txt
        self._generate_robots()

        # Generate sitemap
        self._generate_sitemap(num_products)

    def _generate_product_page(self, product_id: int):
        """Generate single product page"""
        html = f'''
        <!DOCTYPE html>
        <html>
        <head>
            <title>Product {product_id}</title>
            <script type="application/ld+json">
            {{
                "@context": "https://schema.org",
                "@type": "Product",
                "name": "Test Product {product_id}",
                "offers": {{
                    "@type": "Offer",
                    "price": "{product_id * 10 + 9.99}",
                    "priceCurrency": "USD"
                }}
            }}
            </script>
        </head>
        <body>
            <h1 class="product-title">Test Product {product_id}</h1>
            <span class="price">${product_id * 10 + 9.99}</span>
            <div class="description">
                Description for product {product_id}.
                This is a test product for crawler testing.
            </div>
            <span class="stock">In Stock</span>
        </body>
        </html>
        '''

        product_dir = self.output_dir / 'products'
        product_dir.mkdir(exist_ok=True)
        (product_dir / f'{product_id}.html').write_text(html)

    def _generate_sitemap(self, num_products: int):
        """Generate sitemap.xml"""
        urls = '\n'.join([
            f'  <url><loc>http://localhost:8000/products/{i}.html</loc></url>'
            for i in range(num_products)
        ])

        sitemap = f'''<?xml version="1.0" encoding="UTF-8"?>
        <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
        {urls}
        </urlset>
        '''

        (self.output_dir / 'sitemap.xml').write_text(sitemap)
```

### 6.2 E2E Test with Real Crawl

```python
import subprocess
import time

class TestE2ECrawl:
    """End-to-end crawl tests"""

    @pytest.fixture(scope='class')
    def test_site(self, tmp_path_factory):
        """Generate and serve test site"""
        site_dir = tmp_path_factory.mktemp('test_site')

        # Generate test site
        generator = TestSiteGenerator(str(site_dir))
        generator.generate(num_products=50)

        # Start simple HTTP server
        server = subprocess.Popen(
            ['python', '-m', 'http.server', '8888'],
            cwd=site_dir,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )

        time.sleep(1)  # Wait for server
        yield 'http://localhost:8888'

        server.terminate()

    def test_full_crawl(self, test_site, tmp_path):
        """Test complete crawl workflow"""
        output_file = tmp_path / 'results.json'

        # Run crawler
        from crawler import ProductCrawler
        crawler = ProductCrawler(
            start_url=f'{test_site}/sitemap.xml',
            output_file=str(output_file),
        )
        crawler.run()

        # Verify results
        import json
        with open(output_file) as f:
            results = json.load(f)

        assert len(results) == 50
        assert all('title' in r for r in results)
        assert all('price' in r for r in results)

    def test_respects_robots(self, test_site):
        """Test crawler respects robots.txt"""
        from crawler import RobotsTxtChecker

        checker = RobotsTxtChecker(test_site)

        assert checker.can_fetch('/products/1.html')
        assert not checker.can_fetch('/private/secret.html')
```

---

## 7. Performance Testing

### 7.1 Benchmark Framework

```python
import time
from statistics import mean, stdev
from dataclasses import dataclass

@dataclass
class BenchmarkResult:
    name: str
    iterations: int
    total_time: float
    avg_time: float
    min_time: float
    max_time: float
    std_dev: float

class CrawlerBenchmark:
    """Benchmark crawler performance"""

    def __init__(self):
        self.results = []

    def benchmark(self, name: str, func, iterations: int = 10) -> BenchmarkResult:
        """Run benchmark"""
        times = []

        for _ in range(iterations):
            start = time.perf_counter()
            func()
            elapsed = time.perf_counter() - start
            times.append(elapsed)

        result = BenchmarkResult(
            name=name,
            iterations=iterations,
            total_time=sum(times),
            avg_time=mean(times),
            min_time=min(times),
            max_time=max(times),
            std_dev=stdev(times) if len(times) > 1 else 0,
        )

        self.results.append(result)
        return result

    def report(self) -> str:
        """Generate benchmark report"""
        lines = ['# Benchmark Results\n']

        for result in self.results:
            lines.append(f'## {result.name}')
            lines.append(f'- Iterations: {result.iterations}')
            lines.append(f'- Avg Time: {result.avg_time*1000:.2f}ms')
            lines.append(f'- Min/Max: {result.min_time*1000:.2f}ms / {result.max_time*1000:.2f}ms')
            lines.append(f'- Std Dev: {result.std_dev*1000:.2f}ms\n')

        return '\n'.join(lines)


# Usage in tests
def test_parser_performance():
    """Benchmark parser performance"""
    benchmark = CrawlerBenchmark()

    html = '<html>...(large HTML)...</html>'
    parser = ProductParser()

    result = benchmark.benchmark(
        'ProductParser.parse',
        lambda: parser.parse(html),
        iterations=100
    )

    # Assert performance requirements
    assert result.avg_time < 0.01  # Less than 10ms average
    assert result.max_time < 0.05  # No outliers over 50ms
```

---

## 8. CI/CD Integration

### 8.1 GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Crawler Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      redis:
        image: redis
        ports:
          - 6379:6379

    steps:
      - uses: actions/checkout@v4

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.11'

      - name: Install dependencies
        run: |
          pip install -r requirements.txt
          pip install -r requirements-test.txt
          playwright install chromium

      - name: Run unit tests
        run: pytest tests/unit -v --cov=crawler --cov-report=xml

      - name: Run integration tests
        run: pytest tests/integration -v

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.xml

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
      - run: pip install ruff mypy
      - run: ruff check .
      - run: mypy crawler/
```

---

## 9. Test Data Management

### 9.1 Fixtures Organization

```
tests/
├── conftest.py           # Shared fixtures
├── fixtures/
│   ├── html/
│   │   ├── product_page.html
│   │   ├── news_article.html
│   │   └── job_listing.html
│   ├── json/
│   │   ├── api_response.json
│   │   └── sitemap.json
│   └── cassettes/        # VCR recordings
├── unit/
│   ├── test_parsers.py
│   └── test_validators.py
├── integration/
│   ├── test_pipelines.py
│   └── test_http.py
└── e2e/
    └── test_full_crawl.py
```

---

## References

- [pytest-httpbin](https://github.com/kevin1024/pytest-httpbin)
- [responses Library](https://github.com/getsentry/responses)
- [VCR.py Documentation](https://vcrpy.readthedocs.io/)
- [Web Scraper Testing Article](https://datawookie.dev/blog/2025/01/web-scraper-testing/)

---

*Comprehensive testing ensures crawler reliability and maintainability.*
