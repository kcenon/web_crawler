"""Asynchronous gRPC client for the Web Crawler SDK."""

from __future__ import annotations

from collections.abc import AsyncIterator
from typing import TYPE_CHECKING

import grpc.aio

from crawler._converters import (
    crawl_config_to_pb,
    crawl_result_from_pb,
    crawl_stats_from_pb,
)
from crawler.client import _convert_rpc_error
from crawler.exceptions import CrawlerError
from crawler.v1 import crawler_pb2_grpc, types_pb2 as pb
from crawler.types import CrawlConfig, CrawlResult, CrawlStats

if TYPE_CHECKING:
    from types import TracebackType


class AsyncCrawlerClient:
    """Asynchronous client for the Web Crawler gRPC service.

    Supports async context manager protocol.

    Example::

        async with AsyncCrawlerClient() as client:
            result = await client.crawl("https://example.com")
            print(result.content)
    """

    def __init__(
        self, host: str = "localhost", port: int = 50051
    ) -> None:
        self._target = f"{host}:{port}"
        self._channel: grpc.aio.Channel | None = None
        self._stub: crawler_pb2_grpc.CrawlerServiceStub | None = None

    def _ensure_connected(self) -> crawler_pb2_grpc.CrawlerServiceStub:
        if self._stub is None:
            self._channel = grpc.aio.insecure_channel(self._target)
            self._stub = crawler_pb2_grpc.CrawlerServiceStub(self._channel)
        return self._stub

    async def crawl(self, url: str, **kwargs: object) -> CrawlResult:
        """Crawl a single URL asynchronously.

        Args:
            url: The URL to crawl.
            **kwargs: Additional options.

        Returns:
            CrawlResult with the crawled page data.
        """
        stub = self._ensure_connected()
        request = pb.CrawlRequest(urls=[url])

        try:
            response = await stub.Crawl(request)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

        return crawl_result_from_pb(response)

    async def crawl_stream(
        self, urls: list[str], **kwargs: object
    ) -> AsyncIterator[CrawlResult]:
        """Stream crawl results as they complete.

        Args:
            urls: URLs to crawl.
            **kwargs: Additional options.

        Yields:
            CrawlResult for each crawled page.
        """
        stub = self._ensure_connected()

        async def _request_generator() -> AsyncIterator[pb.CrawlRequest]:
            for url in urls:
                yield pb.CrawlRequest(urls=[url])

        try:
            stream = stub.CrawlStream(_request_generator())
            async for response in stream:
                yield crawl_result_from_pb(response)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

    async def start(
        self, config: CrawlConfig, crawler_id: str = ""
    ) -> str:
        """Start a long-running crawler asynchronously.

        Args:
            config: Crawl configuration.
            crawler_id: Optional identifier.

        Returns:
            The server-assigned crawler ID.
        """
        stub = self._ensure_connected()
        request = pb.StartCrawlerRequest(
            config=crawl_config_to_pb(config),
            crawler_id=crawler_id,
        )

        try:
            response = await stub.StartCrawler(request)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

        return response.crawler_id

    async def stop(self, crawler_id: str) -> CrawlStats:
        """Stop a running crawler asynchronously.

        Args:
            crawler_id: Identifier of the crawler.

        Returns:
            Final CrawlStats.
        """
        stub = self._ensure_connected()
        request = pb.StopCrawlerRequest(crawler_id=crawler_id)

        try:
            response = await stub.StopCrawler(request)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

        return crawl_stats_from_pb(response.final_stats)

    async def add_urls(self, crawler_id: str, urls: list[str]) -> int:
        """Add URLs to a running crawler asynchronously.

        Args:
            crawler_id: Identifier of the running crawler.
            urls: URLs to add.

        Returns:
            Number of URLs added.
        """
        stub = self._ensure_connected()
        request = pb.AddURLsRequest(crawler_id=crawler_id, urls=urls)

        try:
            response = await stub.AddURLs(request)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

        return response.added_count

    async def stats(self, crawler_id: str) -> CrawlStats:
        """Get crawler statistics asynchronously.

        Args:
            crawler_id: Identifier of the crawler.

        Returns:
            Current CrawlStats.
        """
        stub = self._ensure_connected()
        request = pb.GetStatsRequest(crawler_id=crawler_id)

        try:
            response = await stub.GetStats(request)
        except grpc.aio.AioRpcError as e:
            raise _convert_rpc_error(e) from e

        return crawl_stats_from_pb(response.stats)

    async def close(self) -> None:
        """Close the async gRPC channel."""
        if self._channel is not None:
            await self._channel.close()
            self._channel = None
            self._stub = None

    async def __aenter__(self) -> AsyncCrawlerClient:
        self._ensure_connected()
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        await self.close()
