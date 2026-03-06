"""Tests for the synchronous and async gRPC clients."""

from __future__ import annotations

from concurrent import futures
from unittest.mock import MagicMock

import grpc
import pytest

from crawler.client import CrawlerClient
from crawler.exceptions import ConnectionError, CrawlerError
from crawler.v1 import crawler_pb2_grpc, types_pb2 as pb
from crawler.types import CrawlConfig, CrawlOptions, CrawlerStatus


class FakeCrawlerServicer(crawler_pb2_grpc.CrawlerServiceServicer):
    """In-process fake gRPC servicer for testing."""

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
            crawler_id=request.crawler_id or "test-crawler-1",
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
                pages_crawled=42,
                pages_failed=3,
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
                pages_crawled=10,
                pages_failed=1,
                pages_queued=5,
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
    """Start an in-process gRPC server and return (server, port)."""
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    crawler_pb2_grpc.add_CrawlerServiceServicer_to_server(
        FakeCrawlerServicer(), server
    )
    port = server.add_insecure_port("[::]:0")
    server.start()
    yield server, port  # type: ignore[misc]
    server.stop(grace=0)


def test_crawl_single_url(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        result = client.crawl("https://example.com")
    assert result.url == "https://example.com"
    assert result.status_code == 200
    assert "example.com" in result.content


def test_crawl_many(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        results = client.crawl_many(["https://example.com"])
    assert len(results) == 1
    assert results[0].status_code == 200


def test_start_and_stop(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        config = CrawlConfig(
            urls=["https://example.com"],
            options=CrawlOptions(max_depth=2),
        )
        crawler_id = client.start(config)
        assert crawler_id == "test-crawler-1"

        stats = client.stop(crawler_id)
        assert stats.pages_crawled == 42
        assert stats.pages_failed == 3
        assert stats.status == CrawlerStatus.STOPPED


def test_get_stats(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        stats = client.stats("test-crawler-1")
    assert stats.pages_crawled == 10
    assert stats.pages_queued == 5
    assert stats.status == CrawlerStatus.RUNNING


def test_add_urls(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        added = client.add_urls(
            "test-crawler-1",
            ["https://a.com", "https://b.com"],
        )
    assert added == 2


def test_context_manager_closes_channel(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    client = CrawlerClient(host="localhost", port=port)
    with client:
        assert client._channel is not None
    assert client._channel is None


def test_connection_error_on_unavailable() -> None:
    with CrawlerClient(host="localhost", port=1) as client:
        with pytest.raises(CrawlerError):
            client.crawl("https://example.com")


def test_close_idempotent() -> None:
    client = CrawlerClient()
    client.close()
    client.close()  # Should not raise.


def test_start_with_custom_id(
    grpc_server: tuple[grpc.Server, int],
) -> None:
    _, port = grpc_server
    with CrawlerClient(host="localhost", port=port) as client:
        config = CrawlConfig(urls=["https://example.com"])
        crawler_id = client.start(config, crawler_id="my-crawler")
    assert crawler_id == "my-crawler"
