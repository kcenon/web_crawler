package extractor

import (
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
)

// extractXPath extracts text content from nodes matching the XPath expression.
func extractXPath(content []byte, expr string) ([]string, error) {
	doc, err := htmlquery.Parse(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("extractor: parse HTML for XPath: %w", err)
	}

	nodes, err := htmlquery.QueryAll(doc, expr)
	if err != nil {
		return nil, fmt.Errorf("extractor: XPath query %q: %w", expr, err)
	}

	values := make([]string, 0, len(nodes))
	for _, n := range nodes {
		values = append(values, htmlquery.InnerText(n))
	}

	return values, nil
}
