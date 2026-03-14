package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkEngine_Throughput measures how many URLs the engine processes
// per second against a local httptest server with no network latency.
//
// Run with:
//
//	go test -bench=BenchmarkEngine_Throughput -benchtime=5s ./pkg/crawler/
func BenchmarkEngine_Throughput(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	const batchSize = 50
	urls := make([]string, batchSize)
	for i := range urls {
		urls[i] = fmt.Sprintf("%s/%d", ts.URL, i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c, err := NewBuilder().WithWorkerCount(10).Build()
		if err != nil {
			b.Fatal(err)
		}
		if err := c.AddURLs(urls); err != nil {
			b.Fatal(err)
		}
		if err := c.Start(context.Background()); err != nil {
			b.Fatal(err)
		}
		if err := c.Wait(); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportMetric(float64(batchSize), "urls/op")
}

// BenchmarkEngine_WorkerScaling measures how throughput scales with worker count.
// Run with: go test -bench=BenchmarkEngine_WorkerScaling -benchtime=3s ./pkg/crawler/
func BenchmarkEngine_WorkerScaling(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer ts.Close()

	for _, workers := range []int{1, 5, 10, 20, 50} {
		workers := workers
		b.Run(fmt.Sprintf("workers=%d", workers), func(b *testing.B) {
			const batchSize = 100
			urls := make([]string, batchSize)
			for i := range urls {
				urls[i] = fmt.Sprintf("%s/%d", ts.URL, i)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				c, err := NewBuilder().WithWorkerCount(workers).Build()
				if err != nil {
					b.Fatal(err)
				}
				_ = c.AddURLs(urls)
				_ = c.Start(context.Background())
				_ = c.Wait()
			}

			b.ReportMetric(float64(batchSize), "urls/op")
		})
	}
}
