"""Exception hierarchy for the Web Crawler SDK."""

from __future__ import annotations


class CrawlerError(Exception):
    """Base exception for all crawler SDK errors."""


class ConnectionError(CrawlerError):
    """Raised when the client cannot connect to the gRPC server."""


class CrawlError(CrawlerError):
    """Raised when a crawl operation fails."""

    def __init__(
        self,
        message: str,
        *,
        code: str = "",
        url: str = "",
        retryable: bool = False,
    ) -> None:
        super().__init__(message)
        self.code = code
        self.url = url
        self.retryable = retryable


class NotFoundError(CrawlerError):
    """Raised when a crawler instance is not found."""


class AlreadyRunningError(CrawlerError):
    """Raised when attempting to start a crawler that is already running."""
