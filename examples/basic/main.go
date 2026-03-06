// Package main demonstrates basic usage of the web crawler SDK.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

func main() {
	c, err := crawler.NewBuilder().
		WithMaxDepth(2).
		WithWorkerCount(5).
		WithUserAgent("web_crawler-example/0.1").
		Build()
	if err != nil {
		log.Fatal(err)
	}

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		fmt.Printf("Crawled: %s [%d] (%d bytes)\n",
			resp.Request.URL, resp.StatusCode, len(resp.Body))
	})

	c.OnError(func(req *crawler.CrawlRequest, err error) {
		fmt.Printf("Error: %s - %v\n", req.URL, err)
	})

	if err := c.AddURL("https://example.com"); err != nil {
		log.Fatal(err)
	}

	if err := c.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	if err := c.Wait(); err != nil {
		log.Fatal(err)
	}

	stats := c.Stats()
	fmt.Printf("\nCompleted: %d requests, %d success, %d errors\n",
		stats.RequestCount, stats.SuccessCount, stats.ErrorCount)
}
