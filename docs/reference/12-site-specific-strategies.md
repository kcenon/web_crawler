# Site-Specific Crawling Strategies

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Practical patterns and strategies for crawling common website types

## Overview

Different website types have distinct structures, anti-bot measures, and data patterns. This guide provides targeted strategies for common site categories.

---

## 1. E-Commerce Sites

### 1.1 Common Patterns

| Element | Common Selectors |
|---------|------------------|
| **Product Title** | `h1`, `.product-title`, `[itemprop="name"]` |
| **Price** | `.price`, `[itemprop="price"]`, `.product-price` |
| **Images** | `.product-image img`, `[itemprop="image"]` |
| **Description** | `.description`, `[itemprop="description"]` |
| **Reviews** | `.review`, `.customer-review`, `[itemprop="review"]` |
| **Stock Status** | `.stock`, `.availability`, `[itemprop="availability"]` |

### 1.2 E-Commerce Crawler Pattern

```python
from dataclasses import dataclass
from typing import Optional
from decimal import Decimal
import scrapy

@dataclass
class Product:
    url: str
    name: str
    price: Optional[Decimal]
    currency: str
    image_url: Optional[str]
    description: Optional[str]
    in_stock: bool
    rating: Optional[float]
    review_count: Optional[int]
    sku: Optional[str]
    brand: Optional[str]
    category: Optional[str]

class EcommerceSpider(scrapy.Spider):
    """Base spider for e-commerce sites"""

    name = 'ecommerce'

    # Override in subclass
    product_list_selector = '.product-list .product'
    product_link_selector = 'a.product-link::attr(href)'
    next_page_selector = '.pagination .next::attr(href)'

    def parse(self, response):
        """Parse product listing page"""
        # Extract product links
        for product in response.css(self.product_list_selector):
            link = product.css(self.product_link_selector).get()
            if link:
                yield response.follow(link, self.parse_product)

        # Follow pagination
        next_page = response.css(self.next_page_selector).get()
        if next_page:
            yield response.follow(next_page, self.parse)

    def parse_product(self, response):
        """Parse individual product page"""
        yield Product(
            url=response.url,
            name=self.extract_name(response),
            price=self.extract_price(response),
            currency=self.extract_currency(response),
            image_url=self.extract_image(response),
            description=self.extract_description(response),
            in_stock=self.extract_stock_status(response),
            rating=self.extract_rating(response),
            review_count=self.extract_review_count(response),
            sku=self.extract_sku(response),
            brand=self.extract_brand(response),
            category=self.extract_category(response),
        )

    def extract_name(self, response) -> str:
        selectors = [
            'h1.product-title::text',
            '[itemprop="name"]::text',
            'h1::text',
        ]
        return self._try_selectors(response, selectors, '')

    def extract_price(self, response) -> Optional[Decimal]:
        selectors = [
            '[itemprop="price"]::attr(content)',
            '.price .amount::text',
            '.product-price::text',
        ]
        price_text = self._try_selectors(response, selectors)
        if price_text:
            import re
            match = re.search(r'[\d,.]+', price_text.replace(',', ''))
            if match:
                return Decimal(match.group())
        return None

    def extract_stock_status(self, response) -> bool:
        # Check for out of stock indicators
        out_of_stock_indicators = [
            '.out-of-stock',
            '.sold-out',
            '[itemprop="availability"][href*="OutOfStock"]',
        ]

        for selector in out_of_stock_indicators:
            if response.css(selector):
                return False

        return True

    def _try_selectors(self, response, selectors: list, default=None):
        """Try multiple selectors, return first match"""
        for selector in selectors:
            value = response.css(selector).get()
            if value:
                return value.strip()
        return default
```

### 1.3 Shopify Sites

```python
class ShopifySpider(scrapy.Spider):
    """Optimized spider for Shopify stores"""

    name = 'shopify'

    def start_requests(self):
        # Shopify provides JSON API
        yield scrapy.Request(
            f'{self.base_url}/products.json',
            callback=self.parse_products_json
        )

    def parse_products_json(self, response):
        """Parse Shopify products JSON API"""
        data = response.json()

        for product in data.get('products', []):
            yield {
                'id': product['id'],
                'title': product['title'],
                'handle': product['handle'],
                'vendor': product['vendor'],
                'product_type': product['product_type'],
                'tags': product['tags'],
                'variants': [
                    {
                        'id': v['id'],
                        'title': v['title'],
                        'price': v['price'],
                        'sku': v['sku'],
                        'available': v['available'],
                    }
                    for v in product['variants']
                ],
                'images': [img['src'] for img in product['images']],
            }

        # Handle pagination
        # Shopify limits to 250 products per page
        if len(data.get('products', [])) == 250:
            current_page = response.meta.get('page', 1)
            yield scrapy.Request(
                f'{self.base_url}/products.json?page={current_page + 1}',
                callback=self.parse_products_json,
                meta={'page': current_page + 1}
            )
```

### 1.4 Price Monitoring Pattern

```python
import hashlib
from datetime import datetime

class PriceMonitor:
    """Monitor price changes across e-commerce sites"""

    def __init__(self, db_manager):
        self.db = db_manager

    async def check_price(self, product: dict) -> dict:
        """Check for price changes"""
        product_id = self._generate_id(product['url'])

        # Get last known price
        last_record = await self.db.get_latest_price(product_id)

        result = {
            'product_id': product_id,
            'url': product['url'],
            'current_price': product['price'],
            'timestamp': datetime.utcnow(),
            'changed': False,
        }

        if last_record:
            if product['price'] != last_record['price']:
                result['changed'] = True
                result['previous_price'] = last_record['price']
                result['change_amount'] = product['price'] - last_record['price']
                result['change_percent'] = (
                    (product['price'] - last_record['price']) /
                    last_record['price'] * 100
                )

        # Store current price
        await self.db.store_price(product_id, product)

        return result

    def _generate_id(self, url: str) -> str:
        """Generate consistent product ID from URL"""
        return hashlib.sha256(url.encode()).hexdigest()[:16]
```

---

## 2. News Sites

### 2.1 News Article Pattern

```python
from dataclasses import dataclass
from datetime import datetime
from typing import Optional, List

@dataclass
class NewsArticle:
    url: str
    title: str
    content: str
    author: Optional[str]
    published_date: Optional[datetime]
    modified_date: Optional[datetime]
    category: Optional[str]
    tags: List[str]
    image_url: Optional[str]
    source: str

class NewsSpider(scrapy.Spider):
    """Spider for news websites"""

    name = 'news'

    # Common news article selectors
    ARTICLE_SELECTORS = {
        'title': [
            'h1.article-title',
            'h1.entry-title',
            '[itemprop="headline"]',
            'article h1',
        ],
        'content': [
            'article .content',
            '.article-body',
            '[itemprop="articleBody"]',
            '.entry-content',
        ],
        'author': [
            '[rel="author"]::text',
            '.author-name::text',
            '[itemprop="author"]::text',
        ],
        'date': [
            '[itemprop="datePublished"]::attr(content)',
            'time[datetime]::attr(datetime)',
            '.article-date::text',
        ],
    }

    def parse_article(self, response):
        """Parse news article page"""
        return NewsArticle(
            url=response.url,
            title=self._extract_with_selectors(response, 'title'),
            content=self._extract_content(response),
            author=self._extract_with_selectors(response, 'author'),
            published_date=self._parse_date(
                self._extract_with_selectors(response, 'date')
            ),
            modified_date=None,
            category=self._extract_category(response),
            tags=self._extract_tags(response),
            image_url=self._extract_main_image(response),
            source=self.name,
        )

    def _extract_content(self, response) -> str:
        """Extract and clean article content"""
        for selector in self.ARTICLE_SELECTORS['content']:
            content_elem = response.css(selector)
            if content_elem:
                # Remove unwanted elements
                for unwanted in ['script', 'style', '.ad', '.related']:
                    content_elem.css(unwanted).drop()

                paragraphs = content_elem.css('p::text').getall()
                return '\n\n'.join(p.strip() for p in paragraphs if p.strip())

        return ''

    def _extract_tags(self, response) -> List[str]:
        """Extract article tags"""
        selectors = [
            '.tags a::text',
            '[rel="tag"]::text',
            '.article-tags a::text',
        ]

        for selector in selectors:
            tags = response.css(selector).getall()
            if tags:
                return [t.strip() for t in tags]

        return []
```

### 2.2 RSS Feed Processing

```python
import feedparser
from datetime import datetime

class RSSCrawler:
    """Crawl news via RSS feeds"""

    def __init__(self):
        self.feeds = []

    def add_feed(self, url: str, source_name: str):
        """Add RSS feed to monitor"""
        self.feeds.append({'url': url, 'source': source_name})

    def fetch_all(self) -> List[dict]:
        """Fetch all feeds"""
        articles = []

        for feed_info in self.feeds:
            feed = feedparser.parse(feed_info['url'])

            for entry in feed.entries:
                articles.append({
                    'source': feed_info['source'],
                    'title': entry.get('title', ''),
                    'url': entry.get('link', ''),
                    'summary': entry.get('summary', ''),
                    'published': self._parse_date(entry.get('published')),
                    'author': entry.get('author', ''),
                    'tags': [t.get('term', '') for t in entry.get('tags', [])],
                })

        return articles

    def _parse_date(self, date_str: str) -> Optional[datetime]:
        """Parse RSS date format"""
        if not date_str:
            return None

        from email.utils import parsedate_to_datetime
        try:
            return parsedate_to_datetime(date_str)
        except:
            return None
```

---

## 3. Social Media Platforms

### 3.1 Important Considerations

> ⚠️ **Legal Warning**: Social media platforms have strict Terms of Service regarding scraping. Always:
> - Check platform TOS before scraping
> - Use official APIs when available
> - Respect rate limits
> - Avoid collecting personal data without consent

### 3.2 Public Profile Pattern (Educational)

```python
class SocialMediaScraper:
    """Educational example - always use official APIs when available"""

    async def get_public_profile(self, url: str) -> dict:
        """Scrape publicly visible profile information"""
        async with async_playwright() as p:
            browser = await p.chromium.launch()
            page = await browser.new_page()

            # Set realistic viewport and user agent
            await page.set_viewport_size({'width': 1920, 'height': 1080})

            await page.goto(url, wait_until='networkidle')

            # Only extract publicly visible information
            data = await page.evaluate('''
                () => {
                    return {
                        // Only public data
                        displayName: document.querySelector('[data-testid="name"]')?.textContent,
                        bio: document.querySelector('[data-testid="bio"]')?.textContent,
                        // Avoid personal identifiers
                    }
                }
            ''')

            await browser.close()
            return data
```

---

## 4. Job Boards

### 4.1 Job Listing Pattern

```python
@dataclass
class JobListing:
    url: str
    title: str
    company: str
    location: str
    salary_min: Optional[int]
    salary_max: Optional[int]
    job_type: str  # full-time, part-time, contract
    description: str
    requirements: List[str]
    posted_date: Optional[datetime]
    source: str

class JobBoardSpider(scrapy.Spider):
    """Spider for job listing sites"""

    name = 'jobs'

    def parse_job(self, response):
        """Parse job listing page"""
        return JobListing(
            url=response.url,
            title=response.css('h1.job-title::text').get('').strip(),
            company=response.css('.company-name::text').get('').strip(),
            location=response.css('.job-location::text').get('').strip(),
            salary_min=self._extract_salary_min(response),
            salary_max=self._extract_salary_max(response),
            job_type=self._extract_job_type(response),
            description=self._extract_description(response),
            requirements=self._extract_requirements(response),
            posted_date=self._extract_date(response),
            source=self.name,
        )

    def _extract_salary_min(self, response) -> Optional[int]:
        """Extract minimum salary"""
        salary_text = response.css('.salary::text').get('')
        # Parse salary range like "$80,000 - $120,000"
        import re
        match = re.search(r'\$?([\d,]+)', salary_text)
        if match:
            return int(match.group(1).replace(',', ''))
        return None

    def _extract_requirements(self, response) -> List[str]:
        """Extract job requirements"""
        requirements = []

        # Look for bullet points in requirements section
        req_section = response.css('.requirements, .qualifications')
        if req_section:
            items = req_section.css('li::text').getall()
            requirements = [item.strip() for item in items if item.strip()]

        return requirements
```

---

## 5. Real Estate Sites

### 5.1 Property Listing Pattern

```python
@dataclass
class PropertyListing:
    url: str
    title: str
    price: Decimal
    currency: str
    property_type: str  # apartment, house, condo
    bedrooms: Optional[int]
    bathrooms: Optional[float]
    area_sqft: Optional[float]
    address: str
    city: str
    state: str
    zip_code: str
    latitude: Optional[float]
    longitude: Optional[float]
    description: str
    features: List[str]
    images: List[str]
    agent_name: Optional[str]
    agent_phone: Optional[str]
    listed_date: Optional[datetime]

class RealEstateSpider(scrapy.Spider):
    """Spider for real estate listings"""

    name = 'realestate'

    def parse_property(self, response):
        """Parse property listing"""
        # Extract JSON-LD structured data if available
        json_ld = self._extract_json_ld(response)

        if json_ld and json_ld.get('@type') == 'RealEstateListing':
            return self._parse_from_json_ld(json_ld, response)

        # Fallback to HTML parsing
        return self._parse_from_html(response)

    def _extract_json_ld(self, response) -> Optional[dict]:
        """Extract JSON-LD structured data"""
        import json

        scripts = response.css('script[type="application/ld+json"]::text').getall()
        for script in scripts:
            try:
                data = json.loads(script)
                if isinstance(data, list):
                    for item in data:
                        if item.get('@type') in ['RealEstateListing', 'Product']:
                            return item
                elif data.get('@type') in ['RealEstateListing', 'Product']:
                    return data
            except json.JSONDecodeError:
                continue

        return None

    def _parse_from_json_ld(self, data: dict, response) -> PropertyListing:
        """Parse from JSON-LD structured data"""
        return PropertyListing(
            url=response.url,
            title=data.get('name', ''),
            price=Decimal(str(data.get('offers', {}).get('price', 0))),
            currency=data.get('offers', {}).get('priceCurrency', 'USD'),
            property_type=data.get('additionalType', ''),
            bedrooms=data.get('numberOfRooms'),
            bathrooms=None,
            area_sqft=data.get('floorSize', {}).get('value'),
            address=data.get('address', {}).get('streetAddress', ''),
            city=data.get('address', {}).get('addressLocality', ''),
            state=data.get('address', {}).get('addressRegion', ''),
            zip_code=data.get('address', {}).get('postalCode', ''),
            latitude=data.get('geo', {}).get('latitude'),
            longitude=data.get('geo', {}).get('longitude'),
            description=data.get('description', ''),
            features=[],
            images=[img.get('url') for img in data.get('image', [])],
            agent_name=None,
            agent_phone=None,
            listed_date=None,
        )
```

---

## 6. Anti-Bot Evasion by Site Type

### 6.1 Detection Levels by Category

| Site Type | Detection Level | Common Measures |
|-----------|-----------------|-----------------|
| **E-commerce** | High | Cloudflare, rate limits, CAPTCHA |
| **News** | Low-Medium | Rate limits, subscription walls |
| **Social Media** | Very High | Fingerprinting, behavior analysis |
| **Job Boards** | Medium | Rate limits, CAPTCHA |
| **Real Estate** | Medium | Rate limits, geo-blocking |

### 6.2 Site-Specific Evasion

```python
class SiteSpecificEvasion:
    """Evasion techniques by site type"""

    STRATEGIES = {
        'ecommerce': {
            'delay': (3, 7),  # Random delay range
            'headers_rotate': True,
            'proxy_type': 'residential',
            'js_render': True,
            'session_duration': 300,  # 5 min sessions
        },
        'news': {
            'delay': (1, 3),
            'headers_rotate': False,
            'proxy_type': 'datacenter',
            'js_render': False,
            'session_duration': 600,
        },
        'social': {
            'delay': (5, 15),
            'headers_rotate': True,
            'proxy_type': 'residential',
            'js_render': True,
            'session_duration': 180,
            'fingerprint_rotate': True,
        },
    }

    def get_config(self, site_type: str) -> dict:
        """Get crawling configuration for site type"""
        return self.STRATEGIES.get(site_type, self.STRATEGIES['news'])

    async def create_session(self, site_type: str):
        """Create session with site-appropriate settings"""
        config = self.get_config(site_type)

        if config['js_render']:
            return await self._create_browser_session(config)
        else:
            return self._create_http_session(config)
```

---

## 7. Structured Data Extraction

### 7.1 Schema.org Extractor

```python
import json
import extruct

class StructuredDataExtractor:
    """Extract structured data from web pages"""

    def extract_all(self, html: str, url: str) -> dict:
        """Extract all structured data formats"""
        data = extruct.extract(
            html,
            base_url=url,
            syntaxes=['json-ld', 'microdata', 'opengraph', 'microformat']
        )

        return {
            'json_ld': data.get('json-ld', []),
            'microdata': data.get('microdata', []),
            'opengraph': data.get('opengraph', []),
            'microformat': data.get('microformat', []),
        }

    def extract_products(self, html: str, url: str) -> List[dict]:
        """Extract product structured data"""
        data = self.extract_all(html, url)
        products = []

        # From JSON-LD
        for item in data['json_ld']:
            if item.get('@type') == 'Product':
                products.append(self._normalize_product(item))

        # From Microdata
        for item in data['microdata']:
            if item.get('type') == 'http://schema.org/Product':
                products.append(self._normalize_product(item['properties']))

        return products

    def _normalize_product(self, data: dict) -> dict:
        """Normalize product data to common format"""
        offers = data.get('offers', {})
        if isinstance(offers, list):
            offers = offers[0] if offers else {}

        return {
            'name': data.get('name'),
            'description': data.get('description'),
            'sku': data.get('sku'),
            'brand': data.get('brand', {}).get('name') if isinstance(data.get('brand'), dict) else data.get('brand'),
            'price': offers.get('price'),
            'currency': offers.get('priceCurrency'),
            'availability': offers.get('availability'),
            'image': data.get('image'),
            'rating': data.get('aggregateRating', {}).get('ratingValue'),
            'review_count': data.get('aggregateRating', {}).get('reviewCount'),
        }
```

---

## 8. Quick Reference

### Site Type Decision Matrix

```
What are you scraping?
│
├── Products/Prices
│   ├── Shopify → Use /products.json API
│   ├── WooCommerce → Check for REST API
│   └── Custom → Use structured data + HTML
│
├── News/Articles
│   ├── Has RSS? → Use RSS first
│   ├── Has API? → Use API
│   └── HTML only → Parse article selectors
│
├── Job Listings
│   ├── Has API? → Use API
│   └── HTML → Parse job card pattern
│
└── Real Estate
    ├── Check JSON-LD first
    └── Fall back to HTML parsing
```

---

## References

- [E-commerce Web Scraping Guide 2025](https://webdata-scraping.com/e-commerce-web-scraping-guide/)
- [Web Scraping for E-Commerce 2025 Trends](https://crawlbase.com/blog/2025-web-scraping-for-ecommerce/)
- [Bypassing Modern Bot Detection](https://medium.com/@sohail_saifii/web-scraping-in-2025-bypassing-modern-bot-detection-fcab286b117d)
- [Schema.org Product Markup](https://schema.org/Product)

---

*Adapt strategies to specific sites - no one-size-fits-all approach exists.*
