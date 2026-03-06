"""Tests for type definitions."""

from datetime import timedelta

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


def test_extraction_rule_defaults() -> None:
    rule = ExtractionRule(
        name="title", type=ExtractionType.CSS, pattern="h1"
    )
    assert rule.name == "title"
    assert rule.type == ExtractionType.CSS
    assert rule.pattern == "h1"
    assert rule.multiple is False


def test_extraction_result() -> None:
    result = ExtractionResult(name="links", values=["/a", "/b"])
    assert result.name == "links"
    assert len(result.values) == 2


def test_crawl_options_defaults() -> None:
    opts = CrawlOptions()
    assert opts.max_depth == 0
    assert opts.max_pages == 0
    assert opts.request_delay is None
    assert opts.respect_robots_txt is True
    assert opts.headers == {}


def test_crawl_options_with_values() -> None:
    opts = CrawlOptions(
        max_depth=3,
        max_pages=100,
        request_delay=timedelta(seconds=1),
        timeout=timedelta(minutes=5),
        headers={"User-Agent": "TestBot"},
    )
    assert opts.max_depth == 3
    assert opts.timeout == timedelta(minutes=5)
    assert opts.headers["User-Agent"] == "TestBot"


def test_middleware_config() -> None:
    cfg = MiddlewareConfig(
        name="retry",
        settings={"max_retries": "3"},
        priority=10,
    )
    assert cfg.name == "retry"
    assert cfg.settings["max_retries"] == "3"
    assert cfg.priority == 10


def test_crawl_config() -> None:
    config = CrawlConfig(
        urls=["https://example.com"],
        options=CrawlOptions(max_depth=2),
        extraction_rules=[
            ExtractionRule(
                name="title",
                type=ExtractionType.CSS,
                pattern="h1",
            )
        ],
    )
    assert len(config.urls) == 1
    assert config.options is not None
    assert config.options.max_depth == 2
    assert len(config.extraction_rules) == 1


def test_error_info() -> None:
    err = ErrorInfo(
        code="network",
        message="connection refused",
        url="https://example.com",
        retryable=True,
    )
    assert err.code == "network"
    assert err.retryable is True


def test_crawl_result_defaults() -> None:
    result = CrawlResult(url="https://example.com")
    assert result.url == "https://example.com"
    assert result.status_code == 0
    assert result.content == ""
    assert result.extractions == []
    assert result.error is None
    assert result.crawled_at is None
    assert result.duration is None


def test_crawl_stats_defaults() -> None:
    stats = CrawlStats()
    assert stats.pages_crawled == 0
    assert stats.pages_failed == 0
    assert stats.pages_queued == 0
    assert stats.status == CrawlerStatus.IDLE


def test_crawler_status_values() -> None:
    assert CrawlerStatus.IDLE.value == "idle"
    assert CrawlerStatus.RUNNING.value == "running"
    assert CrawlerStatus.PAUSED.value == "paused"
    assert CrawlerStatus.STOPPED.value == "stopped"


def test_extraction_type_values() -> None:
    assert ExtractionType.CSS.value == "css"
    assert ExtractionType.XPATH.value == "xpath"
    assert ExtractionType.REGEX.value == "regex"
    assert ExtractionType.JSON_PATH.value == "json_path"
