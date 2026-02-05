# Data Quality Assurance Guide

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Data validation, cleaning, and quality assurance for web scraped data

## Overview

Collecting data is only half the battle. Ensuring data quality—that it's clean, accurate, and actionable—is equally important for any serious crawling project.

---

## 1. Data Quality Pipeline

### 1.1 Pipeline Architecture

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Scrape  │───▶│ Validate │───▶│  Clean   │───▶│  Enrich  │───▶│  Store   │
│  (Raw)   │    │ (Check)  │    │(Transform)│   │(Augment) │    │ (Final)  │
└──────────┘    └──────────┘    └──────────┘    └──────────┘    └──────────┘
     │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼
  Raw HTML      Type checks      Normalized     Added fields    Clean DB
              Schema match    Deduplication    Derived data
              Range checks    Format fixes
```

### 1.2 Quality Stages

| Stage | Purpose | Actions |
|-------|---------|---------|
| **Profiling** | Understand data characteristics | Analyze types, distributions, patterns |
| **Validation** | Ensure correctness | Type/range/format checks |
| **Cleansing** | Remove noise | Strip HTML, fix encoding, trim |
| **Normalization** | Standardize formats | Dates, currencies, units |
| **Deduplication** | Remove duplicates | Exact and fuzzy matching |
| **Enrichment** | Add derived data | Categories, calculations |

---

## 2. Validation Framework

### 2.1 Schema Validation with Pydantic

```python
from pydantic import BaseModel, Field, validator, HttpUrl
from typing import Optional
from datetime import datetime
from decimal import Decimal

class ProductSchema(BaseModel):
    """Validated product data schema"""

    url: HttpUrl
    name: str = Field(..., min_length=1, max_length=500)
    price: Decimal = Field(..., gt=0, le=1000000)
    currency: str = Field(..., regex='^[A-Z]{3}$')
    in_stock: bool
    description: Optional[str] = Field(None, max_length=10000)
    rating: Optional[float] = Field(None, ge=0, le=5)
    review_count: Optional[int] = Field(None, ge=0)
    scraped_at: datetime = Field(default_factory=datetime.utcnow)

    @validator('name')
    def clean_name(cls, v):
        """Clean and validate product name"""
        v = v.strip()
        if not v or v.lower() in ['null', 'n/a', 'undefined']:
            raise ValueError('Invalid product name')
        return v

    @validator('price', pre=True)
    def parse_price(cls, v):
        """Parse price from various formats"""
        if isinstance(v, str):
            # Remove currency symbols and commas
            v = v.replace('$', '').replace('€', '').replace(',', '')
            v = v.replace('₩', '').replace('원', '').strip()
        return Decimal(str(v))

    @validator('currency')
    def validate_currency(cls, v):
        """Validate currency code"""
        valid_currencies = ['USD', 'EUR', 'GBP', 'KRW', 'JPY', 'CNY']
        if v not in valid_currencies:
            raise ValueError(f'Invalid currency: {v}')
        return v

    class Config:
        extra = 'forbid'  # Reject unknown fields


def validate_product(data: dict) -> tuple[bool, ProductSchema | str]:
    """Validate product data against schema"""
    try:
        validated = ProductSchema(**data)
        return True, validated
    except Exception as e:
        return False, str(e)
```

### 2.2 Field-Level Validators

```python
import re
from typing import Any

class FieldValidators:
    """Common field validation functions"""

    @staticmethod
    def is_valid_email(value: str) -> bool:
        """Validate email format"""
        pattern = r'^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
        return bool(re.match(pattern, value))

    @staticmethod
    def is_valid_phone(value: str, country: str = 'KR') -> bool:
        """Validate phone number format"""
        patterns = {
            'KR': r'^(010|011|016|017|018|019)-?\d{3,4}-?\d{4}$',
            'US': r'^\+?1?[-.\s]?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}$',
        }
        pattern = patterns.get(country, patterns['US'])
        return bool(re.match(pattern, value))

    @staticmethod
    def is_valid_url(value: str) -> bool:
        """Validate URL format"""
        pattern = r'^https?://[^\s<>"{}|\\^`\[\]]+$'
        return bool(re.match(pattern, value))

    @staticmethod
    def is_in_range(value: float, min_val: float, max_val: float) -> bool:
        """Check if value is within range"""
        return min_val <= value <= max_val

    @staticmethod
    def is_not_empty(value: Any) -> bool:
        """Check if value is not empty/null"""
        if value is None:
            return False
        if isinstance(value, str) and not value.strip():
            return False
        if isinstance(value, (list, dict)) and not value:
            return False
        return True

    @staticmethod
    def is_valid_date(value: str, formats: list[str] = None) -> bool:
        """Validate date string"""
        from datetime import datetime

        formats = formats or [
            '%Y-%m-%d',
            '%Y/%m/%d',
            '%d-%m-%Y',
            '%d/%m/%Y',
            '%Y-%m-%dT%H:%M:%S',
        ]

        for fmt in formats:
            try:
                datetime.strptime(value, fmt)
                return True
            except ValueError:
                continue

        return False
```

### 2.3 Data Profiling

```python
from collections import Counter
from statistics import mean, median, stdev
from typing import Any

class DataProfiler:
    """Profile scraped data to understand quality"""

    def __init__(self):
        self.profiles = {}

    def profile_field(self, field_name: str, values: list[Any]) -> dict:
        """Generate profile for a single field"""
        non_null = [v for v in values if v is not None]

        profile = {
            'total_count': len(values),
            'null_count': len(values) - len(non_null),
            'null_rate': (len(values) - len(non_null)) / len(values) if values else 0,
            'unique_count': len(set(non_null)),
            'uniqueness_rate': len(set(non_null)) / len(non_null) if non_null else 0,
        }

        # Type-specific profiling
        if non_null:
            sample = non_null[0]

            if isinstance(sample, (int, float)):
                profile.update(self._profile_numeric(non_null))
            elif isinstance(sample, str):
                profile.update(self._profile_string(non_null))

        self.profiles[field_name] = profile
        return profile

    def _profile_numeric(self, values: list) -> dict:
        """Profile numeric field"""
        return {
            'min': min(values),
            'max': max(values),
            'mean': mean(values),
            'median': median(values),
            'stdev': stdev(values) if len(values) > 1 else 0,
            'zeros': sum(1 for v in values if v == 0),
            'negatives': sum(1 for v in values if v < 0),
        }

    def _profile_string(self, values: list[str]) -> dict:
        """Profile string field"""
        lengths = [len(v) for v in values]
        return {
            'min_length': min(lengths),
            'max_length': max(lengths),
            'avg_length': mean(lengths),
            'empty_strings': sum(1 for v in values if not v.strip()),
            'top_values': Counter(values).most_common(5),
        }

    def detect_anomalies(self, field_name: str, values: list) -> list[dict]:
        """Detect anomalous values"""
        anomalies = []
        profile = self.profiles.get(field_name)

        if not profile:
            profile = self.profile_field(field_name, values)

        for i, value in enumerate(values):
            if value is None:
                continue

            # Numeric outlier detection (IQR method)
            if isinstance(value, (int, float)) and 'mean' in profile:
                z_score = abs(value - profile['mean']) / (profile['stdev'] + 0.001)
                if z_score > 3:
                    anomalies.append({
                        'index': i,
                        'value': value,
                        'reason': f'Outlier (z-score: {z_score:.2f})'
                    })

        return anomalies

    def generate_report(self) -> str:
        """Generate quality report"""
        lines = ["# Data Quality Report\n"]

        for field, profile in self.profiles.items():
            lines.append(f"\n## {field}")
            lines.append(f"- Total: {profile['total_count']}")
            lines.append(f"- Null Rate: {profile['null_rate']:.1%}")
            lines.append(f"- Unique: {profile['unique_count']}")

            if 'mean' in profile:
                lines.append(f"- Range: {profile['min']} - {profile['max']}")
                lines.append(f"- Mean: {profile['mean']:.2f}")

        return '\n'.join(lines)
```

---

## 3. Data Cleaning

### 3.1 Text Cleaning

```python
import re
import html
import unicodedata

class TextCleaner:
    """Clean and normalize text data"""

    @staticmethod
    def clean_html(text: str) -> str:
        """Remove HTML tags and decode entities"""
        # Decode HTML entities
        text = html.unescape(text)

        # Remove HTML tags
        text = re.sub(r'<[^>]+>', ' ', text)

        # Remove extra whitespace
        text = ' '.join(text.split())

        return text.strip()

    @staticmethod
    def normalize_unicode(text: str) -> str:
        """Normalize Unicode characters"""
        # Normalize to NFC form
        text = unicodedata.normalize('NFC', text)

        # Replace common problematic characters
        replacements = {
            '\u00a0': ' ',   # Non-breaking space
            '\u2019': "'",   # Right single quote
            '\u2018': "'",   # Left single quote
            '\u201c': '"',   # Left double quote
            '\u201d': '"',   # Right double quote
            '\u2013': '-',   # En dash
            '\u2014': '-',   # Em dash
        }

        for old, new in replacements.items():
            text = text.replace(old, new)

        return text

    @staticmethod
    def clean_whitespace(text: str) -> str:
        """Normalize whitespace"""
        # Replace various whitespace with regular space
        text = re.sub(r'[\t\r\n]+', ' ', text)

        # Collapse multiple spaces
        text = re.sub(r' +', ' ', text)

        return text.strip()

    @staticmethod
    def remove_control_chars(text: str) -> str:
        """Remove control characters"""
        return ''.join(char for char in text
                      if unicodedata.category(char) != 'Cc')

    @classmethod
    def clean(cls, text: str) -> str:
        """Apply all cleaning steps"""
        if not text:
            return ''

        text = cls.clean_html(text)
        text = cls.normalize_unicode(text)
        text = cls.remove_control_chars(text)
        text = cls.clean_whitespace(text)

        return text
```

### 3.2 Price Cleaning

```python
import re
from decimal import Decimal, InvalidOperation

class PriceCleaner:
    """Clean and parse price data"""

    CURRENCY_SYMBOLS = {
        '$': 'USD',
        '€': 'EUR',
        '£': 'GBP',
        '¥': 'JPY',
        '₩': 'KRW',
        '원': 'KRW',
    }

    @classmethod
    def parse(cls, price_str: str) -> tuple[Decimal | None, str | None]:
        """Parse price string to decimal and currency"""
        if not price_str:
            return None, None

        price_str = price_str.strip()

        # Detect currency
        currency = None
        for symbol, code in cls.CURRENCY_SYMBOLS.items():
            if symbol in price_str:
                currency = code
                price_str = price_str.replace(symbol, '')
                break

        # Clean price string
        # Remove thousand separators
        price_str = price_str.replace(',', '')

        # Handle Korean format (123,456원)
        price_str = price_str.replace('원', '').strip()

        # Extract numeric part
        match = re.search(r'[\d.]+', price_str)
        if not match:
            return None, currency

        try:
            price = Decimal(match.group())
            return price, currency
        except InvalidOperation:
            return None, currency

    @classmethod
    def normalize(cls, price: Decimal, from_currency: str,
                  to_currency: str = 'USD',
                  rates: dict = None) -> Decimal:
        """Convert price to target currency"""
        default_rates = {
            ('KRW', 'USD'): Decimal('0.00075'),
            ('EUR', 'USD'): Decimal('1.10'),
            ('GBP', 'USD'): Decimal('1.27'),
            ('JPY', 'USD'): Decimal('0.0067'),
        }

        rates = rates or default_rates

        if from_currency == to_currency:
            return price

        key = (from_currency, to_currency)
        if key in rates:
            return price * rates[key]

        return price  # Return original if no conversion available
```

### 3.3 Date Cleaning

```python
from datetime import datetime, date
from typing import Optional
import re

class DateCleaner:
    """Parse and normalize date data"""

    FORMATS = [
        '%Y-%m-%d',
        '%Y/%m/%d',
        '%d-%m-%Y',
        '%d/%m/%Y',
        '%Y.%m.%d',
        '%Y년 %m월 %d일',
        '%B %d, %Y',
        '%b %d, %Y',
        '%Y-%m-%dT%H:%M:%S',
        '%Y-%m-%dT%H:%M:%SZ',
        '%Y-%m-%dT%H:%M:%S%z',
    ]

    RELATIVE_PATTERNS = {
        r'(\d+)\s*(초|second)': lambda m: timedelta(seconds=int(m.group(1))),
        r'(\d+)\s*(분|minute)': lambda m: timedelta(minutes=int(m.group(1))),
        r'(\d+)\s*(시간|hour)': lambda m: timedelta(hours=int(m.group(1))),
        r'(\d+)\s*(일|day)': lambda m: timedelta(days=int(m.group(1))),
        r'(\d+)\s*(주|week)': lambda m: timedelta(weeks=int(m.group(1))),
    }

    @classmethod
    def parse(cls, date_str: str) -> Optional[datetime]:
        """Parse date string to datetime object"""
        if not date_str:
            return None

        date_str = date_str.strip()

        # Try standard formats
        for fmt in cls.FORMATS:
            try:
                return datetime.strptime(date_str, fmt)
            except ValueError:
                continue

        # Try relative dates
        for pattern, delta_func in cls.RELATIVE_PATTERNS.items():
            match = re.search(pattern, date_str, re.IGNORECASE)
            if match:
                from datetime import timedelta
                delta = delta_func(match)
                return datetime.now() - delta

        # Try parsing with dateutil as fallback
        try:
            from dateutil import parser
            return parser.parse(date_str)
        except:
            return None

    @classmethod
    def normalize(cls, dt: datetime, format: str = '%Y-%m-%d') -> str:
        """Normalize datetime to standard format"""
        return dt.strftime(format)
```

---

## 4. Deduplication

### 4.1 Exact Deduplication

```python
import hashlib
from typing import Callable

class ExactDeduplicator:
    """Remove exact duplicate records"""

    def __init__(self, key_fields: list[str]):
        self.key_fields = key_fields
        self.seen_hashes = set()

    def get_hash(self, record: dict) -> str:
        """Generate hash for record based on key fields"""
        key_values = [str(record.get(field, '')) for field in self.key_fields]
        key_string = '|'.join(key_values)
        return hashlib.sha256(key_string.encode()).hexdigest()

    def is_duplicate(self, record: dict) -> bool:
        """Check if record is duplicate"""
        record_hash = self.get_hash(record)

        if record_hash in self.seen_hashes:
            return True

        self.seen_hashes.add(record_hash)
        return False

    def deduplicate(self, records: list[dict]) -> list[dict]:
        """Remove duplicates from list of records"""
        unique = []
        for record in records:
            if not self.is_duplicate(record):
                unique.append(record)
        return unique
```

### 4.2 Fuzzy Deduplication

```python
from difflib import SequenceMatcher

class FuzzyDeduplicator:
    """Remove near-duplicate records using fuzzy matching"""

    def __init__(self, key_field: str, threshold: float = 0.9):
        self.key_field = key_field
        self.threshold = threshold
        self.seen_values = []

    def similarity(self, a: str, b: str) -> float:
        """Calculate similarity ratio between two strings"""
        return SequenceMatcher(None, a.lower(), b.lower()).ratio()

    def is_near_duplicate(self, record: dict) -> bool:
        """Check if record is similar to any seen record"""
        value = record.get(self.key_field, '')

        for seen in self.seen_values:
            if self.similarity(value, seen) >= self.threshold:
                return True

        self.seen_values.append(value)
        return False

    def deduplicate(self, records: list[dict]) -> list[dict]:
        """Remove near-duplicates from list"""
        unique = []
        for record in records:
            if not self.is_near_duplicate(record):
                unique.append(record)
        return unique
```

### 4.3 Record Linking

```python
class RecordLinker:
    """Link potentially duplicate records across sources"""

    def __init__(self, blocking_fields: list[str], comparison_fields: list[str]):
        self.blocking_fields = blocking_fields
        self.comparison_fields = comparison_fields

    def create_blocks(self, records: list[dict]) -> dict[str, list[dict]]:
        """Create blocks of potentially matching records"""
        blocks = {}

        for record in records:
            # Create blocking key
            key_parts = [str(record.get(f, ''))[:3].lower()
                        for f in self.blocking_fields]
            block_key = '|'.join(key_parts)

            if block_key not in blocks:
                blocks[block_key] = []
            blocks[block_key].append(record)

        return blocks

    def compare_records(self, rec1: dict, rec2: dict) -> float:
        """Calculate match score between two records"""
        scores = []

        for field in self.comparison_fields:
            val1 = str(rec1.get(field, '')).lower()
            val2 = str(rec2.get(field, '')).lower()

            if not val1 or not val2:
                continue

            similarity = SequenceMatcher(None, val1, val2).ratio()
            scores.append(similarity)

        return sum(scores) / len(scores) if scores else 0

    def find_matches(self, records: list[dict],
                     threshold: float = 0.8) -> list[tuple[dict, dict, float]]:
        """Find matching record pairs"""
        matches = []
        blocks = self.create_blocks(records)

        for block_records in blocks.values():
            if len(block_records) < 2:
                continue

            for i in range(len(block_records)):
                for j in range(i + 1, len(block_records)):
                    score = self.compare_records(
                        block_records[i], block_records[j]
                    )
                    if score >= threshold:
                        matches.append((
                            block_records[i],
                            block_records[j],
                            score
                        ))

        return matches
```

---

## 5. Real-Time Quality Monitoring

### 5.1 Scrapy Integration with Spidermon

```python
# monitors.py
from spidermon import Monitor, MonitorSuite, monitors
from spidermon.contrib.monitors.mixins import StatsMonitorMixin

class ItemValidationMonitor(Monitor, StatsMonitorMixin):
    """Monitor item validation during crawl"""

    @monitors.name('Item validation rate')
    def test_item_validation_rate(self):
        """Check that most items pass validation"""
        total = self.stats.get('item_scraped_count', 0)
        invalid = self.stats.get('item_dropped_count', 0)

        if total == 0:
            return

        error_rate = invalid / total
        self.assertTrue(
            error_rate < 0.1,
            msg=f'High validation error rate: {error_rate:.1%}'
        )

    @monitors.name('Required fields present')
    def test_required_fields(self):
        """Check required fields are present"""
        missing_fields = self.stats.get('item_missing_fields', {})

        for field, count in missing_fields.items():
            self.assertTrue(
                count == 0,
                msg=f'Missing required field {field}: {count} items'
            )


class SpiderCloseMonitorSuite(MonitorSuite):
    monitors = [ItemValidationMonitor]
```

### 5.2 Quality Metrics Dashboard

```python
from dataclasses import dataclass, field
from datetime import datetime
from typing import Dict, List

@dataclass
class QualityMetrics:
    """Track data quality metrics over time"""

    timestamp: datetime = field(default_factory=datetime.utcnow)
    total_records: int = 0
    valid_records: int = 0
    invalid_records: int = 0
    duplicate_records: int = 0
    null_rates: Dict[str, float] = field(default_factory=dict)
    validation_errors: Dict[str, int] = field(default_factory=dict)

    @property
    def validity_rate(self) -> float:
        """Calculate overall validity rate"""
        if self.total_records == 0:
            return 0
        return self.valid_records / self.total_records

    def to_dict(self) -> dict:
        """Convert to dictionary for storage/reporting"""
        return {
            'timestamp': self.timestamp.isoformat(),
            'total_records': self.total_records,
            'valid_records': self.valid_records,
            'validity_rate': f'{self.validity_rate:.1%}',
            'duplicate_records': self.duplicate_records,
            'null_rates': self.null_rates,
            'top_errors': dict(sorted(
                self.validation_errors.items(),
                key=lambda x: -x[1]
            )[:10]),
        }


class QualityTracker:
    """Track quality metrics across crawl sessions"""

    def __init__(self):
        self.sessions: List[QualityMetrics] = []
        self.current: QualityMetrics = None

    def start_session(self):
        """Start new tracking session"""
        self.current = QualityMetrics()

    def record_valid(self):
        """Record valid item"""
        self.current.total_records += 1
        self.current.valid_records += 1

    def record_invalid(self, error_type: str):
        """Record invalid item"""
        self.current.total_records += 1
        self.current.invalid_records += 1
        self.current.validation_errors[error_type] = \
            self.current.validation_errors.get(error_type, 0) + 1

    def record_duplicate(self):
        """Record duplicate item"""
        self.current.duplicate_records += 1

    def end_session(self):
        """End session and store metrics"""
        self.sessions.append(self.current)
        return self.current.to_dict()

    def get_trend(self, metric: str, periods: int = 10) -> List[float]:
        """Get trend for a metric over recent sessions"""
        recent = self.sessions[-periods:]
        return [getattr(s, metric, 0) for s in recent]
```

---

## 6. Best Practices Checklist

### Pre-Crawl
- [ ] Define data schema with validation rules
- [ ] Set up profiling for first batch
- [ ] Establish quality thresholds

### During Crawl
- [ ] Validate each item against schema
- [ ] Track validation error rates
- [ ] Monitor for anomalies
- [ ] Log quality metrics

### Post-Crawl
- [ ] Run full data profiling
- [ ] Generate quality report
- [ ] Review and fix systematic issues
- [ ] Archive quality metrics

---

## References

- [Data Quality Assurance in Web Scraping 2025](https://hirinfotech.com/data-quality-assurance-in-web-scraping-your-2025-guide-to-reliable-data/)
- [Data Cleaning After Web Scraping](https://hirinfotech.com/data-cleaning-after-web-scraping-the-essential-guide-for-2025/)
- [Data Validation for Reliable Web Scraping](https://www.scrapehero.com/data-validation-in-web-scraping/)
- [Web Data Quality Pipeline](https://substack.thewebscraping.club/p/web-data-quality-pipeline)

---

*Quality data drives quality decisions. Invest in validation and cleaning upfront.*
