"""Tests for the exception hierarchy."""

from crawler.exceptions import (
    AlreadyRunningError,
    ConnectionError,
    CrawlError,
    CrawlerError,
    NotFoundError,
)


def test_exception_hierarchy() -> None:
    assert issubclass(ConnectionError, CrawlerError)
    assert issubclass(CrawlError, CrawlerError)
    assert issubclass(NotFoundError, CrawlerError)
    assert issubclass(AlreadyRunningError, CrawlerError)


def test_crawl_error_fields() -> None:
    err = CrawlError(
        "test error",
        code="network",
        url="https://example.com",
        retryable=True,
    )
    assert str(err) == "test error"
    assert err.code == "network"
    assert err.url == "https://example.com"
    assert err.retryable is True


def test_crawl_error_defaults() -> None:
    err = CrawlError("simple error")
    assert err.code == ""
    assert err.url == ""
    assert err.retryable is False


def test_exceptions_catchable_as_base() -> None:
    try:
        raise ConnectionError("cannot connect")
    except CrawlerError as e:
        assert "cannot connect" in str(e)
