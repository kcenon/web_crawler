"""Convert between Python types and protobuf messages."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

from google.protobuf.duration_pb2 import Duration
from google.protobuf.timestamp_pb2 import Timestamp

from crawler.v1 import types_pb2 as pb
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

_EXTRACTION_TYPE_TO_PB: dict[ExtractionType, int] = {
    ExtractionType.CSS: pb.EXTRACTION_TYPE_CSS,
    ExtractionType.XPATH: pb.EXTRACTION_TYPE_XPATH,
    ExtractionType.REGEX: pb.EXTRACTION_TYPE_REGEX,
    ExtractionType.JSON_PATH: pb.EXTRACTION_TYPE_JSON_PATH,
}

_PB_TO_EXTRACTION_TYPE: dict[int, ExtractionType] = {
    v: k for k, v in _EXTRACTION_TYPE_TO_PB.items()
}

_PB_TO_CRAWLER_STATUS: dict[int, CrawlerStatus] = {
    pb.CRAWLER_STATUS_IDLE: CrawlerStatus.IDLE,
    pb.CRAWLER_STATUS_RUNNING: CrawlerStatus.RUNNING,
    pb.CRAWLER_STATUS_PAUSED: CrawlerStatus.PAUSED,
    pb.CRAWLER_STATUS_STOPPED: CrawlerStatus.STOPPED,
}

_ERROR_CODE_NAMES: dict[int, str] = {
    pb.ERROR_CODE_UNSPECIFIED: "unspecified",
    pb.ERROR_CODE_NETWORK: "network",
    pb.ERROR_CODE_TIMEOUT: "timeout",
    pb.ERROR_CODE_RATE_LIMITED: "rate_limited",
    pb.ERROR_CODE_BLOCKED: "blocked",
    pb.ERROR_CODE_PARSE_FAILED: "parse_failed",
    pb.ERROR_CODE_INVALID_URL: "invalid_url",
    pb.ERROR_CODE_INTERNAL: "internal",
}


def _timedelta_to_duration(td: timedelta) -> Duration:
    total_seconds = int(td.total_seconds())
    nanos = int((td.total_seconds() - total_seconds) * 1e9)
    d = Duration()
    d.seconds = total_seconds
    d.nanos = nanos
    return d


def _duration_to_timedelta(d: Duration) -> timedelta:
    return timedelta(seconds=d.seconds, microseconds=d.nanos // 1000)


def _timestamp_to_datetime(ts: Timestamp) -> datetime:
    return datetime.fromtimestamp(
        ts.seconds + ts.nanos / 1e9, tz=timezone.utc
    )


def extraction_rule_to_pb(rule: ExtractionRule) -> pb.ExtractionRule:
    return pb.ExtractionRule(
        name=rule.name,
        type=_EXTRACTION_TYPE_TO_PB.get(
            rule.type, pb.EXTRACTION_TYPE_UNSPECIFIED
        ),
        pattern=rule.pattern,
        multiple=rule.multiple,
    )


def middleware_config_to_pb(cfg: MiddlewareConfig) -> pb.MiddlewareConfig:
    return pb.MiddlewareConfig(
        name=cfg.name,
        settings=cfg.settings,
        priority=cfg.priority,
    )


def crawl_options_to_pb(opts: CrawlOptions) -> pb.CrawlOptions:
    pb_opts = pb.CrawlOptions(
        max_depth=opts.max_depth,
        max_pages=opts.max_pages,
        respect_robots_txt=opts.respect_robots_txt,
        headers=opts.headers,
    )
    if opts.request_delay is not None:
        pb_opts.request_delay.CopyFrom(
            _timedelta_to_duration(opts.request_delay)
        )
    if opts.timeout is not None:
        pb_opts.timeout.CopyFrom(_timedelta_to_duration(opts.timeout))
    return pb_opts


def crawl_config_to_pb(config: CrawlConfig) -> pb.CrawlConfig:
    pb_cfg = pb.CrawlConfig(urls=config.urls)
    if config.options is not None:
        pb_cfg.options.CopyFrom(crawl_options_to_pb(config.options))
    for rule in config.extraction_rules:
        pb_cfg.extraction_rules.append(extraction_rule_to_pb(rule))
    for mw in config.middlewares:
        pb_cfg.middlewares.append(middleware_config_to_pb(mw))
    return pb_cfg


def crawl_result_from_pb(resp: pb.CrawlResponse) -> CrawlResult:
    extractions = [
        ExtractionResult(name=e.name, values=list(e.values))
        for e in resp.extractions
    ]

    error = None
    if resp.HasField("error"):
        error = ErrorInfo(
            code=_ERROR_CODE_NAMES.get(resp.error.code, "unknown"),
            message=resp.error.message,
            url=resp.error.url,
            retryable=resp.error.retryable,
        )

    crawled_at = None
    if resp.HasField("crawled_at"):
        crawled_at = _timestamp_to_datetime(resp.crawled_at)

    duration = None
    if resp.HasField("duration"):
        duration = _duration_to_timedelta(resp.duration)

    return CrawlResult(
        url=resp.url,
        status_code=resp.status_code,
        content=resp.content,
        extractions=extractions,
        error=error,
        crawled_at=crawled_at,
        duration=duration,
    )


def crawl_stats_from_pb(stats: pb.CrawlStats) -> CrawlStats:
    avg_latency = None
    if stats.HasField("avg_latency"):
        avg_latency = _duration_to_timedelta(stats.avg_latency)

    started_at = None
    if stats.HasField("started_at"):
        started_at = _timestamp_to_datetime(stats.started_at)

    status = _PB_TO_CRAWLER_STATUS.get(stats.status, CrawlerStatus.IDLE)

    return CrawlStats(
        pages_crawled=stats.pages_crawled,
        pages_failed=stats.pages_failed,
        pages_queued=stats.pages_queued,
        avg_latency=avg_latency,
        started_at=started_at,
        status=status,
    )
