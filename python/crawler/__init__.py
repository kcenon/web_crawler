"""Web Crawler SDK Python bindings.

Provides synchronous and asynchronous Python clients for the
Web Crawler gRPC server.

Quick start::

    from crawler import CrawlerClient

    with CrawlerClient() as client:
        result = client.crawl("https://example.com")
        print(result.status_code)
"""

from crawler.async_client import AsyncCrawlerClient
from crawler.client import CrawlerClient
from crawler.exceptions import (
    AlreadyRunningError,
    ConnectionError,
    CrawlError,
    CrawlerError,
    NotFoundError,
)
from crawler.types import (
    CrawlConfig,
    CrawlOptions,
    CrawlResult,
    CrawlStats,
    CrawlerStatus,
    ErrorInfo,
    ExtractionResult,
    ExtractionRule,
    ExtractionType,
    MiddlewareConfig,
)

__all__ = [
    # Clients
    "CrawlerClient",
    "AsyncCrawlerClient",
    # Types
    "CrawlConfig",
    "CrawlOptions",
    "CrawlResult",
    "CrawlStats",
    "CrawlerStatus",
    "ErrorInfo",
    "ExtractionResult",
    "ExtractionRule",
    "ExtractionType",
    "MiddlewareConfig",
    # Exceptions
    "CrawlerError",
    "ConnectionError",
    "CrawlError",
    "NotFoundError",
    "AlreadyRunningError",
]
