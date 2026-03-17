// Package parser provides built-in ParserPlugin implementations.
package parser

import (
	"bytes"
	"context"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/plugin"
)

// HTMLParser extracts structured data and links from HTML responses
// using GoQuery CSS selectors.
type HTMLParser struct {
	// Rules maps field names to CSS selectors for data extraction.
	// Each selector's text content is placed under the corresponding key
	// in ParseResult.Data. If nil, only link extraction is performed.
	Rules map[string]string
}

// NewHTMLParser creates an HTMLParser with the given extraction rules.
// Pass nil for link-only extraction.
func NewHTMLParser(rules map[string]string) *HTMLParser {
	return &HTMLParser{Rules: rules}
}

// Name returns the plugin identifier.
func (h *HTMLParser) Name() string { return "html" }

// Init is a no-op; HTMLParser requires no initialisation.
func (h *HTMLParser) Init(map[string]any) error { return nil }

// Close is a no-op; HTMLParser holds no resources.
func (h *HTMLParser) Close() error { return nil }

// CanParse reports whether the content type is HTML.
func (h *HTMLParser) CanParse(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "application/xhtml")
}

// Parse extracts data and links from the crawl response body.
func (h *HTMLParser) Parse(_ context.Context, resp *crawler.CrawlResponse) (*plugin.ParseResult, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}

	data := make(map[string]any)

	// Apply extraction rules.
	for name, sel := range h.Rules {
		var values []string
		doc.Find(sel).Each(func(_ int, s *goquery.Selection) {
			values = append(values, strings.TrimSpace(s.Text()))
		})
		if len(values) == 1 {
			data[name] = values[0]
		} else if len(values) > 1 {
			data[name] = values
		}
	}

	// Extract links from <a href="..."> elements.
	var links []string
	doc.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			href = strings.TrimSpace(href)
			if href != "" && href != "#" {
				links = append(links, href)
			}
		}
	})

	return &plugin.ParseResult{
		Data:  data,
		Links: links,
	}, nil
}

// Compile-time interface check.
var _ plugin.ParserPlugin = (*HTMLParser)(nil)
