"""Type definitions for the Web Crawler SDK."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta
from enum import Enum


class ExtractionType(Enum):
    """Strategy used to extract data from a page."""

    CSS = "css"
    XPATH = "xpath"
    REGEX = "regex"
    JSON_PATH = "json_path"


class CrawlerStatus(Enum):
    """Lifecycle state of a crawler instance."""

    IDLE = "idle"
    RUNNING = "running"
    PAUSED = "paused"
    STOPPED = "stopped"


@dataclass
class ExtractionRule:
    """Defines how to extract a named field from page content."""

    name: str
    type: ExtractionType
    pattern: str
    multiple: bool = False


@dataclass
class ExtractionResult:
    """Output of a single extraction rule."""

    name: str
    values: list[str] = field(default_factory=list)


@dataclass
class CrawlOptions:
    """Per-request crawl parameters."""

    max_depth: int = 0
    max_pages: int = 0
    request_delay: timedelta | None = None
    timeout: timedelta | None = None
    respect_robots_txt: bool = True
    headers: dict[str, str] = field(default_factory=dict)


@dataclass
class MiddlewareConfig:
    """Configuration for a single middleware in the pipeline."""

    name: str
    settings: dict[str, str] = field(default_factory=dict)
    priority: int = 0


@dataclass
class CrawlConfig:
    """Full configuration for a crawl job."""

    urls: list[str] = field(default_factory=list)
    options: CrawlOptions | None = None
    extraction_rules: list[ExtractionRule] = field(default_factory=list)
    middlewares: list[MiddlewareConfig] = field(default_factory=list)


@dataclass
class ErrorInfo:
    """Structured error details attached to a crawl result."""

    code: str
    message: str
    url: str = ""
    retryable: bool = False


@dataclass
class CrawlResult:
    """Result from crawling a single page."""

    url: str
    status_code: int = 0
    content: str = ""
    extractions: list[ExtractionResult] = field(default_factory=list)
    error: ErrorInfo | None = None
    crawled_at: datetime | None = None
    duration: timedelta | None = None


@dataclass
class CrawlStats:
    """Runtime statistics for a crawler instance."""

    pages_crawled: int = 0
    pages_failed: int = 0
    pages_queued: int = 0
    avg_latency: timedelta | None = None
    started_at: datetime | None = None
    status: CrawlerStatus = CrawlerStatus.IDLE
