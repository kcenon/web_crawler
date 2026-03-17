package plugin

import (
	"context"
	"strings"
	"testing"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

// testParser is a minimal ParserPlugin for router testing.
type testParser struct {
	name string
	ct   string // content type prefix to match
}

func (p *testParser) Name() string              { return p.name }
func (p *testParser) Init(map[string]any) error { return nil }
func (p *testParser) Close() error              { return nil }
func (p *testParser) CanParse(ct string) bool   { return strings.Contains(ct, p.ct) }
func (p *testParser) Parse(_ context.Context, resp *crawler.CrawlResponse) (*ParseResult, error) {
	return &ParseResult{
		Data: map[string]any{"parser": p.name, "url": resp.Request.URL},
	}, nil
}

func TestParserRouter_MatchesCorrectParser(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterParser("html", &testParser{name: "html", ct: "text/html"})
	_ = r.RegisterParser("json", &testParser{name: "json", ct: "application/json"})

	router := NewParserRouter(r)

	resp := &crawler.CrawlResponse{
		Request:     &crawler.CrawlRequest{URL: "https://example.com"},
		ContentType: "text/html; charset=utf-8",
		Body:        []byte("<html></html>"),
	}

	result, err := router.Parse(context.Background(), resp)
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["parser"] != "html" {
		t.Errorf("expected html parser, got %v", result.Data["parser"])
	}
}

func TestParserRouter_JSONContentType(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterParser("html", &testParser{name: "html", ct: "text/html"})
	_ = r.RegisterParser("json", &testParser{name: "json", ct: "application/json"})

	router := NewParserRouter(r)

	resp := &crawler.CrawlResponse{
		Request:     &crawler.CrawlRequest{URL: "https://api.example.com"},
		ContentType: "application/json",
		Body:        []byte(`{"key": "value"}`),
	}

	result, err := router.Parse(context.Background(), resp)
	if err != nil {
		t.Fatal(err)
	}
	if result.Data["parser"] != "json" {
		t.Errorf("expected json parser, got %v", result.Data["parser"])
	}
}

func TestParserRouter_NoMatchingParser(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterParser("html", &testParser{name: "html", ct: "text/html"})

	router := NewParserRouter(r)

	resp := &crawler.CrawlResponse{
		Request:     &crawler.CrawlRequest{URL: "https://example.com"},
		ContentType: "application/pdf",
		Body:        []byte{},
	}

	_, err := router.Parse(context.Background(), resp)
	if err == nil {
		t.Error("expected error for unhandled content type")
	}
	if !strings.Contains(err.Error(), "application/pdf") {
		t.Errorf("error should mention content type, got: %v", err)
	}
}

func TestParserRouter_EmptyRegistry(t *testing.T) {
	r := NewRegistry()
	router := NewParserRouter(r)

	resp := &crawler.CrawlResponse{
		Request:     &crawler.CrawlRequest{URL: "https://example.com"},
		ContentType: "text/html",
		Body:        []byte("<html></html>"),
	}

	_, err := router.Parse(context.Background(), resp)
	if err == nil {
		t.Error("expected error for empty registry")
	}
}
