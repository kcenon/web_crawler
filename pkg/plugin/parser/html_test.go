package parser

import (
	"context"
	"testing"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

const testHTML = `<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
  <h1>Hello World</h1>
  <p class="desc">A description</p>
  <a href="https://example.com/page1">Link 1</a>
  <a href="https://example.com/page2">Link 2</a>
  <a href="#">Empty</a>
  <a href="">Blank</a>
</body>
</html>`

func resp(ct string, body string) *crawler.CrawlResponse {
	return &crawler.CrawlResponse{
		Request:     &crawler.CrawlRequest{URL: "https://example.com"},
		StatusCode:  200,
		Body:        []byte(body),
		ContentType: ct,
	}
}

func TestHTMLParser_Name(t *testing.T) {
	p := NewHTMLParser(nil)
	if p.Name() != "html" {
		t.Errorf("Name() = %q, want %q", p.Name(), "html")
	}
}

func TestHTMLParser_CanParse(t *testing.T) {
	p := NewHTMLParser(nil)

	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := p.CanParse(tt.ct); got != tt.want {
			t.Errorf("CanParse(%q) = %v, want %v", tt.ct, got, tt.want)
		}
	}
}

func TestHTMLParser_Parse_LinksOnly(t *testing.T) {
	p := NewHTMLParser(nil)
	result, err := p.Parse(context.Background(), resp("text/html", testHTML))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Links) != 2 {
		t.Fatalf("got %d links, want 2", len(result.Links))
	}
	if result.Links[0] != "https://example.com/page1" {
		t.Errorf("links[0] = %q", result.Links[0])
	}
	if result.Links[1] != "https://example.com/page2" {
		t.Errorf("links[1] = %q", result.Links[1])
	}
}

func TestHTMLParser_Parse_WithRules(t *testing.T) {
	rules := map[string]string{
		"title": "title",
		"h1":    "h1",
		"desc":  "p.desc",
	}
	p := NewHTMLParser(rules)
	result, err := p.Parse(context.Background(), resp("text/html", testHTML))
	if err != nil {
		t.Fatal(err)
	}

	if result.Data["title"] != "Test Page" {
		t.Errorf("title = %v, want %q", result.Data["title"], "Test Page")
	}
	if result.Data["h1"] != "Hello World" {
		t.Errorf("h1 = %v, want %q", result.Data["h1"], "Hello World")
	}
	if result.Data["desc"] != "A description" {
		t.Errorf("desc = %v, want %q", result.Data["desc"], "A description")
	}
}

func TestHTMLParser_Parse_MultipleMatches(t *testing.T) {
	html := `<ul><li>A</li><li>B</li><li>C</li></ul>`
	p := NewHTMLParser(map[string]string{"items": "li"})
	result, err := p.Parse(context.Background(), resp("text/html", html))
	if err != nil {
		t.Fatal(err)
	}

	items, ok := result.Data["items"].([]string)
	if !ok {
		t.Fatalf("items is %T, want []string", result.Data["items"])
	}
	if len(items) != 3 {
		t.Errorf("got %d items, want 3", len(items))
	}
}

func TestHTMLParser_Parse_NoLinks(t *testing.T) {
	html := `<p>No links here</p>`
	p := NewHTMLParser(nil)
	result, err := p.Parse(context.Background(), resp("text/html", html))
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Links) != 0 {
		t.Errorf("got %d links, want 0", len(result.Links))
	}
}

func TestHTMLParser_InitAndClose(t *testing.T) {
	p := NewHTMLParser(nil)
	if err := p.Init(nil); err != nil {
		t.Errorf("Init() error = %v", err)
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
