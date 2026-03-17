// Package main demonstrates how to build custom plugins and wire them
// together using the plugin registry and YAML configuration.
//
// It shows:
//   - A custom in-memory storage plugin
//   - A custom logging notifier plugin
//   - Registration via factory + YAML config
//   - Using the ParserRouter for content-type dispatch
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/plugin"
	"github.com/kcenon/web_crawler/pkg/plugin/parser"
	"github.com/kcenon/web_crawler/pkg/storage"
)

// --- Custom Storage Plugin -------------------------------------------

// MemoryStorage keeps crawled items in memory. Useful for testing or
// small crawl jobs where persistence is not needed.
type MemoryStorage struct {
	mu    sync.Mutex
	items []storage.Item
}

func (m *MemoryStorage) Name() string              { return "memory" }
func (m *MemoryStorage) Init(map[string]any) error { return nil }
func (m *MemoryStorage) Close() error              { return nil }

func (m *MemoryStorage) Store(_ context.Context, items []storage.Item) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = append(m.items, items...)
	return nil
}

func (m *MemoryStorage) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.items)
}

// --- Custom Notifier Plugin ------------------------------------------

// LogNotifier prints crawl events to the console.
type LogNotifier struct{}

func (n *LogNotifier) Name() string              { return "log" }
func (n *LogNotifier) Init(map[string]any) error { return nil }
func (n *LogNotifier) Close() error              { return nil }

func (n *LogNotifier) Notify(_ context.Context, event *plugin.CrawlEvent) error {
	fmt.Printf("[%s] %s\n", event.Type, event.Message)
	return nil
}

// --- Main ------------------------------------------------------------

func main() {
	// 1. Create a plugin registry.
	registry := plugin.NewRegistry()

	// 2. Register custom plugins.
	mem := &MemoryStorage{}
	if err := registry.RegisterStorage("memory", mem); err != nil {
		log.Fatal(err)
	}

	logNotifier := &LogNotifier{}
	if err := registry.RegisterNotifier("log", logNotifier); err != nil {
		log.Fatal(err)
	}

	// 3. Register the built-in HTML parser.
	htmlParser := parser.NewHTMLParser(map[string]string{
		"title": "title",
		"h1":    "h1",
	})
	if err := registry.RegisterParser("html", htmlParser); err != nil {
		log.Fatal(err)
	}

	// 4. Use the ParserRouter for automatic content-type dispatch.
	router := plugin.NewParserRouter(registry)

	// 5. Build and run a crawler.
	c, err := crawler.NewBuilder().
		WithMaxDepth(1).
		WithMaxPages(5).
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Notify on start.
	_ = logNotifier.Notify(context.Background(), &plugin.CrawlEvent{
		Type:    plugin.EventStarted,
		Message: "crawl starting",
	})

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		// Parse the response.
		result, parseErr := router.Parse(context.Background(), resp)
		if parseErr != nil {
			fmt.Printf("  skip non-HTML: %s\n", resp.Request.URL)
			return
		}

		// Store the result.
		item := storage.Item{
			URL:       resp.Request.URL,
			Data:      result.Data,
			CrawledAt: time.Now(),
		}
		_ = mem.Store(context.Background(), []storage.Item{item})

		fmt.Printf("  stored: %s (links: %d)\n", resp.Request.URL, len(result.Links))
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

	// Notify on completion.
	_ = logNotifier.Notify(context.Background(), &plugin.CrawlEvent{
		Type:    plugin.EventCompleted,
		Message: fmt.Sprintf("crawl finished, %d items stored", mem.Count()),
	})

	// 6. Clean up all plugins.
	if errs := registry.CloseAll(); len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("close error: %v\n", e)
		}
	}
}
