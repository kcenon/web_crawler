package testutil

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

func TestIntegration_CrawlSinglePage(t *testing.T) {
	srv := NewHTTPServer(
		Page{
			Path:        "/",
			ContentType: "text/html",
			Body:        "<html><body><h1>Hello World</h1></body></html>",
		},
	)
	defer srv.Close()

	c := crawler.NewEngine(crawler.Config{
		MaxDepth:    1,
		MaxPages:    1,
		WorkerCount: 1,
	})

	var (
		mu       sync.Mutex
		gotURL   string
		gotCode  int
		gotBody  int
		gotError string
	)

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		mu.Lock()
		gotURL = resp.Request.URL
		gotCode = resp.StatusCode
		gotBody = len(resp.Body)
		mu.Unlock()
	})

	c.OnError(func(_ *crawler.CrawlRequest, err error) {
		mu.Lock()
		gotError = err.Error()
		mu.Unlock()
	})

	AssertNoError(t, c.AddURL(srv.URL+"/"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	AssertNoError(t, c.Start(ctx))
	_ = c.Wait()

	mu.Lock()
	defer mu.Unlock()

	if gotError != "" {
		t.Fatalf("unexpected crawl error: %s", gotError)
	}
	AssertEqual(t, gotURL, srv.URL+"/")
	AssertEqual(t, gotCode, 200)
	if gotBody == 0 {
		t.Error("expected non-empty response body")
	}
}

func TestIntegration_CrawlMultiplePages(t *testing.T) {
	srv := NewHTTPServer(
		Page{Path: "/a", Body: "Page A"},
		Page{Path: "/b", Body: "Page B"},
		Page{Path: "/c", Body: "Page C"},
	)
	defer srv.Close()

	c := crawler.NewEngine(crawler.Config{
		MaxDepth:    1,
		MaxPages:    10,
		WorkerCount: 3,
	})

	var (
		mu   sync.Mutex
		urls []string
	)

	c.OnResponse(func(resp *crawler.CrawlResponse) {
		mu.Lock()
		urls = append(urls, resp.Request.URL)
		mu.Unlock()
	})

	AssertNoError(t, c.AddURL(srv.URL+"/a"))
	AssertNoError(t, c.AddURL(srv.URL+"/b"))
	AssertNoError(t, c.AddURL(srv.URL+"/c"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	AssertNoError(t, c.Start(ctx))
	_ = c.Wait()

	mu.Lock()
	defer mu.Unlock()

	AssertEqual(t, len(urls), 3)
}

func TestIntegration_CrawlHandlesErrors(t *testing.T) {
	srv := NewHTTPServer(
		Page{Path: "/ok", StatusCode: 200, Body: "OK"},
		Page{Path: "/fail", StatusCode: 500, Body: "Error"},
	)
	defer srv.Close()

	c := crawler.NewEngine(crawler.Config{
		MaxDepth:    1,
		MaxPages:    10,
		WorkerCount: 2,
	})

	var (
		mu        sync.Mutex
		responses int
	)

	c.OnResponse(func(_ *crawler.CrawlResponse) {
		mu.Lock()
		responses++
		mu.Unlock()
	})

	AssertNoError(t, c.AddURL(srv.URL+"/ok"))
	AssertNoError(t, c.AddURL(srv.URL+"/fail"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	AssertNoError(t, c.Start(ctx))
	_ = c.Wait()

	mu.Lock()
	defer mu.Unlock()

	// Both pages return responses (500 is still a response, not an error)
	AssertEqual(t, responses, 2)
}

func TestIntegration_CrawlWithHeaders(t *testing.T) {
	var gotUA string
	srv := NewHTTPServer(
		Page{Path: "/", Body: "OK"},
	)
	// Replace handler to capture user-agent
	origHandler := srv.Config.Handler
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		origHandler.ServeHTTP(w, r)
	})
	defer srv.Close()

	c := crawler.NewEngine(crawler.Config{
		MaxDepth:    1,
		MaxPages:    1,
		WorkerCount: 1,
		UserAgent:   "test-crawler/1.0",
	})

	c.OnResponse(func(_ *crawler.CrawlResponse) {})

	AssertNoError(t, c.AddURL(srv.URL+"/"))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	AssertNoError(t, c.Start(ctx))
	_ = c.Wait()

	AssertEqual(t, gotUA, "test-crawler/1.0")
}
