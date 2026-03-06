"""Tests for the async gRPC client."""

from __future__ import annotations

import asyncio
from concurrent import futures

import grpc
import pytest

from crawler.async_client import AsyncCrawlerClient
from crawler.v1 import crawler_pb2_grpc, types_pb2 as pb
from crawler.types import CrawlConfig, CrawlOptions, CrawlerStatus


class FakeAsyncCrawlerServicer(
    crawler_pb2_grpc.CrawlerServiceServicer
):
    """Fake gRPC servicer for async client tests."""

    def Crawl(
        self, request: pb.CrawlRequest, context: grpc.ServicerContext
    ) -> pb.CrawlResponse:
        url = request.urls[0] if request.urls else ""
        return pb.CrawlResponse(
            url=url,
            status_code=200,
            content=f"<html>{url}</html>",
        )

    def StartCrawler(
        self,
        request: pb.StartCrawlerRequest,
        context: grpc.ServicerContext,
    ) -> pb.StartCrawlerResponse:
        return pb.StartCrawlerResponse(
            crawler_id=request.crawler_id or "async-crawler-1",
            status=pb.CRAWLER_STATUS_RUNNING,
        )

    def StopCrawler(
        self,
        request: pb.StopCrawlerRequest,
        context: grpc.ServicerContext,
    ) -> pb.StopCrawlerResponse:
        return pb.StopCrawlerResponse(
            crawler_id=request.crawler_id,
            status=pb.CRAWLER_STATUS_STOPPED,
            final_stats=pb.CrawlStats(
                pages_crawled=50,
                status=pb.CRAWLER_STATUS_STOPPED,
            ),
        )

    def GetStats(
        self,
        request: pb.GetStatsRequest,
        context: grpc.ServicerContext,
    ) -> pb.GetStatsResponse:
        return pb.GetStatsResponse(
            stats=pb.CrawlStats(
                pages_crawled=25,
                status=pb.CRAWLER_STATUS_RUNNING,
            )
        )

    def AddURLs(
        self,
        request: pb.AddURLsRequest,
        context: grpc.ServicerContext,
    ) -> pb.AddURLsResponse:
        return pb.AddURLsResponse(added_count=len(request.urls))


@pytest.fixture()
def grpc_server() -> tuple[grpc.Server, int]:
    """Start an in-process gRPC server for async tests."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    crawler_pb2_grpc.add_CrawlerServiceServicer_to_server(
        FakeAsyncCrawlerServicer(), server
    )
    port = server.add_insecure_port("[::]:0")
    server.start()
    yield server, port  # type: ignore[misc]
    server.stop(grace=0)


@pytest.mark.asyncio
async def test_async_crawl(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    async with AsyncCrawlerClient(
        host="localhost", port=port
    ) as client:
        result = await client.crawl("https://example.com")
    assert result.url == "https://example.com"
    assert result.status_code == 200


@pytest.mark.asyncio
async def test_async_start_stop(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    async with AsyncCrawlerClient(
        host="localhost", port=port
    ) as client:
        config = CrawlConfig(
            urls=["https://example.com"],
            options=CrawlOptions(max_depth=1),
        )
        crawler_id = await client.start(config)
        assert crawler_id == "async-crawler-1"

        stats = await client.stop(crawler_id)
        assert stats.pages_crawled == 50
        assert stats.status == CrawlerStatus.STOPPED


@pytest.mark.asyncio
async def test_async_stats(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    async with AsyncCrawlerClient(
        host="localhost", port=port
    ) as client:
        stats = await client.stats("test-crawler")
    assert stats.pages_crawled == 25
    assert stats.status == CrawlerStatus.RUNNING


@pytest.mark.asyncio
async def test_async_add_urls(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    async with AsyncCrawlerClient(
        host="localhost", port=port
    ) as client:
        added = await client.add_urls(
            "test-crawler", ["https://a.com"]
        )
    assert added == 1


@pytest.mark.asyncio
async def test_async_context_manager_closes(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    client = AsyncCrawlerClient(host="localhost", port=port)
    async with client:
        assert client._channel is not None
    assert client._channel is None


@pytest.mark.asyncio
async def test_async_close_idempotent() -> None:
    client = AsyncCrawlerClient()
    await client.close()
    await client.close()  # Should not raise.
