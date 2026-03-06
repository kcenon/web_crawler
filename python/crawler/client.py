"""Synchronous gRPC client for the Web Crawler SDK."""

from __future__ import annotations

from typing import TYPE_CHECKING

import grpc

from crawler._converters import (
    crawl_config_to_pb,
    crawl_result_from_pb,
    crawl_stats_from_pb,
)
from crawler.exceptions import ConnectionError, CrawlError
from crawler.v1 import crawler_pb2_grpc, types_pb2 as pb
from crawler.types import CrawlConfig, CrawlResult, CrawlStats

if TYPE_CHECKING:
    from types import TracebackType


class CrawlerClient:
    """Synchronous client for the Web Crawler gRPC service.

    Supports context manager protocol for automatic resource cleanup.

    Example::

        with CrawlerClient() as client:
            result = client.crawl("https://example.com")
            print(result.content)
    """

    def __init__(
        self, host: str = "localhost", port: int = 50051
    ) -> None:
        self._target = f"{host}:{port}"
        self._channel: grpc.Channel | None = None
        self._stub: crawler_pb2_grpc.CrawlerServiceStub | None = None

    def _ensure_connected(self) -> crawler_pb2_grpc.CrawlerServiceStub:
        if self._stub is None:
            self._channel = grpc.insecure_channel(self._target)
            self._stub = crawler_pb2_grpc.CrawlerServiceStub(self._channel)
        return self._stub

    def crawl(self, url: str, **kwargs: object) -> CrawlResult:
        """Crawl a single URL and return the result.

        Args:
            url: The URL to crawl.
            **kwargs: Additional options passed to CrawlConfig.

        Returns:
            CrawlResult with the crawled page data.
        """
        results = self.crawl_many([url], **kwargs)
        return results[0]

    def crawl_many(
        self, urls: list[str], **kwargs: object
    ) -> list[CrawlResult]:
        """Crawl multiple URLs and return all results.

        Args:
            urls: List of URLs to crawl.
            **kwargs: Additional options passed to CrawlConfig.

        Returns:
            List of CrawlResult, one per URL.
        """
        stub = self._ensure_connected()

        config = None
        if kwargs:
            config = pb.CrawlConfig(urls=urls)

        request = pb.CrawlRequest(urls=urls)
        if config is not None:
            request.config.CopyFrom(config)

        try:
            response = stub.Crawl(request)
        except grpc.RpcError as e:
            raise _convert_rpc_error(e) from e

        return [crawl_result_from_pb(response)]

    def start(self, config: CrawlConfig, crawler_id: str = "") -> str:
        """Start a long-running crawler.

        Args:
            config: Crawl configuration.
            crawler_id: Optional identifier for the crawler instance.

        Returns:
            The server-assigned crawler ID.
        """
        stub = self._ensure_connected()

        request = pb.StartCrawlerRequest(
            config=crawl_config_to_pb(config),
            crawler_id=crawler_id,
        )

        try:
            response = stub.StartCrawler(request)
        except grpc.RpcError as e:
            raise _convert_rpc_error(e) from e

        return response.crawler_id

    def stop(self, crawler_id: str) -> CrawlStats:
        """Stop a running crawler and return final statistics.

        Args:
            crawler_id: Identifier of the crawler to stop.

        Returns:
            Final CrawlStats snapshot.
        """
        stub = self._ensure_connected()

        request = pb.StopCrawlerRequest(crawler_id=crawler_id)

        try:
            response = stub.StopCrawler(request)
        except grpc.RpcError as e:
            raise _convert_rpc_error(e) from e

        return crawl_stats_from_pb(response.final_stats)

    def add_urls(self, crawler_id: str, urls: list[str]) -> int:
        """Add URLs to a running crawler's frontier.

        Args:
            crawler_id: Identifier of the running crawler.
            urls: URLs to add.

        Returns:
            Number of URLs actually added (duplicates excluded).
        """
        stub = self._ensure_connected()

        request = pb.AddURLsRequest(crawler_id=crawler_id, urls=urls)

        try:
            response = stub.AddURLs(request)
        except grpc.RpcError as e:
            raise _convert_rpc_error(e) from e

        return response.added_count

    def stats(self, crawler_id: str) -> CrawlStats:
        """Get runtime statistics for a running crawler.

        Args:
            crawler_id: Identifier of the crawler.

        Returns:
            Current CrawlStats.
        """
        stub = self._ensure_connected()

        request = pb.GetStatsRequest(crawler_id=crawler_id)

        try:
            response = stub.GetStats(request)
        except grpc.RpcError as e:
            raise _convert_rpc_error(e) from e

        return crawl_stats_from_pb(response.stats)

    def close(self) -> None:
        """Close the gRPC channel."""
        if self._channel is not None:
            self._channel.close()
            self._channel = None
            self._stub = None

    def __enter__(self) -> CrawlerClient:
        self._ensure_connected()
        return self

    def __exit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        self.close()


def _convert_rpc_error(error: grpc.RpcError) -> CrawlerError:
    """Convert a gRPC error to a SDK exception."""
    code = error.code()  # type: ignore[union-attr]
    details = error.details()  # type: ignore[union-attr]

    if code == grpc.StatusCode.UNAVAILABLE:
        return ConnectionError(f"Server unavailable: {details}")
    if code == grpc.StatusCode.NOT_FOUND:
        from crawler.exceptions import NotFoundError

        return NotFoundError(f"Not found: {details}")

    return CrawlError(
        f"gRPC error ({code.name}): {details}",
        code=code.name,
    )
