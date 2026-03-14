package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark <url>",
	Short: "Benchmark crawler throughput and latency",
	Long: `Run a load test against a target URL and report throughput, latency
percentiles, and memory usage.

If --requests is specified the benchmark runs until that many requests are
completed. Otherwise it runs for --duration and counts however many requests
finish within that window.

Examples:
  crawler benchmark https://example.com
  crawler benchmark https://example.com --concurrency 20 --duration 30s
  crawler benchmark https://example.com --concurrency 10 --requests 1000`,
	Args: cobra.ExactArgs(1),
	RunE: runBenchmark,
}

var (
	benchConcurrency int
	benchDuration    time.Duration
	benchRequests    int
)

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	benchmarkCmd.Flags().IntVarP(&benchConcurrency, "concurrency", "c", 10, "number of concurrent workers")
	benchmarkCmd.Flags().DurationVarP(&benchDuration, "duration", "d", 30*time.Second, "benchmark duration (used when --requests is 0)")
	benchmarkCmd.Flags().IntVarP(&benchRequests, "requests", "n", 0, "total requests to send (0 = run for --duration)")
}

const benchStartNsKey = "bench.start_ns"

// benchmarkReport holds the final benchmark results.
type benchmarkReport struct {
	URL         string  `json:"url"`
	Workers     int     `json:"workers"`
	TotalSent   int     `json:"total_requests_sent"`
	Successful  int64   `json:"successful"`
	Failed      int64   `json:"failed"`
	WallTime    string  `json:"wall_time"`
	Throughput  float64 `json:"throughput_req_per_sec"`
	LatencyP50  string  `json:"latency_p50_ms"`
	LatencyP95  string  `json:"latency_p95_ms"`
	LatencyP99  string  `json:"latency_p99_ms"`
	MemUsageMB  float64 `json:"mem_alloc_mb"`
}

func runBenchmark(_ *cobra.Command, args []string) error {
	targetURL := args[0]

	numRequests := benchRequests
	if numRequests <= 0 {
		// Estimate: fill the queue with enough URLs to stay busy for the
		// requested duration. Assume each worker can do ~100 req/s at best.
		numRequests = benchConcurrency * int(benchDuration.Seconds()) * 100
		if numRequests < benchConcurrency*10 {
			numRequests = benchConcurrency * 10
		}
	}

	slog.Info("starting benchmark",
		"url", targetURL,
		"concurrency", benchConcurrency,
		"requests", numRequests,
	)

	cfg := crawler.Config{
		MaxDepth:    1,
		WorkerCount: benchConcurrency,
	}
	c := crawler.NewEngine(cfg)

	var (
		latMu     sync.Mutex
		latencies []float64
	)

	c.OnRequest(func(req *crawler.CrawlRequest) {
		if req.Meta == nil {
			req.Meta = make(map[string]string)
		}
		req.Meta[benchStartNsKey] = strconv.FormatInt(time.Now().UnixNano(), 10)
	})

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		if ns, ok := resp.Request.Meta[benchStartNsKey]; ok {
			startNs, err := strconv.ParseInt(ns, 10, 64)
			if err == nil {
				ms := float64(time.Now().UnixNano()-startNs) / 1e6
				latMu.Lock()
				latencies = append(latencies, ms)
				latMu.Unlock()
			}
		}
	})

	urls := make([]string, numRequests)
	for i := range urls {
		urls[i] = targetURL
	}
	if err := c.AddURLs(urls); err != nil {
		return fmt.Errorf("add URLs: %w", err)
	}

	var ctx context.Context
	var cancel context.CancelFunc
	if benchRequests <= 0 {
		ctx, cancel = context.WithTimeout(context.Background(), benchDuration)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	wallStart := time.Now()

	if err := c.Start(ctx); err != nil {
		slog.Debug("benchmark start", "error", err)
	}
	if err := c.Wait(); err != nil {
		slog.Debug("benchmark wait", "error", err)
	}

	elapsed := time.Since(wallStart)
	stats := c.Stats()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	report := benchmarkReport{
		URL:        targetURL,
		Workers:    benchConcurrency,
		TotalSent:  numRequests,
		Successful: stats.SuccessCount,
		Failed:     stats.ErrorCount,
		WallTime:   elapsed.Round(time.Millisecond).String(),
		MemUsageMB: math.Round(float64(memStats.Alloc)/1024/1024*100) / 100,
	}

	if elapsed.Seconds() > 0 {
		report.Throughput = math.Round(float64(stats.SuccessCount)/elapsed.Seconds()*100) / 100
	}

	latMu.Lock()
	lats := latencies
	latMu.Unlock()

	if len(lats) > 0 {
		sort.Float64s(lats)
		report.LatencyP50 = fmt.Sprintf("%.2f ms", percentile(lats, 50))
		report.LatencyP95 = fmt.Sprintf("%.2f ms", percentile(lats, 95))
		report.LatencyP99 = fmt.Sprintf("%.2f ms", percentile(lats, 99))
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// percentile returns the p-th percentile value from a sorted slice.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(p/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
