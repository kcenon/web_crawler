# SPA & JavaScript Rendering Guide

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Strategies for crawling Single Page Applications and JavaScript-heavy websites

## Overview

Modern web applications increasingly rely on JavaScript for content rendering. Traditional HTTP-based scrapers fail because SPAs load minimal HTML initially and render everything client-side.

---

## 1. Understanding SPA Architecture

### 1.1 How SPAs Load Content

```
Traditional Website:
Server → Complete HTML → Browser displays

SPA (Single Page Application):
Server → Minimal HTML + JS bundle → JS executes →
API calls → Data received → DOM updated → Content visible
```

### 1.2 Common SPA Frameworks

| Framework | Detection Signs |
|-----------|-----------------|
| **React** | `<div id="root">`, `__REACT_DEVTOOLS_GLOBAL_HOOK__` |
| **Vue.js** | `<div id="app">`, `__VUE__`, `v-` attributes |
| **Angular** | `ng-` attributes, `angular.min.js` |
| **Next.js** | `_next/` paths, `__NEXT_DATA__` script |
| **Nuxt.js** | `_nuxt/` paths, `__NUXT__` variable |

### 1.3 SPA Content Loading Timeline

```
0ms    ─────────────────────────────────────────────────────
       │ Initial HTML loaded (nearly empty)
100ms  ─────────────────────────────────────────────────────
       │ JavaScript bundle loaded
500ms  ─────────────────────────────────────────────────────
       │ Framework initialized
       │ Initial render (skeleton/loading)
800ms  ─────────────────────────────────────────────────────
       │ API calls initiated
1500ms ─────────────────────────────────────────────────────
       │ Data received, DOM updated
       │ ✓ Content fully visible
2000ms ─────────────────────────────────────────────────────
```

---

## 2. Headless Browser Approach

### 2.1 Playwright for SPAs

```python
from playwright.async_api import async_playwright
import asyncio

class SPACrawler:
    def __init__(self):
        self.browser = None
        self.context = None

    async def start(self):
        """Initialize browser"""
        playwright = await async_playwright().start()
        self.browser = await playwright.chromium.launch(
            headless=True,
            args=[
                '--disable-gpu',
                '--disable-dev-shm-usage',
                '--no-sandbox',
            ]
        )
        self.context = await self.browser.new_context(
            viewport={'width': 1920, 'height': 1080},
            user_agent='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'
        )

    async def crawl_spa(self, url: str, wait_for: str = None) -> dict:
        """Crawl SPA page with proper waiting"""
        page = await self.context.new_page()

        try:
            # Navigate and wait for network idle
            await page.goto(url, wait_until='networkidle')

            # Wait for specific content if specified
            if wait_for:
                await page.wait_for_selector(wait_for, timeout=30000)

            # Additional wait for any lazy loading
            await self._wait_for_content_stable(page)

            return {
                'url': url,
                'html': await page.content(),
                'title': await page.title(),
            }

        finally:
            await page.close()

    async def _wait_for_content_stable(self, page, timeout: int = 5000):
        """Wait until page content stops changing"""
        previous_html = ""
        stable_count = 0
        start_time = asyncio.get_event_loop().time()

        while (asyncio.get_event_loop().time() - start_time) < timeout / 1000:
            current_html = await page.content()

            if current_html == previous_html:
                stable_count += 1
                if stable_count >= 3:  # Stable for 3 checks
                    return
            else:
                stable_count = 0
                previous_html = current_html

            await asyncio.sleep(0.5)

    async def close(self):
        """Cleanup"""
        if self.context:
            await self.context.close()
        if self.browser:
            await self.browser.close()
```

### 2.2 Handling Infinite Scroll

```python
async def crawl_infinite_scroll(self, url: str, max_scrolls: int = 10) -> list:
    """Crawl page with infinite scroll"""
    page = await self.context.new_page()
    all_items = []

    try:
        await page.goto(url, wait_until='networkidle')

        for scroll_num in range(max_scrolls):
            # Get current items
            items = await page.query_selector_all('.item-selector')
            current_count = len(items)

            # Extract data from new items
            for item in items[len(all_items):]:
                data = await self._extract_item_data(item)
                all_items.append(data)

            # Scroll to bottom
            await page.evaluate('window.scrollTo(0, document.body.scrollHeight)')

            # Wait for new content
            try:
                await page.wait_for_function(
                    f'document.querySelectorAll(".item-selector").length > {current_count}',
                    timeout=5000
                )
            except:
                # No new content loaded - reached end
                break

            await asyncio.sleep(1)  # Politeness delay

        return all_items

    finally:
        await page.close()

async def _extract_item_data(self, element) -> dict:
    """Extract data from a single item element"""
    return {
        'title': await element.query_selector('.title').inner_text(),
        'price': await element.query_selector('.price').inner_text(),
    }
```

### 2.3 Click-to-Load Content

```python
async def crawl_click_to_load(self, url: str, load_more_selector: str,
                               item_selector: str, max_clicks: int = 5) -> list:
    """Crawl page with 'Load More' button"""
    page = await self.context.new_page()
    all_items = []

    try:
        await page.goto(url, wait_until='networkidle')

        for click_num in range(max_clicks):
            # Get current count
            items = await page.query_selector_all(item_selector)
            current_count = len(items)

            # Find and click 'Load More'
            load_more = await page.query_selector(load_more_selector)

            if not load_more:
                break  # No more button

            # Check if button is visible and enabled
            is_visible = await load_more.is_visible()
            is_enabled = await load_more.is_enabled()

            if not (is_visible and is_enabled):
                break

            await load_more.click()

            # Wait for new items
            try:
                await page.wait_for_function(
                    f'document.querySelectorAll("{item_selector}").length > {current_count}',
                    timeout=10000
                )
            except:
                break

        # Extract all items
        items = await page.query_selector_all(item_selector)
        for item in items:
            data = await self._extract_item_data(item)
            all_items.append(data)

        return all_items

    finally:
        await page.close()
```

---

## 3. API Discovery Approach

### 3.1 Why API Discovery?

| Aspect | HTML Scraping | API Scraping |
|--------|---------------|--------------|
| **Data Format** | Parse HTML | Clean JSON |
| **Stability** | Breaks on UI changes | More stable |
| **Performance** | Slower (full render) | Faster (data only) |
| **Bandwidth** | Higher | Lower |
| **Complexity** | Simpler setup | Requires discovery |

### 3.2 Finding Hidden APIs

```python
from playwright.async_api import async_playwright
import json

class APIDiscovery:
    def __init__(self):
        self.captured_requests = []
        self.captured_responses = []

    async def discover_apis(self, url: str) -> list[dict]:
        """Discover API endpoints used by a page"""
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            context = await browser.new_context()
            page = await context.new_page()

            # Intercept all requests
            page.on('request', self._handle_request)
            page.on('response', self._handle_response)

            await page.goto(url, wait_until='networkidle')

            # Interact with page to trigger more API calls
            await self._trigger_interactions(page)

            await browser.close()

        return self._analyze_captured_data()

    def _handle_request(self, request):
        """Capture request details"""
        if self._is_api_request(request):
            self.captured_requests.append({
                'url': request.url,
                'method': request.method,
                'headers': dict(request.headers),
                'post_data': request.post_data,
            })

    async def _handle_response(self, response):
        """Capture response details"""
        request = response.request

        if self._is_api_request(request):
            try:
                body = await response.json()
            except:
                body = await response.text()

            self.captured_responses.append({
                'url': request.url,
                'status': response.status,
                'headers': dict(response.headers),
                'body': body,
            })

    def _is_api_request(self, request) -> bool:
        """Determine if request is likely an API call"""
        url = request.url.lower()
        content_type = request.headers.get('accept', '')

        # Check for API indicators
        api_patterns = ['/api/', '/graphql', '/v1/', '/v2/', '/rest/']
        if any(pattern in url for pattern in api_patterns):
            return True

        # Check for JSON content type
        if 'application/json' in content_type:
            return True

        # Check for XHR/Fetch
        if request.resource_type in ['xhr', 'fetch']:
            return True

        return False

    async def _trigger_interactions(self, page):
        """Interact with page to trigger API calls"""
        # Scroll to trigger lazy loading
        await page.evaluate('window.scrollTo(0, document.body.scrollHeight)')
        await asyncio.sleep(1)

        # Click common interactive elements
        for selector in ['.load-more', '.show-more', '[data-load]']:
            try:
                element = await page.query_selector(selector)
                if element:
                    await element.click()
                    await asyncio.sleep(0.5)
            except:
                pass

    def _analyze_captured_data(self) -> list[dict]:
        """Analyze and return useful API endpoints"""
        apis = []

        for req, resp in zip(self.captured_requests, self.captured_responses):
            if resp['status'] == 200:
                apis.append({
                    'endpoint': req['url'],
                    'method': req['method'],
                    'request_headers': req['headers'],
                    'response_sample': resp['body'][:1000] if isinstance(resp['body'], str)
                                       else resp['body'],
                })

        return apis
```

### 3.3 Replicating API Calls

```python
import requests
from urllib.parse import urlparse, parse_qs

class APIReplicator:
    def __init__(self, discovered_api: dict):
        self.endpoint = discovered_api['endpoint']
        self.method = discovered_api['method']
        self.headers = self._clean_headers(discovered_api['request_headers'])

    def _clean_headers(self, headers: dict) -> dict:
        """Remove browser-specific headers"""
        skip_headers = [
            'accept-encoding', 'accept-language', 'sec-',
            'upgrade-insecure-requests', 'cookie'
        ]

        return {
            k: v for k, v in headers.items()
            if not any(k.lower().startswith(skip) for skip in skip_headers)
        }

    def fetch_data(self, params: dict = None, page: int = 1) -> dict:
        """Fetch data from API directly"""
        url = self.endpoint

        # Handle pagination
        if '?' in url:
            url += f'&page={page}'
        else:
            url += f'?page={page}'

        if params:
            for key, value in params.items():
                url += f'&{key}={value}'

        response = requests.request(
            method=self.method,
            url=url,
            headers=self.headers,
        )

        return response.json()

    def fetch_all_pages(self, max_pages: int = 100) -> list:
        """Fetch all pages of data"""
        all_data = []

        for page in range(1, max_pages + 1):
            data = self.fetch_data(page=page)

            # Detect end of data (varies by API)
            if not data or (isinstance(data, dict) and not data.get('results')):
                break

            if isinstance(data, list):
                all_data.extend(data)
            elif isinstance(data, dict) and 'results' in data:
                all_data.extend(data['results'])
            else:
                all_data.append(data)

        return all_data
```

---

## 4. Handling Specific SPA Patterns

### 4.1 React Hydration Detection

```python
async def wait_for_react_hydration(page):
    """Wait for React app to fully hydrate"""
    await page.wait_for_function('''
        () => {
            // Check if React DevTools hook exists
            if (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) {
                const fiberRoots = window.__REACT_DEVTOOLS_GLOBAL_HOOK__.getFiberRoots(1);
                if (fiberRoots && fiberRoots.size > 0) {
                    return true;
                }
            }

            // Fallback: check for hydration markers
            const root = document.getElementById('root') || document.getElementById('__next');
            if (root && root.children.length > 0) {
                return !root.innerHTML.includes('loading');
            }

            return false;
        }
    ''', timeout=30000)
```

### 4.2 Vue.js Detection

```python
async def wait_for_vue_mount(page):
    """Wait for Vue app to mount"""
    await page.wait_for_function('''
        () => {
            // Check for Vue 3
            if (window.__VUE__) {
                return true;
            }

            // Check for Vue 2
            const app = document.getElementById('app');
            if (app && app.__vue__) {
                return true;
            }

            // Check for mounted components
            const vueComponents = document.querySelectorAll('[data-v-]');
            return vueComponents.length > 0;
        }
    ''', timeout=30000)
```

### 4.3 Next.js Data Extraction

```python
async def extract_nextjs_data(page) -> dict:
    """Extract data from Next.js __NEXT_DATA__ script"""
    next_data = await page.evaluate('''
        () => {
            const script = document.getElementById('__NEXT_DATA__');
            if (script) {
                return JSON.parse(script.textContent);
            }
            return null;
        }
    ''')

    if next_data:
        return {
            'props': next_data.get('props', {}),
            'page': next_data.get('page'),
            'query': next_data.get('query'),
        }

    return {}
```

### 4.4 GraphQL Endpoint Discovery

```python
async def discover_graphql(page, url: str) -> dict | None:
    """Discover and analyze GraphQL endpoint"""
    graphql_requests = []

    def capture_graphql(request):
        if 'graphql' in request.url.lower():
            graphql_requests.append({
                'url': request.url,
                'query': request.post_data,
            })

    page.on('request', capture_graphql)
    await page.goto(url, wait_until='networkidle')

    if graphql_requests:
        return {
            'endpoint': graphql_requests[0]['url'],
            'sample_queries': [r['query'] for r in graphql_requests],
        }

    return None
```

---

## 5. Performance Optimization

### 5.1 Resource Blocking

```python
async def create_optimized_page(context):
    """Create page with unnecessary resources blocked"""
    page = await context.new_page()

    # Block heavy resources
    await page.route('**/*', lambda route: (
        route.abort() if route.request.resource_type in [
            'image', 'media', 'font', 'stylesheet'
        ] else route.continue_()
    ))

    # Block tracking scripts
    await page.route('**/*analytics*', lambda route: route.abort())
    await page.route('**/*tracking*', lambda route: route.abort())
    await page.route('**/*facebook*', lambda route: route.abort())
    await page.route('**/*google-analytics*', lambda route: route.abort())

    return page
```

### 5.2 Selective Rendering

```python
async def selective_render(url: str, wait_selectors: list[str]) -> str:
    """Only wait for specific content, not full page"""
    page = await context.new_page()

    # Don't wait for network idle
    await page.goto(url, wait_until='domcontentloaded')

    # Wait only for required selectors
    for selector in wait_selectors:
        try:
            await page.wait_for_selector(selector, timeout=10000)
        except:
            pass  # Continue if selector not found

    return await page.content()
```

### 5.3 Browser Pool

```python
import asyncio
from contextlib import asynccontextmanager

class BrowserPool:
    def __init__(self, size: int = 5):
        self.size = size
        self.browsers = []
        self.available = asyncio.Queue()
        self.playwright = None

    async def start(self):
        """Initialize browser pool"""
        from playwright.async_api import async_playwright
        self.playwright = await async_playwright().start()

        for _ in range(self.size):
            browser = await self.playwright.chromium.launch(headless=True)
            context = await browser.new_context()
            self.browsers.append(browser)
            await self.available.put(context)

    @asynccontextmanager
    async def get_context(self):
        """Get browser context from pool"""
        context = await self.available.get()
        try:
            yield context
        finally:
            await self.available.put(context)

    async def close(self):
        """Cleanup all browsers"""
        for browser in self.browsers:
            await browser.close()
        await self.playwright.stop()

# Usage
pool = BrowserPool(size=10)
await pool.start()

async with pool.get_context() as context:
    page = await context.new_page()
    await page.goto(url)
    # ... scrape
    await page.close()
```

---

## 6. Troubleshooting

### 6.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Empty content | JS not executed | Use headless browser |
| Missing data | Async loading | Proper wait strategies |
| Different content | Geolocation/cookies | Set locale, clear cookies |
| Blocked | Bot detection | See anti-bot evasion guide |
| Slow | Full page render | Block resources, use API |

### 6.2 Debug Mode

```python
async def debug_spa_crawl(url: str):
    """Debug SPA crawling issues"""
    async with async_playwright() as p:
        browser = await p.chromium.launch(
            headless=False,  # Show browser
            devtools=True,   # Open DevTools
            slow_mo=1000,    # Slow down actions
        )

        page = await browser.new_page()

        # Log all console messages
        page.on('console', lambda msg: print(f'Console: {msg.text}'))

        # Log all network errors
        page.on('pageerror', lambda err: print(f'Error: {err}'))

        await page.goto(url)

        # Pause for manual inspection
        await page.pause()

        await browser.close()
```

---

## 7. Decision Framework

### When to Use Each Approach

```
Start
  │
  ▼
Is content in initial HTML?
  │
  ├─ Yes → Use HTTP scraping (requests + BeautifulSoup)
  │
  └─ No → Does site have API?
            │
            ├─ Yes, documented → Use official API
            │
            ├─ Yes, undocumented → Discover and replicate API calls
            │
            └─ No → Use headless browser (Playwright)
                     │
                     ├─ Simple SPA → wait_for_selector + content()
                     │
                     ├─ Infinite scroll → scroll + wait pattern
                     │
                     └─ Complex interactions → Full automation
```

---

## References

- [Best Way to Scrape SPAs](https://www.firecrawl.dev/glossary/web-scraping-apis/best-way-to-scrape-single-page-applications-spas)
- [Scraping Single Page Applications](https://www.scrapingbee.com/blog/scraping-single-page-applications/)
- [How to Reverse Engineer APIs](https://blog.apify.com/reverse-engineer-apis/)
- [Scraping JS-Heavy Websites 2025](https://brightdata.com/blog/web-data/scraping-js-heavy-websites)

---

*Choose the simplest approach that works: HTTP scraping > API calls > Headless browser*
