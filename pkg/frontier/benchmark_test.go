package frontier

import (
	"context"
	"fmt"
	"testing"
)

// BenchmarkFrontier_Add measures the throughput of adding URL entries.
//
// Run with:
//
//	go test -bench=BenchmarkFrontier_Add -benchtime=5s ./pkg/frontier/
func BenchmarkFrontier_Add(b *testing.B) {
	f := NewMemoryFrontier(Config{CrawlDelay: 0})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = f.Add(&URLEntry{
			URL:      fmt.Sprintf("http://example.com/%d", i),
			Priority: PriorityNormal,
		})
	}
}

// BenchmarkFrontier_AddNext measures the round-trip throughput of add + consume.
//
// Run with:
//
//	go test -bench=BenchmarkFrontier_AddNext -benchtime=5s ./pkg/frontier/
func BenchmarkFrontier_AddNext(b *testing.B) {
	f := NewMemoryFrontier(Config{CrawlDelay: 0})
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = f.Add(&URLEntry{
			URL:      fmt.Sprintf("http://example.com/%d", i),
			Priority: PriorityNormal,
		})
		_, _ = f.Next(ctx)
	}
}
