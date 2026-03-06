package observability

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// newTestMetrics creates isolated Metrics with a fresh registry for testing.
func newTestMetrics(t *testing.T) (*Metrics, *prometheus.Registry) {
	t.Helper()
	reg := prometheus.NewRegistry()

	m := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "crawler_requests_total",
			Help: "Total number of crawl requests by status and domain.",
		}, []string{"status", "domain"}),

		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "crawler_request_duration_seconds",
			Help:    "Duration of crawl requests in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"domain"}),

		ActiveRequests: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_active_requests",
			Help: "Number of currently active crawl requests.",
		}),

		URLsQueued: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "crawler_urls_queued",
			Help: "Number of URLs waiting in the crawl queue.",
		}),

		BytesDownloaded: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "crawler_bytes_downloaded_total",
			Help: "Total bytes downloaded by the crawler.",
		}),

		ErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "crawler_errors_total",
			Help: "Total number of crawl errors by type.",
		}, []string{"type"}),
	}

	reg.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.ActiveRequests,
		m.URLsQueued,
		m.BytesDownloaded,
		m.ErrorsTotal,
	)

	return m, reg
}

func getCounterValue(cv *prometheus.CounterVec, labels ...string) float64 {
	var metric dto.Metric
	counter, err := cv.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	if err := counter.Write(&metric); err != nil {
		return 0
	}
	return metric.GetCounter().GetValue()
}

func getGaugeValue(g prometheus.Gauge) float64 {
	var metric dto.Metric
	if err := g.Write(&metric); err != nil {
		return 0
	}
	return metric.GetGauge().GetValue()
}

func getCounterScalarValue(c prometheus.Counter) float64 {
	var metric dto.Metric
	if err := c.Write(&metric); err != nil {
		return 0
	}
	return metric.GetCounter().GetValue()
}

func TestNewMetrics_Registration(t *testing.T) {
	m := NewMetrics()
	if m.RequestsTotal == nil {
		t.Error("RequestsTotal not initialized")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration not initialized")
	}
	if m.ActiveRequests == nil {
		t.Error("ActiveRequests not initialized")
	}
	if m.URLsQueued == nil {
		t.Error("URLsQueued not initialized")
	}
	if m.BytesDownloaded == nil {
		t.Error("BytesDownloaded not initialized")
	}
	if m.ErrorsTotal == nil {
		t.Error("ErrorsTotal not initialized")
	}
}

func TestMetrics_RecordRequest(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordRequest("200", "example.com", 150*time.Millisecond)
	m.RecordRequest("200", "example.com", 200*time.Millisecond)
	m.RecordRequest("404", "example.com", 50*time.Millisecond)

	if v := getCounterValue(m.RequestsTotal, "200", "example.com"); v != 2 {
		t.Errorf("expected 2 requests with status 200, got %v", v)
	}
	if v := getCounterValue(m.RequestsTotal, "404", "example.com"); v != 1 {
		t.Errorf("expected 1 request with status 404, got %v", v)
	}
}

func TestMetrics_RecordError(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.RecordError("timeout")
	m.RecordError("timeout")
	m.RecordError("connection")

	if v := getCounterValue(m.ErrorsTotal, "timeout"); v != 2 {
		t.Errorf("expected 2 timeout errors, got %v", v)
	}
	if v := getCounterValue(m.ErrorsTotal, "connection"); v != 1 {
		t.Errorf("expected 1 connection error, got %v", v)
	}
}

func TestMetrics_Gauges(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.ActiveRequests.Set(5)
	if v := getGaugeValue(m.ActiveRequests); v != 5 {
		t.Errorf("expected active_requests=5, got %v", v)
	}

	m.ActiveRequests.Dec()
	if v := getGaugeValue(m.ActiveRequests); v != 4 {
		t.Errorf("expected active_requests=4, got %v", v)
	}

	m.URLsQueued.Set(100)
	if v := getGaugeValue(m.URLsQueued); v != 100 {
		t.Errorf("expected urls_queued=100, got %v", v)
	}
}

func TestMetrics_BytesDownloaded(t *testing.T) {
	m, _ := newTestMetrics(t)

	m.BytesDownloaded.Add(1024)
	m.BytesDownloaded.Add(2048)

	if v := getCounterScalarValue(m.BytesDownloaded); v != 3072 {
		t.Errorf("expected bytes_downloaded=3072, got %v", v)
	}
}

func TestMetricsServer_StartsAndStops(t *testing.T) {
	srv := NewMetricsServer(MetricsServerConfig{
		Port: 0, // will use default 9090, but we test creation
	})

	if srv == nil {
		t.Fatal("expected non-nil MetricsServer")
	}
	if srv.cfg.Path != "/metrics" {
		t.Errorf("expected default path /metrics, got %s", srv.cfg.Path)
	}
}

func TestMetricsServer_ServesMetrics(t *testing.T) {
	// Use a high port to avoid conflicts
	port := 19091
	srv := NewMetricsServer(MetricsServerConfig{
		Port: port,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	if err != nil {
		t.Fatalf("failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	if !strings.Contains(body, "go_") {
		t.Error("expected Go runtime metrics in output")
	}

	cancel()
	<-errCh
}
