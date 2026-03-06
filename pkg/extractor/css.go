package extractor

import (
	"bytes"
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

// extractCSS extracts data using a CSS selector via GoQuery.
// The attribute parameter controls what is extracted:
//   - "text": inner text of matched elements
//   - "html": inner HTML of matched elements
//   - anything else: the named attribute (e.g. "href", "src")
//
// If attribute is empty, defaults to "text".
func extractCSS(content []byte, selector, attribute string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("extractor: parse HTML for CSS: %w", err)
	}

	if attribute == "" {
		attribute = "text"
	}

	var values []string
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		switch attribute {
		case "text":
			values = append(values, s.Text())
		case "html":
			html, htmlErr := s.Html()
			if htmlErr == nil {
				values = append(values, html)
			}
		default:
			if val, exists := s.Attr(attribute); exists {
				values = append(values, val)
			}
		}
	})

	return values, nil
}
