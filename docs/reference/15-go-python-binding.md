# Go-Python Binding Guide

> **Version**: 1.0.0
> **Last Updated**: 2026-02-05
> **Strategy**: Go Core + Python Bindings (Strategy C)
> **Purpose**: Implementation guide for connecting Go core with Python SDK

## Overview

This document covers strategies and implementation details for binding the Go core engine to Python, enabling data scientists and Python developers to leverage the high-performance Go crawler through a familiar Pythonic API.

---

## 1. Binding Strategy Comparison

### 1.1 Options Overview

| Method | Performance | Complexity | Deployment | Use Case |
|--------|-------------|------------|------------|----------|
| **gRPC** | High | Medium | Separate processes | Microservices, distributed |
| **CGO** | Highest | High | Single binary | Embedded, serverless |
| **REST API** | Medium | Low | HTTP | Simple integration |
| **Unix Socket** | High | Low | Local only | Single machine, IPC |
| **Shared Memory** | Highest | Very High | Complex | Ultra-low latency |

### 1.2 Recommended Approach

```
Primary: gRPC (microservices, distributed deployments)
Secondary: CGO (single binary, embedded use cases)
Fallback: REST API (simple integration, debugging)
```

**Rationale**:
- gRPC: Best balance of performance and maintainability
- CGO: Maximum performance when needed
- REST: Universal compatibility, easy debugging

---

## 2. gRPC Binding (Primary)

### 2.1 Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Python Application                          │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     crawler_sdk package                       │    │
│  │  • CrawlerClient (sync)    • AsyncCrawlerClient (async)      │    │
│  │  • Type hints              • Pydantic models                  │    │
│  └──────────────────────────────┬────────────────────────────────┘    │
│                                 │                                     │
│                                 │ gRPC (HTTP/2 + Protobuf)            │
│                                 │                                     │
└─────────────────────────────────┼─────────────────────────────────────┘
                                  │
┌─────────────────────────────────▼─────────────────────────────────────┐
│                           Go gRPC Server                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                     CrawlerService                           │    │
│  │  • Crawl()           • CrawlBatch()                         │    │
│  │  • StartJob()        • StopJob()                            │    │
│  └──────────────────────────────┬────────────────────────────────┘    │
│                                 │                                     │
│                                 ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                      Go Core Engine                          │    │
│  └─────────────────────────────────────────────────────────────┘    │
└───────────────────────────────────────────────────────────────────────┘
```

### 2.2 Protocol Buffer Definition

```protobuf
// api/proto/crawler/v1/crawler.proto
syntax = "proto3";

package crawler.v1;

option go_package = "github.com/yourorg/crawler-sdk/api/proto/crawler/v1;crawlerv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";

// CrawlerService provides web crawling capabilities
service CrawlerService {
    // Crawl fetches a single URL
    rpc Crawl(CrawlRequest) returns (CrawlResponse);

    // CrawlBatch fetches multiple URLs with streaming response
    rpc CrawlBatch(CrawlBatchRequest) returns (stream CrawlResponse);

    // CrawlStream accepts streaming URLs and returns streaming responses
    rpc CrawlStream(stream CrawlRequest) returns (stream CrawlResponse);

    // StartJob starts a continuous crawling job
    rpc StartJob(StartJobRequest) returns (StartJobResponse);

    // GetJobStatus returns the status of a crawling job
    rpc GetJobStatus(GetJobStatusRequest) returns (GetJobStatusResponse);

    // StopJob stops a crawling job
    rpc StopJob(StopJobRequest) returns (StopJobResponse);

    // Health returns service health status
    rpc Health(HealthRequest) returns (HealthResponse);
}

// CrawlRequest represents a single crawl request
message CrawlRequest {
    string url = 1;
    CrawlOptions options = 2;
    map<string, string> metadata = 3;
}

// CrawlOptions configures crawl behavior
message CrawlOptions {
    bool render_js = 1;
    google.protobuf.Duration timeout = 2;
    map<string, string> headers = 3;
    string proxy = 4;
    int32 max_retries = 5;
    google.protobuf.Duration retry_delay = 6;
    bool follow_redirects = 7;
    int32 max_redirects = 8;
    string user_agent = 9;
    repeated string cookies = 10;
}

// CrawlResponse contains the crawl result
message CrawlResponse {
    string url = 1;
    string final_url = 2;  // After redirects
    int32 status_code = 3;
    map<string, string> headers = 4;
    bytes content = 5;
    string content_type = 6;
    google.protobuf.Duration fetch_time = 7;
    bool from_cache = 8;
    string error = 9;
    ErrorType error_type = 10;
    map<string, string> metadata = 11;
}

// ErrorType categorizes errors
enum ErrorType {
    ERROR_TYPE_UNSPECIFIED = 0;
    ERROR_TYPE_NETWORK = 1;
    ERROR_TYPE_TIMEOUT = 2;
    ERROR_TYPE_RATE_LIMITED = 3;
    ERROR_TYPE_BLOCKED = 4;
    ERROR_TYPE_ROBOTS_TXT = 5;
    ERROR_TYPE_PARSE = 6;
    ERROR_TYPE_INVALID_URL = 7;
}

// CrawlBatchRequest requests multiple URLs
message CrawlBatchRequest {
    repeated string urls = 1;
    CrawlOptions options = 2;
    int32 concurrency = 3;
}

// StartJobRequest starts a crawling job
message StartJobRequest {
    repeated string seed_urls = 1;
    CrawlOptions options = 2;
    JobConfig config = 3;
}

// JobConfig configures a crawling job
message JobConfig {
    int32 max_pages = 1;
    int32 max_depth = 2;
    repeated string allowed_domains = 3;
    repeated string disallowed_domains = 4;
    repeated string url_patterns = 5;
    double requests_per_second = 6;
    bool respect_robots_txt = 7;
    google.protobuf.Duration job_timeout = 8;
}

// StartJobResponse contains the job ID
message StartJobResponse {
    string job_id = 1;
    string status = 2;
}

// GetJobStatusRequest requests job status
message GetJobStatusRequest {
    string job_id = 1;
}

// GetJobStatusResponse contains job status
message GetJobStatusResponse {
    string job_id = 1;
    JobStatus status = 2;
    JobStats stats = 3;
}

// JobStatus represents job state
enum JobStatus {
    JOB_STATUS_UNSPECIFIED = 0;
    JOB_STATUS_PENDING = 1;
    JOB_STATUS_RUNNING = 2;
    JOB_STATUS_PAUSED = 3;
    JOB_STATUS_COMPLETED = 4;
    JOB_STATUS_FAILED = 5;
    JOB_STATUS_CANCELLED = 6;
}

// JobStats contains job statistics
message JobStats {
    int64 urls_queued = 1;
    int64 urls_crawled = 2;
    int64 urls_successful = 3;
    int64 urls_failed = 4;
    int64 bytes_received = 5;
    google.protobuf.Timestamp started_at = 6;
    google.protobuf.Duration elapsed = 7;
    double requests_per_second = 8;
}

// StopJobRequest stops a job
message StopJobRequest {
    string job_id = 1;
    bool force = 2;  // Force immediate stop
}

// StopJobResponse confirms job stop
message StopJobResponse {
    bool success = 1;
    string message = 2;
}

// HealthRequest requests health status
message HealthRequest {}

// HealthResponse contains health status
message HealthResponse {
    bool healthy = 1;
    string version = 2;
    int64 uptime_seconds = 3;
    map<string, string> components = 4;
}
```

### 2.3 Go Server Implementation

```go
// internal/server/grpc.go
package server

import (
    "context"
    "io"
    "sync"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/durationpb"
    "google.golang.org/protobuf/types/known/timestamppb"

    pb "github.com/yourorg/crawler-sdk/api/proto/crawler/v1"
    "github.com/yourorg/crawler-sdk/internal/engine"
    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

type CrawlerServer struct {
    pb.UnimplementedCrawlerServiceServer

    engine *engine.Engine
    jobs   map[string]*engine.Job
    mu     sync.RWMutex
}

func NewCrawlerServer(eng *engine.Engine) *CrawlerServer {
    return &CrawlerServer{
        engine: eng,
        jobs:   make(map[string]*engine.Job),
    }
}

func (s *CrawlerServer) Crawl(ctx context.Context, req *pb.CrawlRequest) (*pb.CrawlResponse, error) {
    // Convert protobuf request to internal type
    crawlReq := s.toInternalRequest(req)

    // Execute crawl
    result, err := s.engine.Crawl(ctx, crawlReq)
    if err != nil {
        return s.errorResponse(req.Url, err), nil
    }

    return s.toProtoResponse(result), nil
}

func (s *CrawlerServer) CrawlBatch(req *pb.CrawlBatchRequest, stream pb.CrawlerService_CrawlBatchServer) error {
    ctx := stream.Context()

    // Create worker pool
    concurrency := int(req.Concurrency)
    if concurrency <= 0 {
        concurrency = 10
    }

    results := make(chan *crawler.CrawlResult, len(req.Urls))
    var wg sync.WaitGroup

    // Semaphore for concurrency control
    sem := make(chan struct{}, concurrency)

    for _, url := range req.Urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()

            sem <- struct{}{}        // Acquire
            defer func() { <-sem }() // Release

            crawlReq := &crawler.Request{
                URL:     u,
                Options: s.toInternalOptions(req.Options),
            }

            result, err := s.engine.Crawl(ctx, crawlReq)
            if err != nil {
                results <- &crawler.CrawlResult{URL: u, Error: err}
                return
            }
            results <- result
        }(url)
    }

    // Close results channel when all done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Stream results
    for result := range results {
        resp := s.toProtoResponse(result)
        if err := stream.Send(resp); err != nil {
            return err
        }
    }

    return nil
}

func (s *CrawlerServer) CrawlStream(stream pb.CrawlerService_CrawlStreamServer) error {
    ctx := stream.Context()

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        crawlReq := s.toInternalRequest(req)
        result, err := s.engine.Crawl(ctx, crawlReq)

        var resp *pb.CrawlResponse
        if err != nil {
            resp = s.errorResponse(req.Url, err)
        } else {
            resp = s.toProtoResponse(result)
        }

        if err := stream.Send(resp); err != nil {
            return err
        }
    }
}

func (s *CrawlerServer) StartJob(ctx context.Context, req *pb.StartJobRequest) (*pb.StartJobResponse, error) {
    job, err := s.engine.StartJob(ctx, &engine.JobConfig{
        SeedURLs:         req.SeedUrls,
        MaxPages:         int(req.Config.MaxPages),
        MaxDepth:         int(req.Config.MaxDepth),
        AllowedDomains:   req.Config.AllowedDomains,
        RequestsPerSec:   req.Config.RequestsPerSecond,
        RespectRobotsTxt: req.Config.RespectRobotsTxt,
    })
    if err != nil {
        return nil, status.Errorf(codes.Internal, "failed to start job: %v", err)
    }

    s.mu.Lock()
    s.jobs[job.ID] = job
    s.mu.Unlock()

    return &pb.StartJobResponse{
        JobId:  job.ID,
        Status: "running",
    }, nil
}

func (s *CrawlerServer) GetJobStatus(ctx context.Context, req *pb.GetJobStatusRequest) (*pb.GetJobStatusResponse, error) {
    s.mu.RLock()
    job, exists := s.jobs[req.JobId]
    s.mu.RUnlock()

    if !exists {
        return nil, status.Errorf(codes.NotFound, "job not found: %s", req.JobId)
    }

    stats := job.Stats()

    return &pb.GetJobStatusResponse{
        JobId:  job.ID,
        Status: s.toProtoJobStatus(job.Status()),
        Stats: &pb.JobStats{
            UrlsQueued:        stats.URLsQueued,
            UrlsCrawled:       stats.URLsCrawled,
            UrlsSuccessful:    stats.URLsSuccessful,
            UrlsFailed:        stats.URLsFailed,
            BytesReceived:     stats.BytesReceived,
            StartedAt:         timestamppb.New(stats.StartedAt),
            Elapsed:           durationpb.New(stats.Elapsed),
            RequestsPerSecond: stats.RequestsPerSecond,
        },
    }, nil
}

func (s *CrawlerServer) StopJob(ctx context.Context, req *pb.StopJobRequest) (*pb.StopJobResponse, error) {
    s.mu.RLock()
    job, exists := s.jobs[req.JobId]
    s.mu.RUnlock()

    if !exists {
        return nil, status.Errorf(codes.NotFound, "job not found: %s", req.JobId)
    }

    if req.Force {
        job.ForceStop()
    } else {
        job.GracefulStop()
    }

    return &pb.StopJobResponse{
        Success: true,
        Message: "job stopped",
    }, nil
}

func (s *CrawlerServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
    return &pb.HealthResponse{
        Healthy:       true,
        Version:       "2.0.0",
        UptimeSeconds: int64(time.Since(s.engine.StartTime()).Seconds()),
        Components: map[string]string{
            "engine":   "healthy",
            "frontier": "healthy",
            "storage":  "healthy",
        },
    }, nil
}

// Helper methods for type conversion
func (s *CrawlerServer) toInternalRequest(req *pb.CrawlRequest) *crawler.Request {
    return &crawler.Request{
        URL:      req.Url,
        Options:  s.toInternalOptions(req.Options),
        Metadata: req.Metadata,
    }
}

func (s *CrawlerServer) toInternalOptions(opts *pb.CrawlOptions) *crawler.RequestOptions {
    if opts == nil {
        return nil
    }

    var timeout time.Duration
    if opts.Timeout != nil {
        timeout = opts.Timeout.AsDuration()
    }

    return &crawler.RequestOptions{
        RenderJS:   opts.RenderJs,
        Timeout:    timeout,
        Proxy:      opts.Proxy,
        MaxRetries: int(opts.MaxRetries),
        Headers:    opts.Headers,
    }
}

func (s *CrawlerServer) toProtoResponse(result *crawler.CrawlResult) *pb.CrawlResponse {
    resp := &pb.CrawlResponse{
        Url:         result.URL,
        FinalUrl:    result.FinalURL,
        StatusCode:  int32(result.StatusCode),
        Headers:     result.Headers,
        Content:     result.Content,
        ContentType: result.ContentType,
        FetchTime:   durationpb.New(result.FetchTime),
        FromCache:   result.FromCache,
        Metadata:    result.Metadata,
    }

    if result.Error != nil {
        resp.Error = result.Error.Error()
        resp.ErrorType = s.toProtoErrorType(result.Error)
    }

    return resp
}

func (s *CrawlerServer) errorResponse(url string, err error) *pb.CrawlResponse {
    return &pb.CrawlResponse{
        Url:       url,
        Error:     err.Error(),
        ErrorType: s.toProtoErrorType(err),
    }
}

func (s *CrawlerServer) toProtoErrorType(err error) pb.ErrorType {
    // Map internal error types to proto
    switch {
    case errors.Is(err, crawler.ErrTimeout):
        return pb.ErrorType_ERROR_TYPE_TIMEOUT
    case errors.Is(err, crawler.ErrRateLimited):
        return pb.ErrorType_ERROR_TYPE_RATE_LIMITED
    case errors.Is(err, crawler.ErrBlocked):
        return pb.ErrorType_ERROR_TYPE_BLOCKED
    default:
        return pb.ErrorType_ERROR_TYPE_NETWORK
    }
}

func (s *CrawlerServer) toProtoJobStatus(status engine.JobStatus) pb.JobStatus {
    switch status {
    case engine.JobStatusPending:
        return pb.JobStatus_JOB_STATUS_PENDING
    case engine.JobStatusRunning:
        return pb.JobStatus_JOB_STATUS_RUNNING
    case engine.JobStatusCompleted:
        return pb.JobStatus_JOB_STATUS_COMPLETED
    case engine.JobStatusFailed:
        return pb.JobStatus_JOB_STATUS_FAILED
    case engine.JobStatusCancelled:
        return pb.JobStatus_JOB_STATUS_CANCELLED
    default:
        return pb.JobStatus_JOB_STATUS_UNSPECIFIED
    }
}
```

### 2.4 Python Client (Complete)

```python
# bindings/python/crawler_sdk/__init__.py
"""
Crawler SDK - Python bindings for Go-based web crawler

Usage:
    from crawler_sdk import CrawlerClient, CrawlOptions

    with CrawlerClient() as client:
        result = client.crawl("https://example.com")
        print(result.content)
"""

from .client import CrawlerClient, CrawlOptions, CrawlResult
from .async_client import AsyncCrawlerClient
from .job import Job, JobConfig, JobStatus, JobStats
from .exceptions import (
    CrawlerError,
    NetworkError,
    TimeoutError,
    RateLimitedError,
    BlockedError,
)

__version__ = "2.0.0"
__all__ = [
    "CrawlerClient",
    "AsyncCrawlerClient",
    "CrawlOptions",
    "CrawlResult",
    "Job",
    "JobConfig",
    "JobStatus",
    "JobStats",
    "CrawlerError",
    "NetworkError",
    "TimeoutError",
    "RateLimitedError",
    "BlockedError",
]
```

```python
# bindings/python/crawler_sdk/client.py
"""Synchronous client for the crawler SDK."""

from __future__ import annotations

import grpc
from typing import Iterator, Optional, Dict, List, Any
from dataclasses import dataclass, field
from datetime import timedelta

from .generated import crawler_pb2, crawler_pb2_grpc
from .exceptions import CrawlerError, _map_error


@dataclass
class CrawlOptions:
    """Options for crawl requests."""
    render_js: bool = False
    timeout: timedelta = field(default_factory=lambda: timedelta(seconds=30))
    headers: Optional[Dict[str, str]] = None
    proxy: Optional[str] = None
    max_retries: int = 3
    retry_delay: timedelta = field(default_factory=lambda: timedelta(seconds=1))
    follow_redirects: bool = True
    max_redirects: int = 10
    user_agent: Optional[str] = None
    cookies: Optional[List[str]] = None

    def to_proto(self) -> crawler_pb2.CrawlOptions:
        """Convert to protobuf message."""
        from google.protobuf.duration_pb2 import Duration

        timeout_pb = Duration()
        timeout_pb.FromTimedelta(self.timeout)

        retry_delay_pb = Duration()
        retry_delay_pb.FromTimedelta(self.retry_delay)

        return crawler_pb2.CrawlOptions(
            render_js=self.render_js,
            timeout=timeout_pb,
            headers=self.headers or {},
            proxy=self.proxy or "",
            max_retries=self.max_retries,
            retry_delay=retry_delay_pb,
            follow_redirects=self.follow_redirects,
            max_redirects=self.max_redirects,
            user_agent=self.user_agent or "",
            cookies=self.cookies or [],
        )


@dataclass
class CrawlResult:
    """Result of a crawl operation."""
    url: str
    final_url: str
    status_code: int
    headers: Dict[str, str]
    content: bytes
    content_type: str
    fetch_time: timedelta
    from_cache: bool
    metadata: Dict[str, str]
    error: Optional[str] = None

    @property
    def success(self) -> bool:
        """Check if the crawl was successful."""
        return self.error is None and 200 <= self.status_code < 400

    @property
    def text(self) -> str:
        """Get content as text."""
        return self.content.decode('utf-8', errors='replace')

    def json(self) -> Any:
        """Parse content as JSON."""
        import json
        return json.loads(self.content)

    @classmethod
    def from_proto(cls, resp: crawler_pb2.CrawlResponse) -> CrawlResult:
        """Create from protobuf message."""
        return cls(
            url=resp.url,
            final_url=resp.final_url or resp.url,
            status_code=resp.status_code,
            headers=dict(resp.headers),
            content=resp.content,
            content_type=resp.content_type,
            fetch_time=resp.fetch_time.ToTimedelta() if resp.fetch_time else timedelta(),
            from_cache=resp.from_cache,
            metadata=dict(resp.metadata),
            error=resp.error if resp.error else None,
        )


class CrawlerClient:
    """Synchronous Python client for Go crawler SDK.

    Example:
        >>> with CrawlerClient() as client:
        ...     result = client.crawl("https://example.com")
        ...     print(result.status_code)
        200

        >>> # With options
        >>> options = CrawlOptions(render_js=True, timeout=timedelta(seconds=60))
        >>> result = client.crawl("https://spa-site.com", options=options)
    """

    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        secure: bool = False,
        credentials: Optional[grpc.ChannelCredentials] = None,
    ):
        """Initialize the client.

        Args:
            host: gRPC server host
            port: gRPC server port
            secure: Use TLS connection
            credentials: gRPC credentials for secure connection
        """
        self.address = f"{host}:{port}"

        if secure:
            if credentials is None:
                credentials = grpc.ssl_channel_credentials()
            self.channel = grpc.secure_channel(self.address, credentials)
        else:
            self.channel = grpc.insecure_channel(self.address)

        self.stub = crawler_pb2_grpc.CrawlerServiceStub(self.channel)

    def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
        metadata: Optional[Dict[str, str]] = None,
    ) -> CrawlResult:
        """Crawl a single URL.

        Args:
            url: URL to crawl
            options: Crawl options
            metadata: Additional metadata

        Returns:
            CrawlResult containing the response

        Raises:
            CrawlerError: If the crawl fails
        """
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlRequest(
            url=url,
            options=options.to_proto(),
            metadata=metadata or {},
        )

        try:
            response = self.stub.Crawl(request)
            result = CrawlResult.from_proto(response)

            if result.error:
                raise _map_error(response.error_type, result.error, url)

            return result

        except grpc.RpcError as e:
            raise CrawlerError(f"gRPC error: {e.code()}: {e.details()}") from e

    def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> Iterator[CrawlResult]:
        """Crawl multiple URLs with streaming results.

        Args:
            urls: List of URLs to crawl
            options: Crawl options (applied to all URLs)
            concurrency: Number of concurrent requests

        Yields:
            CrawlResult for each URL

        Example:
            >>> urls = ["https://example.com/page1", "https://example.com/page2"]
            >>> for result in client.crawl_batch(urls, concurrency=5):
            ...     print(f"{result.url}: {result.status_code}")
        """
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlBatchRequest(
            urls=urls,
            options=options.to_proto(),
            concurrency=concurrency,
        )

        try:
            for response in self.stub.CrawlBatch(request):
                yield CrawlResult.from_proto(response)

        except grpc.RpcError as e:
            raise CrawlerError(f"gRPC error: {e.code()}: {e.details()}") from e

    def health(self) -> Dict[str, Any]:
        """Check server health.

        Returns:
            Health status dictionary
        """
        response = self.stub.Health(crawler_pb2.HealthRequest())
        return {
            "healthy": response.healthy,
            "version": response.version,
            "uptime_seconds": response.uptime_seconds,
            "components": dict(response.components),
        }

    def close(self):
        """Close the gRPC channel."""
        self.channel.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
```

```python
# bindings/python/crawler_sdk/async_client.py
"""Asynchronous client for the crawler SDK."""

from __future__ import annotations

import grpc.aio
from typing import AsyncIterator, Optional, Dict, List, Any
from datetime import timedelta

from .generated import crawler_pb2, crawler_pb2_grpc
from .client import CrawlOptions, CrawlResult
from .exceptions import CrawlerError, _map_error


class AsyncCrawlerClient:
    """Asynchronous Python client for Go crawler SDK.

    Example:
        >>> async with AsyncCrawlerClient() as client:
        ...     result = await client.crawl("https://example.com")
        ...     print(result.status_code)

        >>> # Concurrent crawling
        >>> import asyncio
        >>> urls = ["https://example.com/1", "https://example.com/2"]
        >>> tasks = [client.crawl(url) for url in urls]
        >>> results = await asyncio.gather(*tasks)
    """

    def __init__(
        self,
        host: str = "localhost",
        port: int = 50051,
        secure: bool = False,
    ):
        """Initialize the async client."""
        self.address = f"{host}:{port}"

        if secure:
            self.channel = grpc.aio.secure_channel(
                self.address,
                grpc.ssl_channel_credentials()
            )
        else:
            self.channel = grpc.aio.insecure_channel(self.address)

        self.stub = crawler_pb2_grpc.CrawlerServiceStub(self.channel)

    async def crawl(
        self,
        url: str,
        options: Optional[CrawlOptions] = None,
        metadata: Optional[Dict[str, str]] = None,
    ) -> CrawlResult:
        """Crawl a single URL asynchronously."""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlRequest(
            url=url,
            options=options.to_proto(),
            metadata=metadata or {},
        )

        try:
            response = await self.stub.Crawl(request)
            result = CrawlResult.from_proto(response)

            if result.error:
                raise _map_error(response.error_type, result.error, url)

            return result

        except grpc.RpcError as e:
            raise CrawlerError(f"gRPC error: {e.code()}: {e.details()}") from e

    async def crawl_batch(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        concurrency: int = 10,
    ) -> AsyncIterator[CrawlResult]:
        """Crawl multiple URLs with async streaming results."""
        options = options or CrawlOptions()

        request = crawler_pb2.CrawlBatchRequest(
            urls=urls,
            options=options.to_proto(),
            concurrency=concurrency,
        )

        try:
            async for response in self.stub.CrawlBatch(request):
                yield CrawlResult.from_proto(response)

        except grpc.RpcError as e:
            raise CrawlerError(f"gRPC error: {e.code()}: {e.details()}") from e

    async def crawl_many(
        self,
        urls: List[str],
        options: Optional[CrawlOptions] = None,
        max_concurrency: int = 10,
    ) -> List[CrawlResult]:
        """Crawl multiple URLs concurrently and return all results.

        This method uses asyncio.gather for maximum concurrency control.
        """
        import asyncio

        semaphore = asyncio.Semaphore(max_concurrency)

        async def crawl_with_semaphore(url: str) -> CrawlResult:
            async with semaphore:
                return await self.crawl(url, options)

        tasks = [crawl_with_semaphore(url) for url in urls]
        return await asyncio.gather(*tasks, return_exceptions=True)

    async def health(self) -> Dict[str, Any]:
        """Check server health asynchronously."""
        response = await self.stub.Health(crawler_pb2.HealthRequest())
        return {
            "healthy": response.healthy,
            "version": response.version,
            "uptime_seconds": response.uptime_seconds,
            "components": dict(response.components),
        }

    async def close(self):
        """Close the gRPC channel."""
        await self.channel.close()

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()
```

```python
# bindings/python/crawler_sdk/exceptions.py
"""Exception types for the crawler SDK."""

from .generated import crawler_pb2


class CrawlerError(Exception):
    """Base exception for crawler errors."""
    pass


class NetworkError(CrawlerError):
    """Network-related error."""
    pass


class TimeoutError(CrawlerError):
    """Request timed out."""
    pass


class RateLimitedError(CrawlerError):
    """Rate limit exceeded."""
    pass


class BlockedError(CrawlerError):
    """Request was blocked."""
    pass


class RobotsTxtError(CrawlerError):
    """Disallowed by robots.txt."""
    pass


class ParseError(CrawlerError):
    """Failed to parse content."""
    pass


def _map_error(error_type: int, message: str, url: str) -> CrawlerError:
    """Map protobuf error type to Python exception."""
    error_map = {
        crawler_pb2.ErrorType.ERROR_TYPE_NETWORK: NetworkError,
        crawler_pb2.ErrorType.ERROR_TYPE_TIMEOUT: TimeoutError,
        crawler_pb2.ErrorType.ERROR_TYPE_RATE_LIMITED: RateLimitedError,
        crawler_pb2.ErrorType.ERROR_TYPE_BLOCKED: BlockedError,
        crawler_pb2.ErrorType.ERROR_TYPE_ROBOTS_TXT: RobotsTxtError,
        crawler_pb2.ErrorType.ERROR_TYPE_PARSE: ParseError,
    }

    error_class = error_map.get(error_type, CrawlerError)
    return error_class(f"{message} (url: {url})")
```

---

## 3. CGO Binding (Secondary)

### 3.1 When to Use CGO

```
✅ Use CGO when:
- Deploying as a single binary (serverless, embedded)
- Ultra-low latency requirements (< 1ms overhead)
- No network dependency between components
- Python extension module distribution

❌ Avoid CGO when:
- Need separate scaling of Go and Python
- Cross-platform distribution complexity
- Team unfamiliar with C/CGO
```

### 3.2 CGO Export Example

```go
// bindings/cgo/crawler.go
package main

/*
#include <stdlib.h>
#include <string.h>

typedef struct {
    char* url;
    int status_code;
    char* content;
    int content_length;
    char* error;
} CrawlResult;

typedef struct {
    int render_js;
    int timeout_seconds;
    char* proxy;
} CrawlOptions;
*/
import "C"
import (
    "context"
    "time"
    "unsafe"

    "github.com/yourorg/crawler-sdk/internal/engine"
    "github.com/yourorg/crawler-sdk/pkg/crawler"
)

var globalEngine *engine.Engine

//export CrawlerInit
func CrawlerInit() C.int {
    var err error
    globalEngine, err = engine.New(engine.DefaultConfig())
    if err != nil {
        return -1
    }
    return 0
}

//export CrawlerCrawl
func CrawlerCrawl(url *C.char, opts *C.CrawlOptions) *C.CrawlResult {
    goURL := C.GoString(url)

    var options *crawler.RequestOptions
    if opts != nil {
        options = &crawler.RequestOptions{
            RenderJS: opts.render_js != 0,
            Timeout:  time.Duration(opts.timeout_seconds) * time.Second,
        }
        if opts.proxy != nil {
            options.Proxy = C.GoString(opts.proxy)
        }
    }

    req := &crawler.Request{
        URL:     goURL,
        Options: options,
    }

    result, err := globalEngine.Crawl(context.Background(), req)

    cResult := (*C.CrawlResult)(C.malloc(C.sizeof_CrawlResult))

    if err != nil {
        cResult.error = C.CString(err.Error())
        return cResult
    }

    cResult.url = C.CString(result.URL)
    cResult.status_code = C.int(result.StatusCode)
    cResult.content = (*C.char)(C.CBytes(result.Content))
    cResult.content_length = C.int(len(result.Content))
    cResult.error = nil

    return cResult
}

//export CrawlerFreeResult
func CrawlerFreeResult(result *C.CrawlResult) {
    if result == nil {
        return
    }
    if result.url != nil {
        C.free(unsafe.Pointer(result.url))
    }
    if result.content != nil {
        C.free(unsafe.Pointer(result.content))
    }
    if result.error != nil {
        C.free(unsafe.Pointer(result.error))
    }
    C.free(unsafe.Pointer(result))
}

//export CrawlerClose
func CrawlerClose() {
    if globalEngine != nil {
        globalEngine.Close()
    }
}

func main() {}
```

### 3.3 Python ctypes Wrapper

```python
# bindings/python/crawler_sdk/cgo_client.py
"""CGO-based client using ctypes."""

import ctypes
from ctypes import c_char_p, c_int, POINTER, Structure
from pathlib import Path
from typing import Optional


class CrawlOptions(Structure):
    _fields_ = [
        ("render_js", c_int),
        ("timeout_seconds", c_int),
        ("proxy", c_char_p),
    ]


class CrawlResult(Structure):
    _fields_ = [
        ("url", c_char_p),
        ("status_code", c_int),
        ("content", c_char_p),
        ("content_length", c_int),
        ("error", c_char_p),
    ]


class CGOCrawlerClient:
    """Python client using CGO-compiled shared library."""

    def __init__(self, lib_path: Optional[str] = None):
        """Initialize CGO client.

        Args:
            lib_path: Path to libcrawler.so/dylib/dll
        """
        if lib_path is None:
            lib_path = self._find_library()

        self.lib = ctypes.CDLL(lib_path)
        self._setup_functions()

        if self.lib.CrawlerInit() != 0:
            raise RuntimeError("Failed to initialize crawler")

    def _find_library(self) -> str:
        """Find the shared library."""
        import platform

        system = platform.system()
        if system == "Linux":
            name = "libcrawler.so"
        elif system == "Darwin":
            name = "libcrawler.dylib"
        elif system == "Windows":
            name = "crawler.dll"
        else:
            raise RuntimeError(f"Unsupported platform: {system}")

        # Search paths
        search_paths = [
            Path(__file__).parent / "lib" / name,
            Path.cwd() / name,
            Path("/usr/local/lib") / name,
        ]

        for path in search_paths:
            if path.exists():
                return str(path)

        raise FileNotFoundError(f"Could not find {name}")

    def _setup_functions(self):
        """Set up ctypes function signatures."""
        self.lib.CrawlerInit.restype = c_int
        self.lib.CrawlerInit.argtypes = []

        self.lib.CrawlerCrawl.restype = POINTER(CrawlResult)
        self.lib.CrawlerCrawl.argtypes = [c_char_p, POINTER(CrawlOptions)]

        self.lib.CrawlerFreeResult.restype = None
        self.lib.CrawlerFreeResult.argtypes = [POINTER(CrawlResult)]

        self.lib.CrawlerClose.restype = None
        self.lib.CrawlerClose.argtypes = []

    def crawl(
        self,
        url: str,
        render_js: bool = False,
        timeout: int = 30,
        proxy: Optional[str] = None,
    ) -> dict:
        """Crawl a URL using CGO backend."""
        opts = CrawlOptions()
        opts.render_js = 1 if render_js else 0
        opts.timeout_seconds = timeout
        opts.proxy = proxy.encode() if proxy else None

        result_ptr = self.lib.CrawlerCrawl(url.encode(), ctypes.byref(opts))

        try:
            result = result_ptr.contents

            if result.error:
                raise RuntimeError(result.error.decode())

            return {
                "url": result.url.decode() if result.url else url,
                "status_code": result.status_code,
                "content": result.content[:result.content_length] if result.content else b"",
            }
        finally:
            self.lib.CrawlerFreeResult(result_ptr)

    def close(self):
        """Close the crawler."""
        self.lib.CrawlerClose()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
```

---

## 4. Build & Distribution

### 4.1 Makefile

```makefile
# Makefile
.PHONY: all proto build build-cgo test clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

all: proto build

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/crawler/v1/crawler.proto
	python -m grpc_tools.protoc -I. \
		--python_out=bindings/python/crawler_sdk/generated \
		--grpc_python_out=bindings/python/crawler_sdk/generated \
		api/proto/crawler/v1/crawler.proto

# Build Go binaries
build:
	@echo "Building server..."
	go build $(LDFLAGS) -o bin/crawler-server ./cmd/server
	@echo "Building CLI..."
	go build $(LDFLAGS) -o bin/crawler ./cmd/crawler

# Build CGO shared library
build-cgo:
	@echo "Building CGO library..."
	CGO_ENABLED=1 go build -buildmode=c-shared \
		-o bindings/python/crawler_sdk/lib/libcrawler.so \
		./bindings/cgo

# Build Python wheel
build-python: proto
	@echo "Building Python package..."
	cd bindings/python && python -m build

# Run tests
test:
	go test -v ./...
	cd bindings/python && pytest

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf bindings/python/dist/
	rm -rf bindings/python/crawler_sdk/generated/
```

### 4.2 Python Package Setup

```python
# bindings/python/setup.py
from setuptools import setup, find_packages

setup(
    name="crawler-sdk",
    version="2.0.0",
    description="Python SDK for Go-based web crawler",
    long_description=open("README.md").read(),
    long_description_content_type="text/markdown",
    author="Your Organization",
    author_email="sdk@yourorg.com",
    url="https://github.com/yourorg/crawler-sdk",
    packages=find_packages(),
    package_data={
        "crawler_sdk": ["lib/*.so", "lib/*.dylib", "lib/*.dll"],
    },
    install_requires=[
        "grpcio>=1.62.0",
        "grpcio-tools>=1.62.0",
        "protobuf>=4.25.0",
    ],
    extras_require={
        "async": ["grpcio-aio>=1.62.0"],
        "parsing": ["beautifulsoup4>=4.12.0", "lxml>=5.0.0", "parsel>=1.9.0"],
        "dev": ["pytest>=8.0.0", "pytest-asyncio>=0.23.0", "black", "mypy", "ruff"],
    },
    python_requires=">=3.10",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
)
```

---

## 5. Usage Examples

### 5.1 Basic Python Usage

```python
# examples/python/basic.py
from crawler_sdk import CrawlerClient, CrawlOptions
from datetime import timedelta

# Simple crawl
with CrawlerClient() as client:
    result = client.crawl("https://example.com")
    print(f"Status: {result.status_code}")
    print(f"Content length: {len(result.content)}")

# With options
options = CrawlOptions(
    render_js=True,
    timeout=timedelta(seconds=60),
    headers={"Accept-Language": "en-US"},
)

with CrawlerClient() as client:
    result = client.crawl("https://spa-site.com", options=options)
    print(result.text[:500])
```

### 5.2 Async Python Usage

```python
# examples/python/async_example.py
import asyncio
from crawler_sdk import AsyncCrawlerClient, CrawlOptions

async def main():
    async with AsyncCrawlerClient() as client:
        # Single crawl
        result = await client.crawl("https://example.com")
        print(f"Status: {result.status_code}")

        # Concurrent crawling
        urls = [f"https://example.com/page/{i}" for i in range(10)]
        results = await client.crawl_many(urls, max_concurrency=5)

        for result in results:
            if isinstance(result, Exception):
                print(f"Error: {result}")
            else:
                print(f"{result.url}: {result.status_code}")

if __name__ == "__main__":
    asyncio.run(main())
```

### 5.3 Data Pipeline Integration

```python
# examples/python/pipeline.py
from crawler_sdk import CrawlerClient, CrawlOptions
from bs4 import BeautifulSoup
import pandas as pd

def extract_products(html: str) -> list[dict]:
    """Extract products from HTML."""
    soup = BeautifulSoup(html, 'lxml')
    products = []

    for item in soup.select('.product-item'):
        products.append({
            'name': item.select_one('.product-name').text.strip(),
            'price': item.select_one('.product-price').text.strip(),
            'url': item.select_one('a')['href'],
        })

    return products

def main():
    urls = [f"https://shop.example.com/page/{i}" for i in range(1, 11)]

    all_products = []

    with CrawlerClient() as client:
        for result in client.crawl_batch(urls, concurrency=3):
            if result.success:
                products = extract_products(result.text)
                all_products.extend(products)
                print(f"Extracted {len(products)} products from {result.url}")
            else:
                print(f"Failed: {result.url} - {result.error}")

    # Save to DataFrame
    df = pd.DataFrame(all_products)
    df.to_csv("products.csv", index=False)
    print(f"Saved {len(df)} products")

if __name__ == "__main__":
    main()
```

---

## References

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [gRPC Python Documentation](https://grpc.io/docs/languages/python/)
- [Protocol Buffers](https://protobuf.dev/)
- [CGO Documentation](https://pkg.go.dev/cmd/cgo)
- [ctypes Documentation](https://docs.python.org/3/library/ctypes.html)

---

*The binding layer bridges Go's performance with Python's accessibility.*
