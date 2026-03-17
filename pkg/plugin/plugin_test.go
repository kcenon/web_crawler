package plugin

import (
	"context"
	"testing"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/storage"
)

// --- test doubles -----------------------------------------------------

// stubStorage is a minimal StoragePlugin for testing.
type stubStorage struct {
	name   string
	items  []storage.Item
	closed bool
}

func (s *stubStorage) Name() string                                    { return s.name }
func (s *stubStorage) Init(map[string]any) error                       { return nil }
func (s *stubStorage) Close() error                                    { s.closed = true; return nil }
func (s *stubStorage) Store(_ context.Context, items []storage.Item) error {
	s.items = append(s.items, items...)
	return nil
}

// stubParser is a minimal ParserPlugin for testing.
type stubParser struct {
	name        string
	contentType string
}

func (p *stubParser) Name() string              { return p.name }
func (p *stubParser) Init(map[string]any) error { return nil }
func (p *stubParser) Close() error              { return nil }
func (p *stubParser) CanParse(ct string) bool   { return ct == p.contentType }
func (p *stubParser) Parse(_ context.Context, _ *crawler.CrawlResponse) (*ParseResult, error) {
	return &ParseResult{Data: map[string]any{"parsed": true}}, nil
}

// stubNotifier is a minimal NotifierPlugin for testing.
type stubNotifier struct {
	name   string
	events []*CrawlEvent
}

func (n *stubNotifier) Name() string              { return n.name }
func (n *stubNotifier) Init(map[string]any) error { return nil }
func (n *stubNotifier) Close() error              { return nil }
func (n *stubNotifier) Notify(_ context.Context, ev *CrawlEvent) error {
	n.events = append(n.events, ev)
	return nil
}

// stubExporter is a minimal ExporterPlugin for testing.
type stubExporter struct {
	name string
}

func (e *stubExporter) Name() string              { return e.name }
func (e *stubExporter) Init(map[string]any) error { return nil }
func (e *stubExporter) Close() error              { return nil }
func (e *stubExporter) Export(_ context.Context, _ []storage.Item, _ string) error {
	return nil
}

// --- interface conformance -------------------------------------------

func TestInterfaceConformance(t *testing.T) {
	// Verify that test doubles satisfy their respective interfaces.
	var _ StoragePlugin = (*stubStorage)(nil)
	var _ ParserPlugin = (*stubParser)(nil)
	var _ NotifierPlugin = (*stubNotifier)(nil)
	var _ ExporterPlugin = (*stubExporter)(nil)
}

// --- StoragePlugin behaviour -----------------------------------------

func TestStoragePlugin_Store(t *testing.T) {
	s := &stubStorage{name: "mem"}
	items := []storage.Item{
		{URL: "https://example.com", Data: map[string]any{"title": "Test"}},
	}

	if err := s.Store(context.Background(), items); err != nil {
		t.Fatal(err)
	}
	if len(s.items) != 1 {
		t.Errorf("stored %d items, want 1", len(s.items))
	}
}

// --- ParserPlugin behaviour ------------------------------------------

func TestParserPlugin_CanParse(t *testing.T) {
	p := &stubParser{name: "html", contentType: "text/html"}

	if !p.CanParse("text/html") {
		t.Error("expected CanParse(text/html) = true")
	}
	if p.CanParse("application/json") {
		t.Error("expected CanParse(application/json) = false")
	}
}

// --- NotifierPlugin behaviour ----------------------------------------

func TestNotifierPlugin_Notify(t *testing.T) {
	n := &stubNotifier{name: "log"}
	ev := &CrawlEvent{Type: EventCompleted, Message: "done"}

	if err := n.Notify(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	if len(n.events) != 1 {
		t.Errorf("received %d events, want 1", len(n.events))
	}
	if n.events[0].Type != EventCompleted {
		t.Errorf("event type = %q, want %q", n.events[0].Type, EventCompleted)
	}
}
