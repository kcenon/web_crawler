"""Tests for protobuf <-> Python type converters."""

from datetime import datetime, timedelta, timezone

from google.protobuf.duration_pb2 import Duration
from google.protobuf.timestamp_pb2 import Timestamp

from crawler._converters import (
    crawl_config_to_pb,
    crawl_result_from_pb,
    crawl_stats_from_pb,
)
from crawler.v1 import types_pb2 as pb
from crawler.types import (
    CrawlConfig,
    CrawlOptions,
    CrawlerStatus,
    ExtractionRule,
    ExtractionType,
    MiddlewareConfig,
)


def test_crawl_config_to_pb_basic() -> None:
    config = CrawlConfig(urls=["https://example.com"])
    pb_cfg = crawl_config_to_pb(config)
    assert list(pb_cfg.urls) == ["https://example.com"]


def test_crawl_config_to_pb_with_options() -> None:
    config = CrawlConfig(
        urls=["https://example.com"],
        options=CrawlOptions(
            max_depth=3,
            max_pages=100,
            request_delay=timedelta(seconds=2),
            timeout=timedelta(minutes=5),
            respect_robots_txt=False,
            headers={"User-Agent": "TestBot"},
        ),
    )
    pb_cfg = crawl_config_to_pb(config)
    assert pb_cfg.options.max_depth == 3
    assert pb_cfg.options.max_pages == 100
    assert pb_cfg.options.request_delay.seconds == 2
    assert pb_cfg.options.timeout.seconds == 300
    assert pb_cfg.options.respect_robots_txt is False
    assert pb_cfg.options.headers["User-Agent"] == "TestBot"


def test_crawl_config_to_pb_with_rules() -> None:
    config = CrawlConfig(
        urls=["https://example.com"],
        extraction_rules=[
            ExtractionRule(
                name="title",
                type=ExtractionType.CSS,
                pattern="h1",
                multiple=True,
            ),
        ],
        middlewares=[
            MiddlewareConfig(
                name="retry",
                settings={"max_retries": "3"},
                priority=10,
            ),
        ],
    )
    pb_cfg = crawl_config_to_pb(config)
    assert len(pb_cfg.extraction_rules) == 1
    assert pb_cfg.extraction_rules[0].name == "title"
    assert pb_cfg.extraction_rules[0].type == pb.EXTRACTION_TYPE_CSS
    assert pb_cfg.extraction_rules[0].multiple is True
    assert len(pb_cfg.middlewares) == 1
    assert pb_cfg.middlewares[0].name == "retry"


def test_crawl_result_from_pb_basic() -> None:
    response = pb.CrawlResponse(
        url="https://example.com",
        status_code=200,
        content="<html></html>",
    )
    result = crawl_result_from_pb(response)
    assert result.url == "https://example.com"
    assert result.status_code == 200
    assert result.content == "<html></html>"
    assert result.error is None
    assert result.crawled_at is None


def test_crawl_result_from_pb_with_error() -> None:
    response = pb.CrawlResponse(
        url="https://example.com",
        error=pb.ErrorInfo(
            code=pb.ERROR_CODE_NETWORK,
            message="connection refused",
            url="https://example.com",
            retryable=True,
        ),
    )
    result = crawl_result_from_pb(response)
    assert result.error is not None
    assert result.error.code == "network"
    assert result.error.message == "connection refused"
    assert result.error.retryable is True


def test_crawl_result_from_pb_with_timestamp() -> None:
    ts = Timestamp()
    ts.FromDatetime(datetime(2025, 1, 15, 12, 0, 0, tzinfo=timezone.utc))
    dur = Duration()
    dur.FromTimedelta(timedelta(milliseconds=500))

    response = pb.CrawlResponse(
        url="https://example.com",
        status_code=200,
        crawled_at=ts,
        duration=dur,
    )
    result = crawl_result_from_pb(response)
    assert result.crawled_at is not None
    assert result.crawled_at.year == 2025
    assert result.duration is not None
    assert result.duration.total_seconds() == 0.5


def test_crawl_result_from_pb_with_extractions() -> None:
    response = pb.CrawlResponse(
        url="https://example.com",
        status_code=200,
        extractions=[
            pb.ExtractionResult(name="title", values=["Hello"]),
            pb.ExtractionResult(name="links", values=["/a", "/b"]),
        ],
    )
    result = crawl_result_from_pb(response)
    assert len(result.extractions) == 2
    assert result.extractions[0].name == "title"
    assert result.extractions[0].values == ["Hello"]
    assert result.extractions[1].values == ["/a", "/b"]


def test_crawl_stats_from_pb() -> None:
    ts = Timestamp()
    ts.FromDatetime(datetime(2025, 1, 15, 12, 0, 0, tzinfo=timezone.utc))
    dur = Duration()
    dur.FromTimedelta(timedelta(milliseconds=150))

    stats = pb.CrawlStats(
        pages_crawled=100,
        pages_failed=5,
        pages_queued=50,
        avg_latency=dur,
        started_at=ts,
        status=pb.CRAWLER_STATUS_RUNNING,
    )
    result = crawl_stats_from_pb(stats)
    assert result.pages_crawled == 100
    assert result.pages_failed == 5
    assert result.pages_queued == 50
    assert result.avg_latency is not None
    assert result.avg_latency.total_seconds() == 0.15
    assert result.started_at is not None
    assert result.status == CrawlerStatus.RUNNING
