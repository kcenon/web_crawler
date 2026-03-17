package plugin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kcenon/web_crawler/pkg/crawler"
	"github.com/kcenon/web_crawler/pkg/storage"
)

const testYAML = `
plugins:
  storage:
    - name: file
      config:
        path: /tmp/test.jsonl
  parsers:
    - name: html
  notifiers:
    - name: webhook
      config:
        url: http://localhost:9999/hook
`

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig([]byte(testYAML))
	if err != nil {
		t.Fatal(err)
	}

	if len(cfg.Plugins.Storage) != 1 {
		t.Fatalf("storage entries = %d, want 1", len(cfg.Plugins.Storage))
	}
	if cfg.Plugins.Storage[0].Name != "file" {
		t.Errorf("storage[0].Name = %q, want %q", cfg.Plugins.Storage[0].Name, "file")
	}
	if cfg.Plugins.Storage[0].Config["path"] != "/tmp/test.jsonl" {
		t.Errorf("storage[0].Config[path] = %v", cfg.Plugins.Storage[0].Config["path"])
	}

	if len(cfg.Plugins.Parsers) != 1 {
		t.Fatalf("parser entries = %d, want 1", len(cfg.Plugins.Parsers))
	}
	if cfg.Plugins.Parsers[0].Name != "html" {
		t.Errorf("parsers[0].Name = %q", cfg.Plugins.Parsers[0].Name)
	}

	if len(cfg.Plugins.Notifiers) != 1 {
		t.Fatalf("notifier entries = %d, want 1", len(cfg.Plugins.Notifiers))
	}
	if cfg.Plugins.Notifiers[0].Config["url"] != "http://localhost:9999/hook" {
		t.Errorf("notifiers[0].Config[url] = %v", cfg.Plugins.Notifiers[0].Config["url"])
	}
}

func TestLoadConfig_Empty(t *testing.T) {
	cfg, err := LoadConfig([]byte(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Plugins.Storage) != 0 {
		t.Error("expected empty storage list")
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	_, err := LoadConfig([]byte("{{invalid yaml"))
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugins.yaml")
	if err := os.WriteFile(path, []byte(testYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Plugins.Storage) != 1 {
		t.Errorf("storage entries = %d, want 1", len(cfg.Plugins.Storage))
	}
}

func TestLoadConfigFile_NotFound(t *testing.T) {
	_, err := LoadConfigFile("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- Loader tests ----------------------------------------------------

// fakeStorage is a test storage plugin.
type fakeStorage struct{ path string }

func (s *fakeStorage) Name() string                                    { return "file" }
func (s *fakeStorage) Init(cfg map[string]any) error                   { return nil }
func (s *fakeStorage) Close() error                                    { return nil }
func (s *fakeStorage) Store(_ context.Context, _ []storage.Item) error { return nil }

// fakeParser is a test parser plugin.
type fakeParser struct{}

func (p *fakeParser) Name() string              { return "html" }
func (p *fakeParser) Init(map[string]any) error { return nil }
func (p *fakeParser) Close() error              { return nil }
func (p *fakeParser) CanParse(string) bool      { return true }
func (p *fakeParser) Parse(_ context.Context, _ *crawler.CrawlResponse) (*ParseResult, error) {
	return &ParseResult{}, nil
}

// fakeNotifier is a test notifier plugin.
type fakeNotifier struct{}

func (n *fakeNotifier) Name() string                                  { return "webhook" }
func (n *fakeNotifier) Init(map[string]any) error                     { return nil }
func (n *fakeNotifier) Close() error                                  { return nil }
func (n *fakeNotifier) Notify(_ context.Context, _ *CrawlEvent) error { return nil }

func TestLoader_Load(t *testing.T) {
	cfg, err := LoadConfig([]byte(testYAML))
	if err != nil {
		t.Fatal(err)
	}

	r := NewRegistry()
	loader := NewLoader(r)

	loader.RegisterStorageFactory("file", func(config map[string]any) (StoragePlugin, error) {
		return &fakeStorage{}, nil
	})
	loader.RegisterParserFactory("html", func(config map[string]any) (ParserPlugin, error) {
		return &fakeParser{}, nil
	})
	loader.RegisterNotifierFactory("webhook", func(config map[string]any) (NotifierPlugin, error) {
		return &fakeNotifier{}, nil
	})

	if err := loader.Load(cfg); err != nil {
		t.Fatal(err)
	}

	// Verify plugins are registered.
	if len(r.ListStorage()) != 1 {
		t.Errorf("storage plugins = %d, want 1", len(r.ListStorage()))
	}
	if len(r.ListParsers()) != 1 {
		t.Errorf("parser plugins = %d, want 1", len(r.ListParsers()))
	}
	if len(r.ListNotifiers()) != 1 {
		t.Errorf("notifier plugins = %d, want 1", len(r.ListNotifiers()))
	}
}

func TestLoader_UnknownPlugin(t *testing.T) {
	cfg, _ := LoadConfig([]byte(`
plugins:
  storage:
    - name: unknown_backend
`))

	r := NewRegistry()
	loader := NewLoader(r)

	err := loader.Load(cfg)
	if err == nil {
		t.Error("expected error for unknown plugin")
	}
}

func TestLoader_EmptyConfig(t *testing.T) {
	cfg, _ := LoadConfig([]byte(""))

	r := NewRegistry()
	loader := NewLoader(r)

	if err := loader.Load(cfg); err != nil {
		t.Errorf("Load empty config should succeed, got: %v", err)
	}
}
