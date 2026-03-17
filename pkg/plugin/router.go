package plugin

import (
	"context"
	"fmt"

	"github.com/kcenon/web_crawler/pkg/crawler"
)

// ParserRouter selects and invokes the appropriate ParserPlugin based
// on the content type of a crawl response. It iterates over registered
// parsers and uses the first one whose CanParse returns true.
type ParserRouter struct {
	registry *Registry
}

// NewParserRouter creates a router backed by the given registry.
func NewParserRouter(r *Registry) *ParserRouter {
	return &ParserRouter{registry: r}
}

// Parse finds a suitable parser for the response's content type and
// delegates to it. Returns ErrNoParser if no registered parser can
// handle the content type.
func (pr *ParserRouter) Parse(ctx context.Context, resp *crawler.CrawlResponse) (*ParseResult, error) {
	names := pr.registry.ListParsers()
	for _, name := range names {
		p, err := pr.registry.GetParser(name)
		if err != nil {
			continue
		}
		if p.CanParse(resp.ContentType) {
			return p.Parse(ctx, resp)
		}
	}
	return nil, fmt.Errorf("plugin: no parser registered for content type %q", resp.ContentType)
}
