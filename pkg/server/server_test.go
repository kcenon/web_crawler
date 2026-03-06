package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/kcenon/web_crawler/pkg/crawler"
	pb "github.com/kcenon/web_crawler/pkg/server/pb"
)

const bufSize = 1024 * 1024

// mockCrawler implements crawler.Crawler with configurable function fields.
type mockCrawler struct {
	crawlFn  func(ctx context.Context, urls []string, cfg *crawler.CrawlConfig) ([]*crawler.Result, error)
	startFn  func(ctx context.Context, id string, cfg *crawler.CrawlConfig) error
	stopFn   func(ctx context.Context, id string) (*crawler.Stats, error)
	statsFn  func(ctx context.Context, id string) (*crawler.Stats, error)
	addURLFn func(ctx context.Context, id string, urls []string) (int, error)
}

func (m *mockCrawler) Crawl(ctx context.Context, urls []string, cfg *crawler.CrawlConfig) ([]*crawler.Result, error) {
	if m.crawlFn != nil {
		return m.crawlFn(ctx, urls, cfg)
	}
	return nil, nil
}

func (m *mockCrawler) Start(ctx context.Context, id string, cfg *crawler.CrawlConfig) error {
	if m.startFn != nil {
		return m.startFn(ctx, id, cfg)
	}
	return nil
}

func (m *mockCrawler) Stop(ctx context.Context, id string) (*crawler.Stats, error) {
	if m.stopFn != nil {
		return m.stopFn(ctx, id)
	}
	return &crawler.Stats{}, nil
}

func (m *mockCrawler) Stats(ctx context.Context, id string) (*crawler.Stats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx, id)
	}
	return &crawler.Stats{}, nil
}

func (m *mockCrawler) AddURLs(ctx context.Context, id string, urls []string) (int, error) {
	if m.addURLFn != nil {
		return m.addURLFn(ctx, id, urls)
	}
	return 0, nil
}

// setupTest creates an in-process gRPC server and client using bufconn.
func setupTest(t *testing.T, mock *mockCrawler) pb.CrawlerServiceClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)

	srv := New(mock, Config{}, WithLogger(slog.Default()))
	srv.grpc = grpc.NewServer(
		grpc.ChainUnaryInterceptor(srv.unaryLoggingInterceptor),
		grpc.ChainStreamInterceptor(srv.streamLoggingInterceptor),
	)
	pb.RegisterCrawlerServiceServer(srv.grpc, srv)

	go func() {
		_ = srv.grpc.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}

	t.Cleanup(func() {
		conn.Close()
		srv.grpc.GracefulStop()
	})

	return pb.NewCrawlerServiceClient(conn)
}

// --- Crawl RPC Tests ---

func TestCrawl_Success(t *testing.T) {
	mock := &mockCrawler{
		crawlFn: func(_ context.Context, urls []string, _ *crawler.CrawlConfig) ([]*crawler.Result, error) {
			return []*crawler.Result{
				{URL: urls[0], Content: "<html>test</html>", StatusCode: 200},
			}, nil
		},
	}
	client := setupTest(t, mock)

	resp, err := client.Crawl(context.Background(), &pb.CrawlRequest{
		Urls: []string{"https://example.com"},
	})
	if err != nil {
		t.Fatalf("Crawl: %v", err)
	}
	if resp.GetUrl() != "https://example.com" {
		t.Errorf("URL = %q, want %q", resp.GetUrl(), "https://example.com")
	}
	if resp.GetStatusCode() != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.GetStatusCode())
	}
	if resp.GetContent() != "<html>test</html>" {
		t.Errorf("Content = %q, want %q", resp.GetContent(), "<html>test</html>")
	}
}

func TestCrawl_EmptyURLs(t *testing.T) {
	client := setupTest(t, &mockCrawler{})

	_, err := client.Crawl(context.Background(), &pb.CrawlRequest{})
	if err == nil {
		t.Fatal("expected error for empty URLs")
	}
	if code := status.Code(err); code != codes.InvalidArgument {
		t.Errorf("code = %v, want InvalidArgument", code)
	}
}

func TestCrawl_CrawlerError(t *testing.T) {
	mock := &mockCrawler{
		crawlFn: func(context.Context, []string, *crawler.CrawlConfig) ([]*crawler.Result, error) {
			return nil, errors.New("connection refused")
		},
	}
	client := setupTest(t, mock)

	_, err := client.Crawl(context.Background(), &pb.CrawlRequest{
		Urls: []string{"https://example.com"},
	})
	if err == nil {
		t.Fatal("expected error from crawler")
	}
	if code := status.Code(err); code != codes.Internal {
		t.Errorf("code = %v, want Internal", code)
	}
}

// --- StartCrawler / StopCrawler Tests ---

func TestStartCrawler_Success(t *testing.T) {
	mock := &mockCrawler{
		startFn: func(context.Context, string, *crawler.CrawlConfig) error {
			return nil
		},
	}
	client := setupTest(t, mock)

	resp, err := client.StartCrawler(context.Background(), &pb.StartCrawlerRequest{
		Config: &pb.CrawlConfig{
			Urls: []string{"https://example.com"},
		},
	})
	if err != nil {
		t.Fatalf("StartCrawler: %v", err)
	}
	if resp.GetCrawlerId() == "" {
		t.Error("expected non-empty crawler ID")
	}
	if resp.GetStatus() != pb.CrawlerStatus_CRAWLER_STATUS_RUNNING {
		t.Errorf("status = %v, want RUNNING", resp.GetStatus())
	}
}

func TestStopCrawler_Success(t *testing.T) {
	mock := &mockCrawler{
		startFn: func(context.Context, string, *crawler.CrawlConfig) error {
			return nil
		},
		stopFn: func(context.Context, string) (*crawler.Stats, error) {
			return &crawler.Stats{PagesCrawled: 42, PagesFailed: 3, PagesQueued: 0}, nil
		},
	}
	client := setupTest(t, mock)

	// Start first
	startResp, err := client.StartCrawler(context.Background(), &pb.StartCrawlerRequest{
		CrawlerId: "test-crawler",
	})
	if err != nil {
		t.Fatalf("StartCrawler: %v", err)
	}

	// Then stop
	stopResp, err := client.StopCrawler(context.Background(), &pb.StopCrawlerRequest{
		CrawlerId: startResp.GetCrawlerId(),
	})
	if err != nil {
		t.Fatalf("StopCrawler: %v", err)
	}
	if stopResp.GetStatus() != pb.CrawlerStatus_CRAWLER_STATUS_STOPPED {
		t.Errorf("status = %v, want STOPPED", stopResp.GetStatus())
	}
	if stopResp.GetFinalStats().GetPagesCrawled() != 42 {
		t.Errorf("PagesCrawled = %d, want 42", stopResp.GetFinalStats().GetPagesCrawled())
	}
}

func TestStopCrawler_NotFound(t *testing.T) {
	client := setupTest(t, &mockCrawler{})

	_, err := client.StopCrawler(context.Background(), &pb.StopCrawlerRequest{
		CrawlerId: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent crawler")
	}
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("code = %v, want NotFound", code)
	}
}

// --- GetStats Tests ---

func TestGetStats_Success(t *testing.T) {
	mock := &mockCrawler{
		startFn: func(context.Context, string, *crawler.CrawlConfig) error {
			return nil
		},
		statsFn: func(context.Context, string) (*crawler.Stats, error) {
			return &crawler.Stats{PagesCrawled: 10, PagesFailed: 1, PagesQueued: 5}, nil
		},
	}
	client := setupTest(t, mock)

	// Start a crawler first
	_, err := client.StartCrawler(context.Background(), &pb.StartCrawlerRequest{
		CrawlerId: "stats-test",
	})
	if err != nil {
		t.Fatalf("StartCrawler: %v", err)
	}

	resp, err := client.GetStats(context.Background(), &pb.GetStatsRequest{
		CrawlerId: "stats-test",
	})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if resp.GetStats().GetPagesCrawled() != 10 {
		t.Errorf("PagesCrawled = %d, want 10", resp.GetStats().GetPagesCrawled())
	}
	if resp.GetStats().GetPagesFailed() != 1 {
		t.Errorf("PagesFailed = %d, want 1", resp.GetStats().GetPagesFailed())
	}
	if resp.GetStats().GetPagesQueued() != 5 {
		t.Errorf("PagesQueued = %d, want 5", resp.GetStats().GetPagesQueued())
	}
}

func TestGetStats_NotFound(t *testing.T) {
	client := setupTest(t, &mockCrawler{})

	_, err := client.GetStats(context.Background(), &pb.GetStatsRequest{
		CrawlerId: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent crawler")
	}
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("code = %v, want NotFound", code)
	}
}

// --- AddURLs Tests ---

func TestAddURLs_Success(t *testing.T) {
	mock := &mockCrawler{
		startFn: func(context.Context, string, *crawler.CrawlConfig) error {
			return nil
		},
		addURLFn: func(_ context.Context, _ string, urls []string) (int, error) {
			return len(urls), nil
		},
	}
	client := setupTest(t, mock)

	// Start a crawler first
	_, err := client.StartCrawler(context.Background(), &pb.StartCrawlerRequest{
		CrawlerId: "add-urls-test",
	})
	if err != nil {
		t.Fatalf("StartCrawler: %v", err)
	}

	resp, err := client.AddURLs(context.Background(), &pb.AddURLsRequest{
		CrawlerId: "add-urls-test",
		Urls:      []string{"https://a.com", "https://b.com"},
	})
	if err != nil {
		t.Fatalf("AddURLs: %v", err)
	}
	if resp.GetAddedCount() != 2 {
		t.Errorf("AddedCount = %d, want 2", resp.GetAddedCount())
	}
}

func TestAddURLs_NotFound(t *testing.T) {
	client := setupTest(t, &mockCrawler{})

	_, err := client.AddURLs(context.Background(), &pb.AddURLsRequest{
		CrawlerId: "nonexistent",
		Urls:      []string{"https://a.com"},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent crawler")
	}
	if code := status.Code(err); code != codes.NotFound {
		t.Errorf("code = %v, want NotFound", code)
	}
}

// --- CrawlStream Tests ---

func TestCrawlStream_Success(t *testing.T) {
	mock := &mockCrawler{
		crawlFn: func(_ context.Context, urls []string, _ *crawler.CrawlConfig) ([]*crawler.Result, error) {
			results := make([]*crawler.Result, len(urls))
			for i, u := range urls {
				results[i] = &crawler.Result{URL: u, Content: "ok", StatusCode: 200}
			}
			return results, nil
		},
	}
	client := setupTest(t, mock)

	stream, err := client.CrawlStream(context.Background())
	if err != nil {
		t.Fatalf("CrawlStream: %v", err)
	}

	// Send two requests
	for _, url := range []string{"https://a.com", "https://b.com"} {
		if err := stream.Send(&pb.CrawlRequest{Urls: []string{url}}); err != nil {
			t.Fatalf("Send: %v", err)
		}
	}
	if err := stream.CloseSend(); err != nil {
		t.Fatalf("CloseSend: %v", err)
	}

	// Collect responses
	var responses []*pb.CrawlResponse
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		responses = append(responses, resp)
	}

	if len(responses) != 2 {
		t.Fatalf("got %d responses, want 2", len(responses))
	}
	if responses[0].GetUrl() != "https://a.com" {
		t.Errorf("response[0].URL = %q, want %q", responses[0].GetUrl(), "https://a.com")
	}
	if responses[1].GetUrl() != "https://b.com" {
		t.Errorf("response[1].URL = %q, want %q", responses[1].GetUrl(), "https://b.com")
	}
}

// --- Graceful Shutdown Test ---

func TestGracefulShutdown(t *testing.T) {
	srv := New(&mockCrawler{}, Config{Port: 0})
	srv.logger = slog.Default()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(50 * time.Millisecond)

	if srv.Addr() == nil {
		t.Fatal("expected non-nil Addr after Start")
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("Start returned unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop within 5s")
	}
}

// --- Health Check Test ---

func TestHealthCheck(t *testing.T) {
	lis := bufconn.Listen(bufSize)

	srv := New(&mockCrawler{}, Config{})
	srv.grpc = grpc.NewServer()
	pb.RegisterCrawlerServiceServer(srv.grpc, srv)

	srv.health = health.NewServer()
	healthpb.RegisterHealthServer(srv.grpc, srv.health)
	srv.health.SetServingStatus("crawler.v1.CrawlerService", healthpb.HealthCheckResponse_SERVING)

	go func() {
		_ = srv.grpc.Serve(lis)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc.NewClient: %v", err)
	}
	t.Cleanup(func() {
		conn.Close()
		srv.grpc.GracefulStop()
	})

	healthClient := healthpb.NewHealthClient(conn)
	resp, err := healthClient.Check(context.Background(), &healthpb.HealthCheckRequest{
		Service: "crawler.v1.CrawlerService",
	})
	if err != nil {
		t.Fatalf("Health.Check: %v", err)
	}
	if resp.GetStatus() != healthpb.HealthCheckResponse_SERVING {
		t.Errorf("health status = %v, want SERVING", resp.GetStatus())
	}
}
