package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/kcenon/web_crawler/pkg/crawler"
	pb "github.com/kcenon/web_crawler/pkg/server/pb"
)

// Config holds the server configuration.
type Config struct {
	Port         int
	DrainTimeout time.Duration
}

func (c Config) withDefaults() Config {
	if c.Port == 0 {
		c.Port = 50051
	}
	if c.DrainTimeout == 0 {
		c.DrainTimeout = 30 * time.Second
	}
	return c
}

// crawlerInstance tracks a running crawler managed by StartCrawler/StopCrawler.
type crawlerInstance struct {
	status pb.CrawlerStatus
	config *crawler.CrawlConfig
}

// Server implements the CrawlerService gRPC server. It wraps a Crawler
// interface and translates between protobuf messages and Go domain types.
type Server struct {
	pb.UnimplementedCrawlerServiceServer

	crawler  crawler.Service
	cfg      Config
	grpc     *grpc.Server
	health   *health.Server
	listener net.Listener
	logger   *slog.Logger

	mu        sync.RWMutex
	instances map[string]*crawlerInstance
}

// New creates a new Server with the given Crawler implementation and config.
func New(c crawler.Service, cfg Config, opts ...Option) *Server {
	cfg = cfg.withDefaults()
	s := &Server{
		crawler:   c,
		cfg:       cfg,
		logger:    slog.Default(),
		instances: make(map[string]*crawlerInstance),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Option configures optional server parameters.
type Option func(*Server)

// WithLogger sets a custom slog.Logger for the server.
func WithLogger(l *slog.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// Start creates a TCP listener, registers gRPC services, and begins serving.
// It blocks until the server is stopped or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", s.cfg.Port, err)
	}

	s.grpc = grpc.NewServer(
		grpc.ChainUnaryInterceptor(s.unaryLoggingInterceptor),
		grpc.ChainStreamInterceptor(s.streamLoggingInterceptor),
	)

	pb.RegisterCrawlerServiceServer(s.grpc, s)

	s.health = health.NewServer()
	healthpb.RegisterHealthServer(s.grpc, s.health)
	s.health.SetServingStatus("crawler.v1.CrawlerService", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(s.grpc)

	s.logger.Info("gRPC server starting", "addr", s.listener.Addr().String())

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.grpc.Serve(s.listener)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.Stop(context.Background()) //nolint:errcheck // best-effort on context cancel
		return ctx.Err()
	}
}

// Stop performs a graceful shutdown, waiting up to DrainTimeout before
// forcing a hard stop.
func (s *Server) Stop(_ context.Context) error {
	if s.grpc == nil {
		return nil
	}

	s.health.SetServingStatus("crawler.v1.CrawlerService", healthpb.HealthCheckResponse_NOT_SERVING)

	done := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("gRPC server stopped gracefully")
	case <-time.After(s.cfg.DrainTimeout):
		s.logger.Warn("drain timeout exceeded, forcing stop")
		s.grpc.Stop()
	}
	return nil
}

// Addr returns the listener address. Useful in tests with port 0.
func (s *Server) Addr() net.Addr {
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

// unaryLoggingInterceptor logs method, duration, and status for unary RPCs.
func (s *Server) unaryLoggingInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	duration := time.Since(start)

	code := status.Code(err)
	s.logger.Info("unary rpc",
		"method", info.FullMethod,
		"duration", duration,
		"code", code.String(),
	)
	return resp, err
}

// streamLoggingInterceptor logs method, duration, and status for streaming RPCs.
func (s *Server) streamLoggingInterceptor(
	srv any,
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()
	err := handler(srv, ss)
	duration := time.Since(start)

	code := status.Code(err)
	s.logger.Info("stream rpc",
		"method", info.FullMethod,
		"duration", duration,
		"code", code.String(),
	)
	return err
}
