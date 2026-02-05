# Data Storage for Web Crawling

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Database selection and data management strategies for crawled data

## Overview

Choosing the right storage solution is critical for crawler performance, scalability, and data analysis. This guide covers database options, schema design, and best practices.

---

## 1. Database Selection Guide

### 1.1 Comparison Matrix

| Feature | PostgreSQL | MongoDB | Redis | SQLite |
|---------|------------|---------|-------|--------|
| **Data Type** | Structured | Semi-structured | Key-Value | Structured |
| **Scale** | Large | Very Large | Medium | Small |
| **Query Power** | Excellent | Good | Basic | Good |
| **Concurrent Writes** | Excellent | Good | Excellent | Poor |
| **Learning Curve** | Medium | Low | Low | Low |
| **Best For** | Complex analysis | Flexible schemas | Caching, queues | Prototypes |

### 1.2 When to Use Each

#### PostgreSQL

**Best for**:
- Structured data with clear relationships
- Complex queries and aggregations
- ACID compliance requirements
- Time-series data (with TimescaleDB)
- Production systems requiring reliability

```sql
-- Example: E-commerce product tracking
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    name TEXT,
    price DECIMAL(10,2),
    currency VARCHAR(3),
    in_stock BOOLEAN,
    crawled_at TIMESTAMP DEFAULT NOW(),
    source_site VARCHAR(100)
);

CREATE INDEX idx_products_site_date ON products(source_site, crawled_at);
```

#### MongoDB

**Best for**:
- Varying document structures
- Rapid schema evolution
- JSON/document storage
- Horizontal scaling needs

```javascript
// Example: News article collection
{
  "_id": ObjectId("..."),
  "url": "https://news.example.com/article/123",
  "title": "Article Title",
  "content": "Full article text...",
  "metadata": {
    "author": "John Doe",
    "published": ISODate("2026-02-05"),
    "tags": ["technology", "ai"],
    "images": [
      {"url": "...", "caption": "..."}
    ]
  },
  "crawl_info": {
    "crawled_at": ISODate("..."),
    "response_time_ms": 234,
    "status_code": 200
  }
}
```

#### Redis

**Best for**:
- URL deduplication
- Crawl queue management
- Rate limiting state
- Session caching
- NOT for primary data storage

```python
# Redis usage patterns for crawling
redis_client.sadd('visited_urls', url_hash)  # Deduplication
redis_client.lpush('crawl_queue', url)        # Queue
redis_client.setex(f'rate:{domain}', 1, timestamp)  # Rate limiting
```

---

## 2. PostgreSQL Implementation

### 2.1 Schema Design

```sql
-- Core crawl tracking
CREATE TABLE crawl_sessions (
    id SERIAL PRIMARY KEY,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'running',
    config JSONB,
    stats JSONB
);

CREATE TABLE crawled_pages (
    id SERIAL PRIMARY KEY,
    session_id INTEGER REFERENCES crawl_sessions(id),
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,  -- For deduplication
    domain VARCHAR(255) NOT NULL,
    status_code INTEGER,
    content_type VARCHAR(100),
    content_length INTEGER,
    response_time_ms INTEGER,
    crawled_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(url_hash)
);

CREATE INDEX idx_pages_domain ON crawled_pages(domain);
CREATE INDEX idx_pages_session ON crawled_pages(session_id);
CREATE INDEX idx_pages_crawled_at ON crawled_pages(crawled_at);

-- Extracted data (example: products)
CREATE TABLE extracted_products (
    id SERIAL PRIMARY KEY,
    page_id INTEGER REFERENCES crawled_pages(id),
    name TEXT,
    price DECIMAL(10,2),
    currency VARCHAR(3),
    description TEXT,
    image_urls TEXT[],
    attributes JSONB,
    extracted_at TIMESTAMP DEFAULT NOW()
);
```

### 2.2 Python Integration with SQLAlchemy

```python
from sqlalchemy import create_engine, Column, Integer, String, DateTime, JSON
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import sessionmaker
from datetime import datetime
import hashlib

Base = declarative_base()

class CrawledPage(Base):
    __tablename__ = 'crawled_pages'

    id = Column(Integer, primary_key=True)
    url = Column(String, nullable=False)
    url_hash = Column(String(64), unique=True, nullable=False)
    domain = Column(String(255), nullable=False)
    status_code = Column(Integer)
    content_type = Column(String(100))
    response_time_ms = Column(Integer)
    crawled_at = Column(DateTime, default=datetime.utcnow)

    @staticmethod
    def hash_url(url: str) -> str:
        return hashlib.sha256(url.encode()).hexdigest()

class DatabaseManager:
    def __init__(self, connection_string: str):
        self.engine = create_engine(
            connection_string,
            pool_size=20,
            max_overflow=10,
            pool_pre_ping=True,  # Check connection health
        )
        Base.metadata.create_all(self.engine)
        self.Session = sessionmaker(bind=self.engine)

    def save_page(self, url: str, status_code: int,
                  content_type: str, response_time: int):
        """Save crawled page to database"""
        session = self.Session()
        try:
            page = CrawledPage(
                url=url,
                url_hash=CrawledPage.hash_url(url),
                domain=urlparse(url).netloc,
                status_code=status_code,
                content_type=content_type,
                response_time_ms=response_time,
            )
            session.merge(page)  # Upsert
            session.commit()
        except Exception as e:
            session.rollback()
            raise
        finally:
            session.close()

    def batch_save(self, pages: list[dict]):
        """Batch insert for performance"""
        session = self.Session()
        try:
            for page_data in pages:
                page = CrawledPage(**page_data)
                session.merge(page)
            session.commit()
        except Exception as e:
            session.rollback()
            raise
        finally:
            session.close()
```

### 2.3 Time-Series with TimescaleDB

```sql
-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Price history table
CREATE TABLE price_history (
    time TIMESTAMPTZ NOT NULL,
    product_id INTEGER NOT NULL,
    price DECIMAL(10,2),
    currency VARCHAR(3),
    in_stock BOOLEAN
);

-- Convert to hypertable
SELECT create_hypertable('price_history', 'time');

-- Efficient time-based queries
SELECT time_bucket('1 day', time) AS day,
       product_id,
       AVG(price) as avg_price,
       MIN(price) as min_price,
       MAX(price) as max_price
FROM price_history
WHERE time > NOW() - INTERVAL '30 days'
GROUP BY day, product_id
ORDER BY day;
```

---

## 3. MongoDB Implementation

### 3.1 Collection Design

```javascript
// Connection setup
const { MongoClient } = require('mongodb');
const client = new MongoClient('mongodb://localhost:27017');
const db = client.db('crawler');

// Collections
const pages = db.collection('crawled_pages');
const products = db.collection('products');

// Indexes for performance
pages.createIndex({ "url": 1 }, { unique: true });
pages.createIndex({ "domain": 1, "crawled_at": -1 });
products.createIndex({ "source_url": 1 });
products.createIndex({ "name": "text", "description": "text" });
```

### 3.2 Python Integration with PyMongo

```python
from pymongo import MongoClient, UpdateOne
from pymongo.errors import BulkWriteError
from datetime import datetime
from typing import Optional

class MongoDBManager:
    def __init__(self, connection_string: str = 'mongodb://localhost:27017'):
        self.client = MongoClient(connection_string)
        self.db = self.client['crawler']
        self.pages = self.db['crawled_pages']
        self.products = self.db['products']

        # Ensure indexes
        self.pages.create_index('url', unique=True)
        self.pages.create_index([('domain', 1), ('crawled_at', -1)])

    def save_page(self, url: str, content: dict, metadata: dict):
        """Save page with upsert"""
        document = {
            'url': url,
            'content': content,
            'metadata': metadata,
            'crawled_at': datetime.utcnow(),
            'updated_at': datetime.utcnow(),
        }

        self.pages.update_one(
            {'url': url},
            {'$set': document, '$setOnInsert': {'created_at': datetime.utcnow()}},
            upsert=True
        )

    def bulk_save(self, documents: list[dict]):
        """Batch save with bulk operations"""
        operations = [
            UpdateOne(
                {'url': doc['url']},
                {'$set': doc, '$setOnInsert': {'created_at': datetime.utcnow()}},
                upsert=True
            )
            for doc in documents
        ]

        try:
            result = self.pages.bulk_write(operations, ordered=False)
            return result.upserted_count + result.modified_count
        except BulkWriteError as e:
            # Handle partial failures
            return e.details['nInserted'] + e.details['nModified']

    def find_by_domain(self, domain: str, limit: int = 100):
        """Query pages by domain"""
        return list(self.pages.find(
            {'domain': domain},
            limit=limit,
            sort=[('crawled_at', -1)]
        ))
```

### 3.3 Flexible Schema for Different Sources

```python
# Same collection, different structures
news_article = {
    'url': 'https://news.example.com/article/123',
    'type': 'news_article',
    'title': 'Breaking News',
    'author': 'Reporter Name',
    'content': 'Article body...',
    'published_date': datetime.utcnow(),
    'tags': ['politics', 'economy'],
}

product_listing = {
    'url': 'https://shop.example.com/product/456',
    'type': 'product',
    'name': 'Product Name',
    'price': 99.99,
    'currency': 'USD',
    'specifications': {
        'color': 'blue',
        'size': 'medium',
        'weight': '500g'
    },
    'reviews_count': 150,
    'rating': 4.5,
}

# Both stored in same collection, queried by type
db.pages.insert_many([news_article, product_listing])
db.pages.find({'type': 'product', 'price': {'$lt': 100}})
```

---

## 4. Data Deduplication

### 4.1 URL-Based Deduplication

```python
import hashlib
from urllib.parse import urlparse, parse_qs, urlencode

class URLDeduplicator:
    def __init__(self):
        self.seen_hashes = set()

    def normalize_url(self, url: str) -> str:
        """Normalize URL for consistent hashing"""
        parsed = urlparse(url)

        # Sort query parameters
        query_params = parse_qs(parsed.query, keep_blank_values=True)
        sorted_query = urlencode(sorted(query_params.items()), doseq=True)

        # Rebuild URL
        normalized = f"{parsed.scheme}://{parsed.netloc}{parsed.path}"
        if sorted_query:
            normalized += f"?{sorted_query}"

        return normalized.lower().rstrip('/')

    def get_hash(self, url: str) -> str:
        """Get hash for normalized URL"""
        normalized = self.normalize_url(url)
        return hashlib.sha256(normalized.encode()).hexdigest()[:16]

    def is_duplicate(self, url: str) -> bool:
        """Check if URL was already seen"""
        url_hash = self.get_hash(url)
        if url_hash in self.seen_hashes:
            return True
        self.seen_hashes.add(url_hash)
        return False
```

### 4.2 Content-Based Deduplication

```python
import simhash

class ContentDeduplicator:
    def __init__(self, threshold: int = 3):
        self.threshold = threshold  # Hamming distance threshold
        self.seen_hashes = []

    def get_simhash(self, content: str) -> int:
        """Generate SimHash for content"""
        # Tokenize
        tokens = content.lower().split()
        # Generate features (word n-grams)
        features = [' '.join(tokens[i:i+3]) for i in range(len(tokens)-2)]
        return simhash.Simhash(features).value

    def is_near_duplicate(self, content: str) -> bool:
        """Check if content is near-duplicate of seen content"""
        new_hash = self.get_simhash(content)

        for seen_hash in self.seen_hashes:
            distance = bin(new_hash ^ seen_hash).count('1')
            if distance <= self.threshold:
                return True

        self.seen_hashes.append(new_hash)
        return False
```

---

## 5. Data Pipeline Architecture

### 5.1 ETL Pipeline

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Crawl   │───▶│  Parse   │───▶│Transform │───▶│   Load   │
│  (Raw)   │    │ (Extract)│    │ (Clean)  │    │  (Store) │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │
     ▼               ▼               ▼               ▼
  Raw HTML       Structured      Normalized      Database
  Storage        Data            Data            Tables
```

### 5.2 Implementation

```python
from dataclasses import dataclass
from typing import Any
import json

@dataclass
class CrawlResult:
    url: str
    html: str
    status_code: int
    headers: dict
    crawled_at: datetime

@dataclass
class ParsedData:
    url: str
    title: str
    content: dict
    metadata: dict

class DataPipeline:
    def __init__(self, db_manager, raw_storage):
        self.db = db_manager
        self.raw_storage = raw_storage

    async def process(self, crawl_result: CrawlResult):
        """Process crawled page through pipeline"""
        # 1. Store raw HTML
        await self.store_raw(crawl_result)

        # 2. Parse/Extract
        parsed = await self.parse(crawl_result)

        # 3. Transform/Clean
        cleaned = await self.transform(parsed)

        # 4. Load to database
        await self.load(cleaned)

    async def store_raw(self, result: CrawlResult):
        """Store raw HTML for reprocessing"""
        filename = f"{result.url.replace('/', '_')}.html"
        await self.raw_storage.save(filename, result.html)

    async def parse(self, result: CrawlResult) -> ParsedData:
        """Extract structured data from HTML"""
        from bs4 import BeautifulSoup
        soup = BeautifulSoup(result.html, 'lxml')

        return ParsedData(
            url=result.url,
            title=soup.title.string if soup.title else '',
            content=self.extract_content(soup),
            metadata=self.extract_metadata(soup),
        )

    async def transform(self, parsed: ParsedData) -> dict:
        """Clean and normalize data"""
        return {
            'url': parsed.url,
            'title': parsed.title.strip(),
            'content': self.clean_content(parsed.content),
            'metadata': parsed.metadata,
            'processed_at': datetime.utcnow(),
        }

    async def load(self, data: dict):
        """Store in database"""
        await self.db.save_page(data)
```

---

## 6. Data Export Formats

### 6.1 Export Utilities

```python
import csv
import json
from pathlib import Path

class DataExporter:
    def __init__(self, db_manager):
        self.db = db_manager

    def to_csv(self, query: dict, output_path: str, fields: list[str]):
        """Export query results to CSV"""
        results = self.db.find(query)

        with open(output_path, 'w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=fields)
            writer.writeheader()

            for doc in results:
                row = {field: doc.get(field, '') for field in fields}
                writer.writerow(row)

    def to_jsonl(self, query: dict, output_path: str):
        """Export to JSON Lines format"""
        results = self.db.find(query)

        with open(output_path, 'w', encoding='utf-8') as f:
            for doc in results:
                doc['_id'] = str(doc.get('_id', ''))  # Convert ObjectId
                f.write(json.dumps(doc, default=str) + '\n')

    def to_parquet(self, query: dict, output_path: str):
        """Export to Parquet for analytics"""
        import pandas as pd

        results = list(self.db.find(query))
        df = pd.DataFrame(results)
        df.to_parquet(output_path, index=False)
```

---

## 7. Best Practices

### 7.1 Performance Tips

| Tip | Description |
|-----|-------------|
| **Batch Writes** | Insert 100-1000 documents at once |
| **Indexing** | Index fields used in queries |
| **Connection Pooling** | Reuse database connections |
| **Async Operations** | Use async drivers for I/O-bound work |
| **Compression** | Enable compression for large text fields |

### 7.2 Data Integrity

```python
# Ensure idempotent operations
def save_with_idempotency(self, url: str, data: dict):
    """Idempotent save - safe to retry"""
    existing = self.db.find_one({'url': url})

    if existing and existing.get('version', 0) >= data.get('version', 0):
        return  # Already have newer version

    self.db.update_one(
        {'url': url},
        {'$set': data},
        upsert=True
    )
```

### 7.3 Backup Strategy

```bash
# PostgreSQL backup
pg_dump -Fc crawler_db > backup_$(date +%Y%m%d).dump

# MongoDB backup
mongodump --db crawler --out ./backups/$(date +%Y%m%d)

# Restore
pg_restore -d crawler_db backup.dump
mongorestore --db crawler ./backups/20260205/crawler
```

---

## References

- [Data Storage for Web Scraping Guide](https://www.scrapeless.com/en/blog/data-storage)
- [PostgreSQL vs MySQL vs MongoDB Comparison](https://data-ox.com/comparison-postgresql-vs-mysql-vs-mongodb-for-web-scraping)
- [Web Scraping with Scrapy and MongoDB](https://realpython.com/web-scraping-with-scrapy-and-mongodb/)
- [Storing and Managing Scraped Data](https://roundproxies.com/blog/store-scraped-data/)

---

*Choose storage based on your data structure, query patterns, and scale requirements.*
