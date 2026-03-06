package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metrics for the crawler.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
	URLsQueued      prometheus.Gauge
	BytesDownloaded prometheus.Counter
	ErrorsTotal     *prometheus.CounterVec
}

// NewMetrics creates and registers Prometheus metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "crawler_requests_total",
			Help: "Total number of crawl requests by status and domain.",
		}, []string{"status", "domain"}),

		RequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "crawler_request_duration_seconds",
			Help:    "Duration of crawl requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"domain"}),

		ActiveRequests: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_active_requests",
			Help: "Number of currently active crawl requests.",
		}),

		URLsQueued: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_urls_queued",
			Help: "Number of URLs waiting in the crawl queue.",
		}),

		BytesDownloaded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "crawler_bytes_downloaded_total",
			Help: "Total bytes downloaded by the crawler.",
		}),

		ErrorsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "crawler_errors_total",
			Help: "Total number of crawl errors by type.",
		}, []string{"type"}),
	}
}

// RecordRequest records a completed request with its status, domain, and duration.
func (m *Metrics) RecordRequest(status, domain string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(status, domain).Inc()
	m.RequestDuration.WithLabelValues(domain).Observe(duration.Seconds())
}

// RecordError increments the error counter for the given error type.
func (m *Metrics) RecordError(errType string) {
	m.ErrorsTotal.WithLabelValues(errType).Inc()
}

// MetricsServerConfig configures the Prometheus metrics HTTP server.
type MetricsServerConfig struct {
	// Port is the port to serve metrics on. Default: 9090.
	Port int

	// Path is the HTTP path for the metrics endpoint. Default: "/metrics".
	Path string

	// Logger is used for server lifecycle logging.
	Logger *slog.Logger
}

// MetricsServer exposes Prometheus metrics via HTTP.
type MetricsServer struct {
	cfg    MetricsServerConfig
	server *http.Server
}

// NewMetricsServer creates a new metrics HTTP server.
func NewMetricsServer(cfg MetricsServerConfig) *MetricsServer {
	if cfg.Port == 0 {
		cfg.Port = 9090
	}
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	mux := http.NewServeMux()
	mux.Handle(cfg.Path, promhttp.Handler())

	return &MetricsServer{
		cfg: cfg,
		server: &http.Server{
			Addr:              fmt.Sprintf(":%d", cfg.Port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Start begins serving metrics. It blocks until the server is stopped or
// the context is cancelled.
func (s *MetricsServer) Start(ctx context.Context) error {
	s.cfg.Logger.Info("metrics server starting", "addr", s.server.Addr, "path", s.cfg.Path)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
