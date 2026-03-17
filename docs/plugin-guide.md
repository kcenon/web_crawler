# Plugin Development Guide

This guide covers the web crawler's plugin system: architecture, built-in plugins,
and how to create custom plugins.

## Architecture Overview

The plugin system is organized into four layers:

```
┌─────────────────────────────────────────────────────────┐
│                    Configuration Layer                    │
│  YAML config → Loader → Factory → Init → Registry       │
├─────────────────────────────────────────────────────────┤
│                   Implementation Layer                    │
│  parser/html.go │ notifier/webhook.go │ storage backends │
├─────────────────────────────────────────────────────────┤
│                     Registry Layer                        │
│  Thread-safe Register / Get / List / CloseAll            │
├─────────────────────────────────────────────────────────┤
│                    Interface Layer                         │
│  Plugin │ StoragePlugin │ ParserPlugin │ NotifierPlugin   │
└─────────────────────────────────────────────────────────┘
```

### Plugin Categories

| Category | Interface | Purpose |
|----------|-----------|---------|
| **Storage** | `StoragePlugin` | Persist crawled items (file, CSV, PostgreSQL, ...) |
| **Parser** | `ParserPlugin` | Extract structured data from responses |
| **Notifier** | `NotifierPlugin` | Send event notifications (webhook, email, ...) |
| **Exporter** | `ExporterPlugin` | Export data to external systems (S3, BigQuery, ...) |

### Base Interface

Every plugin implements the base `Plugin` interface:

```go
type Plugin interface {
    Name() string
    Init(config map[string]any) error
    Close() error
}
```

- **Name()** — unique identifier used for registry lookups
- **Init()** — called once with configuration from YAML or programmatic setup
- **Close()** — release resources when the plugin is no longer needed

## Built-in Plugins

### Storage Plugins

| Name | Package | Description |
|------|---------|-------------|
| `file` | `pkg/storage` | JSON Lines file output |
| `csv` | `pkg/storage` | CSV file output with auto-detected columns |
| `postgres` | `pkg/storage` | PostgreSQL with batch CopyFrom |

### Parser Plugins

| Name | Package | Description |
|------|---------|-------------|
| `html` | `pkg/plugin/parser` | GoQuery CSS extraction + link discovery |

### Notifier Plugins

| Name | Package | Description |
|------|---------|-------------|
| `webhook` | `pkg/plugin/notifier` | JSON POST to HTTP endpoint |

## Creating a Custom Plugin

### Step 1: Choose the Interface

Pick the interface that matches your plugin's purpose:

```go
// Storage — persist items
type StoragePlugin interface {
    Plugin
    Store(ctx context.Context, items []storage.Item) error
}

// Parser — extract data from responses
type ParserPlugin interface {
    Plugin
    CanParse(contentType string) bool
    Parse(ctx context.Context, resp *crawler.CrawlResponse) (*ParseResult, error)
}

// Notifier — send event notifications
type NotifierPlugin interface {
    Plugin
    Notify(ctx context.Context, event *CrawlEvent) error
}

// Exporter — export to external systems
type ExporterPlugin interface {
    Plugin
    Export(ctx context.Context, items []storage.Item, format string) error
}
```

### Step 2: Implement the Interface

Example: a custom in-memory storage plugin.

```go
package main

import (
    "context"
    "sync"

    "github.com/kcenon/web_crawler/pkg/storage"
)

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
```

### Step 3: Register with the Registry

```go
registry := plugin.NewRegistry()
mem := &MemoryStorage{}
_ = mem.Init(nil)
registry.RegisterStorage("memory", mem)
```

### Step 4: (Optional) Add YAML Configuration Support

Register a factory so the plugin can be declared in YAML:

```go
loader := plugin.NewLoader(registry)
loader.RegisterStorageFactory("memory", func(config map[string]any) (plugin.StoragePlugin, error) {
    return &MemoryStorage{}, nil
})
```

Then in `plugins.yaml`:

```yaml
plugins:
  storage:
    - name: memory
```

## YAML Configuration

Plugins can be declared in a YAML config file:

```yaml
plugins:
  storage:
    - name: file
      config:
        path: output/results.jsonl

  parsers:
    - name: html

  notifiers:
    - name: webhook
      config:
        url: https://hooks.example.com/events
```

Load and apply the configuration:

```go
cfg, err := plugin.LoadConfigFile("configs/plugins.yaml")
if err != nil {
    log.Fatal(err)
}

registry := plugin.NewRegistry()
loader := plugin.NewLoader(registry)

// Register factories for built-in plugins
loader.RegisterStorageFactory("file", func(config map[string]any) (plugin.StoragePlugin, error) {
    return storage.NewFileStorage(storage.FileConfig{}), nil
})

if err := loader.Load(cfg); err != nil {
    log.Fatal(err)
}
```

## Content-Type Routing (Parsers)

The `ParserRouter` automatically selects the right parser based on content type:

```go
router := plugin.NewParserRouter(registry)

// Parses using the first registered parser whose CanParse returns true
result, err := router.Parse(ctx, crawlResponse)
```

## Event Types (Notifiers)

| Event Type | Constant | When |
|------------|----------|------|
| Started | `plugin.EventStarted` | Crawl begins |
| Completed | `plugin.EventCompleted` | Crawl finishes |
| Error | `plugin.EventError` | Unrecoverable error |
| Threshold | `plugin.EventThreshold` | Metric threshold crossed |

## API Reference

### Package `plugin`

| Type | Description |
|------|-------------|
| `Plugin` | Base interface: Name, Init, Close |
| `StoragePlugin` | Plugin + Store |
| `ParserPlugin` | Plugin + CanParse, Parse |
| `NotifierPlugin` | Plugin + Notify |
| `ExporterPlugin` | Plugin + Export |
| `Registry` | Thread-safe plugin container |
| `ParserRouter` | Content-type based parser selection |
| `Config` | YAML configuration schema |
| `Loader` | Factory-based plugin instantiation |
| `ParseResult` | Parser output (Data + Links) |
| `CrawlEvent` | Notifier event payload |
| `Entry` | Single plugin declaration in YAML |

### Package `plugin/parser`

| Type | Description |
|------|-------------|
| `HTMLParser` | GoQuery-based HTML parser with CSS rules |

### Package `plugin/notifier`

| Type | Description |
|------|-------------|
| `Webhook` | HTTP POST notifier with custom headers |
| `WebhookConfig` | URL, Timeout, Headers configuration |

### Error Types

| Type | When |
|------|------|
| `ErrDuplicatePlugin` | Name already registered in category |
| `ErrPluginNotFound` | Name not found in registry |
| `ErrNilPlugin` | Nil passed to Register |
