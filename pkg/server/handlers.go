package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kcenon/web_crawler/pkg/crawler"
	pb "github.com/kcenon/web_crawler/pkg/server/pb"
)

// Crawl handles a unary crawl request: validate URLs, delegate to the crawler,
// and return the first result as a CrawlResponse.
func (s *Server) Crawl(ctx context.Context, req *pb.CrawlRequest) (*pb.CrawlResponse, error) {
	if len(req.GetUrls()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one URL is required")
	}

	cfg := protoConfigToCrawlConfig(req.GetConfig())
	start := time.Now()

	results, err := s.crawler.Crawl(ctx, req.GetUrls(), cfg)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "crawl failed: %v", err)
	}

	duration := time.Since(start)

	if len(results) == 0 {
		return &pb.CrawlResponse{}, nil
	}

	r := results[0]
	resp := &pb.CrawlResponse{
		Url:        r.URL,
		StatusCode: clampInt32(r.StatusCode),
		Content:    r.Content,
		CrawledAt:  timestamppb.Now(),
		Duration:   durationpb.New(duration),
	}
	if r.Error != nil {
		resp.Error = &pb.ErrorInfo{
			Code:    pb.ErrorCode_ERROR_CODE_INTERNAL,
			Message: r.Error.Error(),
			Url:     r.URL,
		}
	}
	return resp, nil
}

// CrawlStream handles bidirectional streaming: read requests, crawl, send responses.
func (s *Server) CrawlStream(stream grpc.BidiStreamingServer[pb.CrawlRequest, pb.CrawlResponse]) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		if len(req.GetUrls()) == 0 {
			continue
		}

		cfg := protoConfigToCrawlConfig(req.GetConfig())
		start := time.Now()

		results, crawlErr := s.crawler.Crawl(stream.Context(), req.GetUrls(), cfg)
		if crawlErr != nil {
			if sendErr := stream.Send(&pb.CrawlResponse{
				Error: &pb.ErrorInfo{
					Code:    pb.ErrorCode_ERROR_CODE_INTERNAL,
					Message: crawlErr.Error(),
				},
			}); sendErr != nil {
				return sendErr
			}
			continue
		}

		duration := time.Since(start)
		for _, r := range results {
			resp := &pb.CrawlResponse{
				Url:        r.URL,
				StatusCode: clampInt32(r.StatusCode),
				Content:    r.Content,
				CrawledAt:  timestamppb.Now(),
				Duration:   durationpb.New(duration),
			}
			if r.Error != nil {
				resp.Error = &pb.ErrorInfo{
					Code:    pb.ErrorCode_ERROR_CODE_INTERNAL,
					Message: r.Error.Error(),
					Url:     r.URL,
				}
			}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}

// StartCrawler creates a new long-running crawler instance.
func (s *Server) StartCrawler(ctx context.Context, req *pb.StartCrawlerRequest) (*pb.StartCrawlerResponse, error) {
	id := req.GetCrawlerId()
	if id == "" {
		var err error
		id, err = generateID()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "generate crawler id: %v", err)
		}
	}

	cfg := protoConfigToCrawlConfig(req.GetConfig())
	if err := s.crawler.Start(ctx, id, cfg); err != nil {
		return nil, status.Errorf(codes.Internal, "start crawler: %v", err)
	}

	s.mu.Lock()
	s.instances[id] = &crawlerInstance{
		status: pb.CrawlerStatus_CRAWLER_STATUS_RUNNING,
		config: cfg,
	}
	s.mu.Unlock()

	return &pb.StartCrawlerResponse{
		CrawlerId: id,
		Status:    pb.CrawlerStatus_CRAWLER_STATUS_RUNNING,
	}, nil
}

// StopCrawler stops a running crawler and returns final statistics.
func (s *Server) StopCrawler(ctx context.Context, req *pb.StopCrawlerRequest) (*pb.StopCrawlerResponse, error) {
	id := req.GetCrawlerId()

	s.mu.RLock()
	_, ok := s.instances[id]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "crawler %q not found", id)
	}

	stats, err := s.crawler.Stop(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "stop crawler: %v", err)
	}

	s.mu.Lock()
	s.instances[id].status = pb.CrawlerStatus_CRAWLER_STATUS_STOPPED
	s.mu.Unlock()

	return &pb.StopCrawlerResponse{
		CrawlerId: id,
		Status:    pb.CrawlerStatus_CRAWLER_STATUS_STOPPED,
		FinalStats: &pb.CrawlStats{
			PagesCrawled: stats.PagesCrawled,
			PagesFailed:  stats.PagesFailed,
			PagesQueued:  stats.PagesQueued,
			Status:       pb.CrawlerStatus_CRAWLER_STATUS_STOPPED,
		},
	}, nil
}

// GetStats returns current statistics for a running crawler.
func (s *Server) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	id := req.GetCrawlerId()

	s.mu.RLock()
	inst, ok := s.instances[id]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "crawler %q not found", id)
	}

	stats, err := s.crawler.Stats(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get stats: %v", err)
	}

	return &pb.GetStatsResponse{
		Stats: &pb.CrawlStats{
			PagesCrawled: stats.PagesCrawled,
			PagesFailed:  stats.PagesFailed,
			PagesQueued:  stats.PagesQueued,
			Status:       inst.status,
		},
	}, nil
}

// AddURLs injects additional URLs into a running crawler's frontier.
func (s *Server) AddURLs(ctx context.Context, req *pb.AddURLsRequest) (*pb.AddURLsResponse, error) {
	id := req.GetCrawlerId()

	s.mu.RLock()
	_, ok := s.instances[id]
	s.mu.RUnlock()
	if !ok {
		return nil, status.Errorf(codes.NotFound, "crawler %q not found", id)
	}

	added, err := s.crawler.AddURLs(ctx, id, req.GetUrls())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "add urls: %v", err)
	}

	return &pb.AddURLsResponse{
		AddedCount: clampInt32(added),
	}, nil
}

// protoConfigToCrawlConfig converts a protobuf CrawlConfig to the domain type.
func protoConfigToCrawlConfig(pc *pb.CrawlConfig) *crawler.CrawlConfig {
	if pc == nil {
		return &crawler.CrawlConfig{}
	}
	cfg := &crawler.CrawlConfig{
		URLs: pc.GetUrls(),
	}
	if opts := pc.GetOptions(); opts != nil {
		cfg.MaxDepth = opts.GetMaxDepth()
		cfg.MaxPages = opts.GetMaxPages()
		cfg.RespectRobotsTxt = opts.GetRespectRobotsTxt()
	}
	return cfg
}

// generateID creates a random hex-encoded identifier.
func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// clampInt32 safely converts an int to int32, clamping to [MinInt32, MaxInt32].
func clampInt32(v int) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	if v < math.MinInt32 {
		return math.MinInt32
	}
	return int32(v)
}
