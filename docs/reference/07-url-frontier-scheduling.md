# URL Frontier Management & Crawl Scheduling

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: URL prioritization, frontier management, and recrawl scheduling strategies

## Overview

The URL Frontier is the data structure that stores and manages URLs waiting to be crawled. Effective frontier management is critical for crawler efficiency, politeness, and comprehensive coverage.

---

## 1. URL Frontier Architecture

### 1.1 Conceptual Model

```
┌─────────────────────────────────────────────────────────────────┐
│                        URL Frontier System                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌──────────────────┐         ┌──────────────────┐             │
│   │   Front Queues   │         │   Back Queues    │             │
│   │  (Prioritization)│────────▶│   (Politeness)   │             │
│   │                  │         │                  │             │
│   │  Priority 1: ▓▓▓ │         │  host1.com: ▓▓   │             │
│   │  Priority 2: ▓▓  │         │  host2.com: ▓    │             │
│   │  Priority 3: ▓   │         │  host3.com: ▓▓▓  │             │
│   └──────────────────┘         └────────┬─────────┘             │
│                                         │                        │
│                                         ▼                        │
│                              ┌──────────────────┐               │
│                              │   Heap (Time)    │               │
│                              │ ─────────────────│               │
│                              │ host3: now       │               │
│                              │ host1: now+2s    │               │
│                              │ host2: now+5s    │               │
│                              └──────────────────┘               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Dual-Queue System

**Front Queues (Prioritization)**:
- URLs ordered by importance/priority
- Multiple priority levels (typically 3-5)
- Higher priority = crawled first

**Back Queues (Politeness)**:
- One queue per host/domain
- Ensures crawl-delay compliance
- Prevents server overload

### 1.3 Core Implementation

```python
import heapq
import time
from collections import defaultdict
from dataclasses import dataclass, field
from typing import Optional
from urllib.parse import urlparse
import hashlib

@dataclass(order=True)
class URLEntry:
    priority: int
    timestamp: float = field(compare=False)
    url: str = field(compare=False)
    depth: int = field(compare=False, default=0)
    metadata: dict = field(compare=False, default_factory=dict)

class URLFrontier:
    def __init__(self, num_priority_levels: int = 3):
        self.num_priority_levels = num_priority_levels

        # Front queues: prioritization
        self.front_queues = [[] for _ in range(num_priority_levels)]

        # Back queues: per-host queues
        self.back_queues = defaultdict(list)

        # Heap for scheduling (next available time per host)
        self.host_schedule = []  # (time, host)

        # Host -> next available time
        self.host_next_time = {}

        # Deduplication
        self.seen_urls = set()

        # Default crawl delay
        self.default_delay = 2.0

    def add_url(self, url: str, priority: int = 1, depth: int = 0,
                metadata: dict = None):
        """Add URL to frontier"""
        url_hash = self._hash_url(url)

        if url_hash in self.seen_urls:
            return False

        self.seen_urls.add(url_hash)

        entry = URLEntry(
            priority=priority,
            timestamp=time.time(),
            url=url,
            depth=depth,
            metadata=metadata or {}
        )

        # Add to appropriate front queue
        priority_idx = min(priority, self.num_priority_levels - 1)
        heapq.heappush(self.front_queues[priority_idx], entry)

        return True

    def get_next_url(self) -> Optional[str]:
        """Get next URL respecting priorities and politeness"""
        # Find highest priority non-empty queue
        for priority_idx in range(self.num_priority_levels):
            if self.front_queues[priority_idx]:
                entry = heapq.heappop(self.front_queues[priority_idx])
                host = urlparse(entry.url).netloc

                # Check if host is ready
                next_time = self.host_next_time.get(host, 0)
                current_time = time.time()

                if current_time >= next_time:
                    # Update next available time
                    self.host_next_time[host] = current_time + self.default_delay
                    return entry.url
                else:
                    # Re-queue and try another
                    heapq.heappush(self.front_queues[priority_idx], entry)
                    continue

        return None

    def _hash_url(self, url: str) -> str:
        """Normalize and hash URL"""
        normalized = url.lower().rstrip('/')
        return hashlib.sha256(normalized.encode()).hexdigest()[:16]

    def set_crawl_delay(self, host: str, delay: float):
        """Set custom crawl delay for host"""
        self.host_next_time[host] = time.time() + delay

    def size(self) -> int:
        """Total URLs in frontier"""
        return sum(len(q) for q in self.front_queues)
```

---

## 2. URL Prioritization Strategies

### 2.1 Priority Scoring Factors

| Factor | Weight | Description |
|--------|--------|-------------|
| **Page Importance** | High | Homepage, category pages vs. deep pages |
| **Update Frequency** | High | How often content changes |
| **Link Depth** | Medium | Distance from seed URLs |
| **Backlink Count** | Medium | Number of inbound links |
| **Content Freshness** | Medium | Time since last crawl |
| **Historical Value** | Low | Past data quality from page |

### 2.2 Priority Calculator

```python
from dataclasses import dataclass
from datetime import datetime, timedelta
from urllib.parse import urlparse

@dataclass
class PageMetrics:
    depth: int = 0
    backlink_count: int = 0
    last_crawled: datetime = None
    change_frequency: str = 'unknown'  # hourly, daily, weekly, monthly
    historical_success_rate: float = 1.0

class PriorityCalculator:
    # Change frequency to hours mapping
    FREQUENCY_HOURS = {
        'hourly': 1,
        'daily': 24,
        'weekly': 168,
        'monthly': 720,
        'unknown': 168,  # Default to weekly
    }

    def calculate_priority(self, url: str, metrics: PageMetrics) -> int:
        """Calculate priority score (lower = higher priority)"""
        score = 0

        # 1. Page type bonus (homepages, category pages)
        path = urlparse(url).path
        if path in ['/', '']:
            score -= 100  # Highest priority
        elif path.count('/') <= 2:
            score -= 50   # Category pages

        # 2. Depth penalty
        score += metrics.depth * 10

        # 3. Backlink bonus
        score -= min(metrics.backlink_count, 100)

        # 4. Freshness need
        if metrics.last_crawled:
            hours_since_crawl = (
                datetime.utcnow() - metrics.last_crawled
            ).total_seconds() / 3600

            expected_hours = self.FREQUENCY_HOURS[metrics.change_frequency]

            if hours_since_crawl > expected_hours:
                # Overdue for recrawl
                overdue_ratio = hours_since_crawl / expected_hours
                score -= int(overdue_ratio * 30)

        # 5. Historical success bonus
        score -= int(metrics.historical_success_rate * 20)

        return max(0, score)  # Ensure non-negative

    def classify_priority(self, score: int) -> int:
        """Convert score to priority level (0=highest, 2=lowest)"""
        if score < 50:
            return 0  # High priority
        elif score < 150:
            return 1  # Medium priority
        else:
            return 2  # Low priority
```

### 2.3 Sitemap-Based Prioritization

```python
import xml.etree.ElementTree as ET
from datetime import datetime

class SitemapParser:
    def parse_sitemap(self, xml_content: str) -> list[dict]:
        """Parse sitemap and extract URLs with metadata"""
        root = ET.fromstring(xml_content)
        namespace = {'ns': 'http://www.sitemaps.org/schemas/sitemap/0.9'}

        urls = []
        for url_elem in root.findall('.//ns:url', namespace):
            url_data = {
                'url': url_elem.find('ns:loc', namespace).text,
                'lastmod': None,
                'changefreq': 'unknown',
                'priority': 0.5,
            }

            lastmod = url_elem.find('ns:lastmod', namespace)
            if lastmod is not None:
                url_data['lastmod'] = datetime.fromisoformat(
                    lastmod.text.replace('Z', '+00:00')
                )

            changefreq = url_elem.find('ns:changefreq', namespace)
            if changefreq is not None:
                url_data['changefreq'] = changefreq.text

            priority = url_elem.find('ns:priority', namespace)
            if priority is not None:
                url_data['priority'] = float(priority.text)

            urls.append(url_data)

        return urls

    def prioritize_from_sitemap(self, urls: list[dict]) -> list[tuple]:
        """Convert sitemap data to prioritized URL list"""
        prioritized = []

        for url_data in urls:
            # Higher sitemap priority = lower queue priority (crawl first)
            queue_priority = int((1 - url_data['priority']) * 100)

            # Boost recently modified pages
            if url_data['lastmod']:
                days_old = (datetime.utcnow() - url_data['lastmod']).days
                if days_old < 7:
                    queue_priority -= 20

            prioritized.append((url_data['url'], queue_priority))

        return sorted(prioritized, key=lambda x: x[1])
```

---

## 3. Recrawl Scheduling

### 3.1 Recrawl Strategies

| Strategy | Best For | Implementation |
|----------|----------|----------------|
| **Fixed Interval** | Static sites | Recrawl every N days |
| **Adaptive** | Dynamic sites | Based on change detection |
| **Event-Driven** | News, prices | External triggers |
| **Freshness-Based** | Mixed content | Priority by staleness |

### 3.2 Adaptive Recrawl Scheduler

```python
from datetime import datetime, timedelta
from collections import defaultdict
import math

class AdaptiveRecrawlScheduler:
    def __init__(self):
        # URL -> change history
        self.change_history = defaultdict(list)

        # URL -> last content hash
        self.content_hashes = {}

        # URL -> estimated change rate (changes per day)
        self.change_rates = {}

        # Minimum and maximum recrawl intervals
        self.min_interval = timedelta(hours=1)
        self.max_interval = timedelta(days=30)

    def record_crawl(self, url: str, content_hash: str):
        """Record a crawl and detect changes"""
        now = datetime.utcnow()
        changed = False

        if url in self.content_hashes:
            if self.content_hashes[url] != content_hash:
                changed = True
                self.change_history[url].append(now)

        self.content_hashes[url] = content_hash

        # Update change rate estimate
        self._update_change_rate(url)

        return changed

    def _update_change_rate(self, url: str):
        """Update estimated change rate based on history"""
        history = self.change_history[url]

        if len(history) < 2:
            # Default: assume weekly changes
            self.change_rates[url] = 1/7
            return

        # Calculate average interval between changes
        intervals = []
        for i in range(1, len(history)):
            delta = (history[i] - history[i-1]).total_seconds() / 86400  # days
            intervals.append(delta)

        if intervals:
            avg_interval = sum(intervals) / len(intervals)
            self.change_rates[url] = 1 / max(avg_interval, 0.1)

    def get_next_crawl_time(self, url: str) -> datetime:
        """Calculate when URL should be recrawled"""
        change_rate = self.change_rates.get(url, 1/7)  # Default weekly

        # Interval inversely proportional to change rate
        # More changes = shorter interval
        interval_days = 1 / change_rate

        # Apply bounds
        interval = timedelta(days=interval_days)
        interval = max(self.min_interval, min(interval, self.max_interval))

        # Add some randomization to spread load
        jitter = interval * 0.1
        import random
        interval += timedelta(seconds=random.uniform(-jitter.total_seconds(),
                                                      jitter.total_seconds()))

        return datetime.utcnow() + interval

    def get_overdue_urls(self, urls: list[str]) -> list[tuple]:
        """Get URLs that are overdue for recrawl, sorted by urgency"""
        now = datetime.utcnow()
        overdue = []

        for url in urls:
            next_crawl = self.get_next_crawl_time(url)
            if next_crawl <= now:
                # Calculate how overdue
                overdue_hours = (now - next_crawl).total_seconds() / 3600
                overdue.append((url, overdue_hours))

        # Sort by most overdue first
        return sorted(overdue, key=lambda x: -x[1])
```

### 3.3 Content Change Detection

```python
import hashlib
from difflib import SequenceMatcher

class ChangeDetector:
    def __init__(self, significance_threshold: float = 0.05):
        self.significance_threshold = significance_threshold
        self.content_store = {}

    def compute_hash(self, content: str) -> str:
        """Compute content hash (excluding dynamic elements)"""
        # Remove common dynamic elements
        cleaned = self._clean_content(content)
        return hashlib.sha256(cleaned.encode()).hexdigest()

    def _clean_content(self, content: str) -> str:
        """Remove dynamic elements that shouldn't trigger change detection"""
        import re

        # Remove timestamps, session IDs, etc.
        patterns = [
            r'\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}',  # Timestamps
            r'session[_-]?id=[a-zA-Z0-9]+',  # Session IDs
            r'nonce=[a-zA-Z0-9]+',  # Nonces
            r'csrf[_-]?token=[a-zA-Z0-9]+',  # CSRF tokens
        ]

        for pattern in patterns:
            content = re.sub(pattern, '', content)

        return content

    def has_significant_change(self, url: str, new_content: str) -> tuple[bool, float]:
        """Check if content changed significantly"""
        new_hash = self.compute_hash(new_content)

        if url not in self.content_store:
            self.content_store[url] = {
                'hash': new_hash,
                'content': new_content
            }
            return True, 1.0  # First crawl

        old_hash = self.content_store[url]['hash']

        if old_hash == new_hash:
            return False, 0.0  # No change

        # Calculate change ratio
        old_content = self.content_store[url]['content']
        ratio = SequenceMatcher(None, old_content, new_content).ratio()
        change_ratio = 1 - ratio

        # Update store
        self.content_store[url] = {
            'hash': new_hash,
            'content': new_content
        }

        return change_ratio >= self.significance_threshold, change_ratio
```

---

## 4. Politeness Management

### 4.1 Politeness Heuristics

```
Common Heuristic: Wait time = 10 × fetch duration

Example:
- Page took 200ms to fetch
- Wait 2 seconds before next request to same host
```

### 4.2 Adaptive Politeness

```python
import time
from collections import defaultdict

class AdaptivePoliteness:
    def __init__(self, base_delay: float = 1.0):
        self.base_delay = base_delay
        self.host_stats = defaultdict(lambda: {
            'last_request': 0,
            'last_duration': 0,
            'error_count': 0,
            'success_count': 0,
        })

    def get_delay(self, host: str) -> float:
        """Calculate appropriate delay for host"""
        stats = self.host_stats[host]

        # Base: 10x last fetch duration
        duration_based = stats['last_duration'] * 10

        # Adjust for error rate
        total = stats['error_count'] + stats['success_count']
        if total > 0:
            error_rate = stats['error_count'] / total
            if error_rate > 0.1:
                # High error rate - slow down significantly
                duration_based *= (1 + error_rate * 5)

        return max(self.base_delay, duration_based)

    def record_request(self, host: str, duration: float, success: bool):
        """Record request outcome"""
        stats = self.host_stats[host]
        stats['last_request'] = time.time()
        stats['last_duration'] = duration

        if success:
            stats['success_count'] += 1
            # Decay error count on success
            stats['error_count'] = max(0, stats['error_count'] - 0.1)
        else:
            stats['error_count'] += 1

    def can_request(self, host: str) -> tuple[bool, float]:
        """Check if can make request, return wait time if not"""
        stats = self.host_stats[host]
        elapsed = time.time() - stats['last_request']
        required_delay = self.get_delay(host)

        if elapsed >= required_delay:
            return True, 0

        return False, required_delay - elapsed
```

---

## 5. Large-Scale Frontier Persistence

### 5.1 Redis-Based Frontier

```python
import redis
import json
import time

class RedisFrontier:
    def __init__(self, redis_url: str = 'redis://localhost:6379'):
        self.redis = redis.from_url(redis_url)
        self.queue_prefix = 'frontier:queue:'
        self.seen_key = 'frontier:seen'
        self.schedule_key = 'frontier:schedule'

    def add_url(self, url: str, priority: int = 1,
                metadata: dict = None) -> bool:
        """Add URL to frontier"""
        url_hash = self._hash_url(url)

        # Check if already seen
        if self.redis.sismember(self.seen_key, url_hash):
            return False

        # Add to seen set
        self.redis.sadd(self.seen_key, url_hash)

        # Add to priority queue (sorted set, score = priority)
        entry = json.dumps({
            'url': url,
            'metadata': metadata or {},
            'added': time.time()
        })

        self.redis.zadd(
            f'{self.queue_prefix}{priority}',
            {entry: time.time()}
        )

        return True

    def get_next_urls(self, count: int = 10) -> list[dict]:
        """Get next batch of URLs respecting priorities"""
        urls = []

        for priority in range(3):  # 0, 1, 2
            queue_key = f'{self.queue_prefix}{priority}'

            while len(urls) < count:
                # Pop from sorted set
                entries = self.redis.zpopmin(queue_key, count - len(urls))

                if not entries:
                    break

                for entry, score in entries:
                    data = json.loads(entry)
                    urls.append(data)

            if len(urls) >= count:
                break

        return urls

    def size(self) -> dict:
        """Get frontier size by priority"""
        sizes = {}
        for priority in range(3):
            queue_key = f'{self.queue_prefix}{priority}'
            sizes[priority] = self.redis.zcard(queue_key)
        return sizes

    def _hash_url(self, url: str) -> str:
        import hashlib
        return hashlib.sha256(url.lower().encode()).hexdigest()[:16]
```

### 5.2 Disk-Based Overflow

```python
import sqlite3
from pathlib import Path

class DiskBackedFrontier:
    """Frontier with memory queue + disk overflow"""

    def __init__(self, db_path: str, memory_limit: int = 100000):
        self.db_path = Path(db_path)
        self.memory_limit = memory_limit
        self.memory_queue = []

        self._init_db()

    def _init_db(self):
        """Initialize SQLite database"""
        self.conn = sqlite3.connect(str(self.db_path))
        self.conn.execute('''
            CREATE TABLE IF NOT EXISTS frontier (
                id INTEGER PRIMARY KEY,
                url TEXT UNIQUE,
                priority INTEGER,
                metadata TEXT,
                added REAL
            )
        ''')
        self.conn.execute('CREATE INDEX IF NOT EXISTS idx_priority ON frontier(priority)')
        self.conn.commit()

    def add_url(self, url: str, priority: int, metadata: dict = None):
        """Add URL - memory first, overflow to disk"""
        entry = {
            'url': url,
            'priority': priority,
            'metadata': metadata or {}
        }

        if len(self.memory_queue) < self.memory_limit:
            self.memory_queue.append(entry)
        else:
            # Overflow to disk
            self._write_to_disk(entry)

    def _write_to_disk(self, entry: dict):
        """Write entry to disk"""
        import json
        self.conn.execute('''
            INSERT OR IGNORE INTO frontier (url, priority, metadata, added)
            VALUES (?, ?, ?, ?)
        ''', (entry['url'], entry['priority'],
              json.dumps(entry['metadata']), time.time()))
        self.conn.commit()

    def get_next_url(self) -> dict | None:
        """Get next URL from memory, refill from disk if needed"""
        if not self.memory_queue:
            self._refill_from_disk()

        if self.memory_queue:
            # Sort by priority and return
            self.memory_queue.sort(key=lambda x: x['priority'])
            return self.memory_queue.pop(0)

        return None

    def _refill_from_disk(self, count: int = 10000):
        """Refill memory queue from disk"""
        cursor = self.conn.execute('''
            SELECT id, url, priority, metadata FROM frontier
            ORDER BY priority
            LIMIT ?
        ''', (count,))

        ids_to_delete = []
        for row in cursor:
            self.memory_queue.append({
                'url': row[1],
                'priority': row[2],
                'metadata': json.loads(row[3])
            })
            ids_to_delete.append(row[0])

        if ids_to_delete:
            placeholders = ','.join('?' * len(ids_to_delete))
            self.conn.execute(
                f'DELETE FROM frontier WHERE id IN ({placeholders})',
                ids_to_delete
            )
            self.conn.commit()
```

---

## 6. Best Practices

### 6.1 Frontier Management Checklist

```
□ Implement URL normalization before deduplication
□ Use multi-level priority queues
□ Respect per-host crawl delays
□ Persist frontier state for crash recovery
□ Monitor queue depths by priority
□ Implement adaptive recrawl scheduling
□ Use content hashing for change detection
□ Handle frontier overflow to disk
```

### 6.2 Common Pitfalls

| Pitfall | Solution |
|---------|----------|
| Memory exhaustion | Disk-backed frontier with memory cache |
| Duplicate crawls | Proper URL normalization + hash dedup |
| Unfair host distribution | Per-host back queues |
| Stale priorities | Periodic priority recalculation |
| Lost state on crash | Persistent storage + checkpointing |

---

## References

- [Stanford NLP - The URL Frontier](https://nlp.stanford.edu/IR-book/html/htmledition/the-url-frontier-1.html)
- [What is a URL Frontier?](https://www.firecrawl.dev/glossary/web-crawling-apis/what-is-url-frontier-web-crawling)
- [Frontera Documentation](https://frontera.readthedocs.io/en/v0.2.0/topics/what-is-a-crawl-frontier.html)
- [System Design - Web Crawler](https://medium.com/@tahir.rauf/system-design-web-crawler-b9333e12536c)

---

*Effective frontier management balances coverage, freshness, and politeness.*
