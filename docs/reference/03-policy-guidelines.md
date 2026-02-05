# Web Crawling Policy Guidelines

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Purpose**: Ethical crawling policies, robots.txt compliance, and responsible data collection

## Overview

Responsible web crawling requires adherence to established protocols and ethical guidelines. This document outlines the policies that govern respectful and sustainable crawling practices.

---

## 1. Robots Exclusion Protocol (REP)

### 1.1 What is robots.txt?

`robots.txt` is a plain text file placed at the root of a website that communicates crawler access rules.

```
https://example.com/robots.txt
```

### 1.2 Key Directives

| Directive | Meaning | Example |
|-----------|---------|---------|
| `User-agent` | Specifies which crawler the rules apply to | `User-agent: *` |
| `Disallow` | Paths the crawler should not access | `Disallow: /private/` |
| `Allow` | Explicitly permitted paths | `Allow: /public/` |
| `Crawl-delay` | Seconds to wait between requests | `Crawl-delay: 10` |
| `Sitemap` | Location of XML sitemap | `Sitemap: /sitemap.xml` |

### 1.3 Example robots.txt

```
# Example robots.txt
User-agent: *
Disallow: /admin/
Disallow: /private/
Disallow: /api/internal/
Crawl-delay: 2

User-agent: Googlebot
Allow: /
Crawl-delay: 1

Sitemap: https://example.com/sitemap.xml
```

### 1.4 Legal Status

> **Important**: robots.txt is NOT legally binding, but it establishes intent.

| Aspect | Status |
|--------|--------|
| Legal Contract | ❌ No |
| Access Control System | ❌ No |
| Privacy Consent | ❌ No |
| Authentication | ❌ No |
| Ethical Guideline | ✅ Yes |
| Intent Signal | ✅ Yes |
| Good Faith Indicator | ✅ Yes |

**Legal Implications**:
- Ignoring robots.txt can undermine claims of acting in good faith
- Some jurisdictions (e.g., Texas) may consider ignoring robots.txt as DMCA violation
- Courts often consider robots.txt compliance when evaluating scraping cases

---

## 2. Crawl-Delay Implementation

### 2.1 Understanding Crawl-Delay

```
Crawl-delay: 10
```

This directive means: **Wait 10 seconds between requests** to this site.

### 2.2 Implementation in Python

```python
import time
from urllib.robotparser import RobotFileParser

class PoliteCrawler:
    def __init__(self, user_agent: str):
        self.user_agent = user_agent
        self.robot_parser = RobotFileParser()
        self.crawl_delays = {}

    def get_crawl_delay(self, url: str) -> float:
        """Get crawl delay for a domain from robots.txt"""
        from urllib.parse import urlparse

        domain = urlparse(url).netloc

        if domain not in self.crawl_delays:
            robots_url = f"https://{domain}/robots.txt"
            self.robot_parser.set_url(robots_url)
            self.robot_parser.read()

            delay = self.robot_parser.crawl_delay(self.user_agent)
            self.crawl_delays[domain] = delay or 1.0  # Default 1 second

        return self.crawl_delays[domain]

    def can_fetch(self, url: str) -> bool:
        """Check if URL is allowed by robots.txt"""
        return self.robot_parser.can_fetch(self.user_agent, url)
```

### 2.3 Scrapy Configuration

```python
# settings.py
ROBOTSTXT_OBEY = True
DOWNLOAD_DELAY = 2  # Default delay
RANDOMIZE_DOWNLOAD_DELAY = True  # Adds randomization

# Automatic crawl-delay from robots.txt
AUTOTHROTTLE_ENABLED = True
AUTOTHROTTLE_START_DELAY = 1
AUTOTHROTTLE_MAX_DELAY = 60
AUTOTHROTTLE_TARGET_CONCURRENCY = 1.0
```

---

## 3. User-Agent Policy

### 3.1 Requirements

A responsible crawler **must** identify itself with a clear, stable User-Agent string.

**Good User-Agent Format**:
```
MyCrawler/1.0 (+https://example.com/crawler-info; contact@example.com)
```

**Components**:
- Crawler name and version
- URL with crawler information
- Contact email for issues

### 3.2 Implementation

```python
# Custom User-Agent
USER_AGENT = 'MyCrawler/1.0 (+https://mycompany.com/crawler; crawler@mycompany.com)'

# In Scrapy settings.py
USER_AGENT = 'MyCrawler/1.0 (+https://mycompany.com/crawler)'
```

### 3.3 What NOT to Do

```python
# ❌ Don't impersonate browsers without good reason
USER_AGENT = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)...'

# ❌ Don't use empty or generic user agents
USER_AGENT = ''
USER_AGENT = 'Python-urllib/3.11'

# ❌ Don't impersonate Googlebot
USER_AGENT = 'Googlebot/2.1'
```

---

## 4. Rate Limiting Policies

### 4.1 Politeness Principles

| Principle | Description |
|-----------|-------------|
| **Respect Server Capacity** | Don't overwhelm with requests |
| **Honor Crawl-Delay** | Follow robots.txt timing |
| **Adaptive Throttling** | Slow down on errors |
| **Off-Peak Crawling** | Prefer low-traffic hours |

### 4.2 Recommended Defaults

```python
# Conservative defaults (recommended for unknown sites)
DEFAULT_DELAY = 2.0  # seconds between requests
MAX_CONCURRENT_REQUESTS = 8
REQUESTS_PER_DOMAIN = 2

# Aggressive (only with permission)
DEFAULT_DELAY = 0.5
MAX_CONCURRENT_REQUESTS = 32
REQUESTS_PER_DOMAIN = 8
```

### 4.3 Adaptive Rate Limiting

```python
class AdaptiveRateLimiter:
    def __init__(self, base_delay: float = 1.0):
        self.base_delay = base_delay
        self.current_delay = base_delay
        self.consecutive_errors = 0

    def on_success(self):
        """Decrease delay on success (with floor)"""
        self.consecutive_errors = 0
        self.current_delay = max(
            self.base_delay,
            self.current_delay * 0.9
        )

    def on_error(self, status_code: int):
        """Increase delay on error (exponential backoff)"""
        self.consecutive_errors += 1

        if status_code == 429:  # Too Many Requests
            self.current_delay *= 2
        elif status_code >= 500:
            self.current_delay *= 1.5

        # Cap at maximum delay
        self.current_delay = min(60.0, self.current_delay)

    def get_delay(self) -> float:
        return self.current_delay
```

---

## 5. Data Collection Ethics

### 5.1 Data Minimization

Collect only what you need:

| Data Type | Collection Policy |
|-----------|------------------|
| Public content | ✅ Collect if needed |
| Personal identifiers | ⚠️ Avoid unless essential |
| Private data | ❌ Never collect |
| Sensitive data | ❌ Never collect |

### 5.2 Purpose Limitation

```
✅ Collect data for stated purpose only
✅ Delete data when purpose is fulfilled
✅ Document data retention policies
❌ Don't repurpose data without consent
❌ Don't share data without authorization
```

### 5.3 Transparency

- Maintain a crawler information page
- Provide contact information
- Explain data usage clearly
- Honor removal requests promptly

---

## 6. Terms of Service Compliance

### 6.1 TOS Review Checklist

Before crawling any site, check for:

- [ ] Explicit prohibition of automated access
- [ ] Data usage restrictions
- [ ] API availability (preferred over scraping)
- [ ] Licensing requirements
- [ ] Geographic restrictions

### 6.2 Common TOS Restrictions

```
❌ "No automated data collection"
❌ "Bots, spiders, and crawlers prohibited"
❌ "No scraping without written consent"
❌ "Commercial use requires license"
```

### 6.3 Response to TOS Violations

If you discover your crawling violates TOS:

1. **Stop immediately**
2. **Delete collected data**
3. **Seek permission** if you want to continue
4. **Document the decision**

---

## 7. Crawler Information Page Template

Create a page explaining your crawler:

```markdown
# About MyCrawler

## What is MyCrawler?
MyCrawler is an automated tool that collects [specific data type]
from public websites for [specific purpose].

## How We Crawl
- We respect robots.txt directives
- We limit requests to [X] per second
- We identify ourselves clearly in User-Agent

## What We Collect
- [List specific data types]
- We do NOT collect personal information

## Contact Us
- Email: crawler@example.com
- Report issues: https://example.com/crawler-issues

## Opt-Out
To exclude your site from our crawler:
1. Add to robots.txt: User-agent: MyCrawler / Disallow: /
2. Contact us at crawler@example.com
```

---

## 8. International Considerations

### 8.1 Jurisdiction-Specific Rules

| Region | Key Considerations |
|--------|-------------------|
| **EU** | GDPR compliance required for personal data |
| **USA** | CFAA, DMCA, state-specific laws |
| **Korea** | Copyright Act, PIPA, Unfair Competition Act |
| **China** | Cybersecurity Law, data localization |

### 8.2 Cross-Border Data Transfers

When crawling internationally:
- Understand local data protection laws
- Implement appropriate safeguards
- Consider data residency requirements

---

## 9. Policy Implementation Checklist

### Before Crawling
- [ ] Review target site's robots.txt
- [ ] Check Terms of Service
- [ ] Set appropriate User-Agent
- [ ] Configure rate limiting
- [ ] Prepare crawler information page

### During Crawling
- [ ] Monitor response times
- [ ] Track error rates
- [ ] Respect crawl-delay changes
- [ ] Log compliance metrics

### After Crawling
- [ ] Delete unnecessary data
- [ ] Document data retention
- [ ] Regular compliance audits

---

## References

- [Robots.txt Scraping Compliance Guide](https://www.promptcloud.com/blog/robots-txt-scraping-compliance-guide/)
- [Web Scraping Ethics Guide](https://medium.com/@ridhopujiono.work/web-scraping-2-ethics-legality-robots-txt-how-to-stay-out-of-trouble-39052f7dc63f)
- [Google's robots.txt Specification](https://developers.google.com/crawling/docs/robots-txt/robots-txt-spec)
- [Ethical Web Scraping Guide](https://finddatalab.com/ethicalscraping)

---

*Ethical crawling builds trust and ensures long-term sustainability of web scraping activities.*
