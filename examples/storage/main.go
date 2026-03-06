// Package main demonstrates crawling with file-based storage output.
//
// This example crawls a URL and saves results to both JSON Lines
// and CSV formats using the storage plugins.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/storage"
)

func main() {
	// Set up JSON Lines storage.
	jsonStore := storage.NewFileStorage(storage.FileConfig{
		Path: "output/results.jsonl",
	})
	if err := jsonStore.Init(nil); err != nil {
		log.Fatal(err)
	}
	defer jsonStore.Close()

	// Set up CSV storage.
	csvStore := storage.NewCSVStorage(storage.CSVConfig{
		Path:    "output/results.csv",
		Columns: []string{"url", "data.title", "data.status", "crawled_at"},
	})
	if err := csvStore.Init(nil); err != nil {
		log.Fatal(err)
	}
	defer csvStore.Close()

	// Build the crawler.
	c, err := crawler.NewBuilder().
		WithMaxDepth(1).
		WithMaxPages(10).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// On each response, store the result.
	c.OnResponse(func(resp *crawler.CrawlResponse) {
		item := storage.Item{
			URL: resp.Request.URL,
			Data: map[string]any{
				"title":  fmt.Sprintf("Page at %s", resp.Request.URL),
				"status": resp.StatusCode,
				"size":   len(resp.Body),
			},
			CrawledAt: time.Now(),
		}

		_ = jsonStore.Store(context.Background(), []storage.Item{item})
		_ = csvStore.Store(context.Background(), []storage.Item{item})

		fmt.Printf("Stored: %s [%d]\n", resp.Request.URL, resp.StatusCode)
	})

	if err := c.AddURL("https://example.com"); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Start(ctx); err != nil {
		log.Fatal(err)
	}
	_ = c.Wait()

	fmt.Println("\nResults saved to output/results.jsonl and output/results.csv")
}
