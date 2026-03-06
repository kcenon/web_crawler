package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/frontier"
	"github.com/kcenon/web_crawler/pkg/storage"
)

func BenchmarkCrawlSinglePage(b *testing.B) {
	srv := NewHTTPServer(
		Page{Path: "/", Body: "<html><body>Hello</body></html>"},
	)
	defer srv.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := crawler.NewEngine(crawler.Config{
			MaxDepth:    1,
			MaxPages:    1,
			WorkerCount: 1,
		})

		done := make(chan struct{})
		c.OnResponse(func(_ *crawler.CrawlResponse) {
			close(done)
		})

		_ = c.AddURL(srv.URL + "/")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = c.Start(ctx)

		select {
		case <-done:
		case <-ctx.Done():
		}

		_ = c.Wait()
		cancel()
	}
}

func BenchmarkFrontierAddAndNext(b *testing.B) {
	b.ReportAllocs()

	f := frontier.NewMemoryFrontier(frontier.Config{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Add(&frontier.URLEntry{URL: "https://example.com/page", Priority: 0})
		_, _ = f.Next(ctx)
	}
}

func BenchmarkStorageFileWrite(b *testing.B) {
	dir := b.TempDir()

	fs := storage.NewFileStorage(storage.FileConfig{
		Path: dir + "/bench.jsonl",
	})
	_ = fs.Init(nil)
	defer fs.Close()

	items := []storage.Item{
		{
			URL:       "https://example.com/page",
			Data:      map[string]any{"title": "Test Page"},
			CrawledAt: time.Now(),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = fs.Store(context.Background(), items)
	}
}

func BenchmarkStorageCSVWrite(b *testing.B) {
	dir := b.TempDir()

	cs := storage.NewCSVStorage(storage.CSVConfig{
		Path:    dir + "/bench.csv",
		Columns: []string{"url", "data.title", "crawled_at"},
	})
	_ = cs.Init(nil)
	defer cs.Close()

	items := []storage.Item{
		{
			URL:       "https://example.com/page",
			Data:      map[string]any{"title": "Test Page"},
			CrawledAt: time.Now(),
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cs.Store(context.Background(), items)
	}
}

func BenchmarkCrawlConcurrent(b *testing.B) {
	srv := NewHTTPServer(
		Page{Path: "/", Body: "<html><body>Hello</body></html>"},
	)
	defer srv.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c := crawler.NewEngine(crawler.Config{
			MaxDepth:    1,
			MaxPages:    10,
			WorkerCount: 4,
		})

		var wg sync.WaitGroup
		wg.Add(4)
		c.OnResponse(func(_ *crawler.CrawlResponse) {
			wg.Done()
		})

		for j := 0; j < 4; j++ {
			_ = c.AddURL(srv.URL + "/")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = c.Start(ctx)
		wg.Wait()
		_ = c.Wait()
		cancel()
	}
}
