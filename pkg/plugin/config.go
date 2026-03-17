package plugin

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level YAML configuration for plugins.
//
// Example YAML:
//
//	plugins:
//	  storage:
//	    - name: file
//	      config:
//	        path: output/results.jsonl
//	  parsers:
//	    - name: html
//	  notifiers:
//	    - name: webhook
//	      config:
//	        url: https://example.com/hook
type Config struct {
	Plugins PluginsConfig `yaml:"plugins"`
}

// PluginsConfig groups plugin declarations by category.
type PluginsConfig struct {
	Storage   []PluginEntry `yaml:"storage,omitempty"`
	Parsers   []PluginEntry `yaml:"parsers,omitempty"`
	Notifiers []PluginEntry `yaml:"notifiers,omitempty"`
	Exporters []PluginEntry `yaml:"exporters,omitempty"`
}

// PluginEntry declares a single plugin instance.
type PluginEntry struct {
	// Name identifies which plugin to instantiate (must match a registered factory).
	Name string `yaml:"name"`

	// Config is passed to Plugin.Init as map[string]any.
	Config map[string]any `yaml:"config,omitempty"`
}

// LoadConfigFile reads a YAML file and returns the parsed Config.
func LoadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path) //nolint:gosec // user-specified config file
	if err != nil {
		return nil, fmt.Errorf("plugin config: read %s: %w", path, err)
	}
	return LoadConfig(data)
}

// LoadConfig parses YAML bytes into a Config.
func LoadConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("plugin config: parse: %w", err)
	}
	return &cfg, nil
}

// StorageFactory creates a StoragePlugin from a config map.
type StorageFactory func(config map[string]any) (StoragePlugin, error)

// ParserFactory creates a ParserPlugin from a config map.
type ParserFactory func(config map[string]any) (ParserPlugin, error)

// NotifierFactory creates a NotifierPlugin from a config map.
type NotifierFactory func(config map[string]any) (NotifierPlugin, error)

// ExporterFactory creates an ExporterPlugin from a config map.
type ExporterFactory func(config map[string]any) (ExporterPlugin, error)

// Loader instantiates and registers plugins from a Config using
// registered factory functions.
type Loader struct {
	registry  *Registry
	storage   map[string]StorageFactory
	parsers   map[string]ParserFactory
	notifiers map[string]NotifierFactory
	exporters map[string]ExporterFactory
}

// NewLoader creates a Loader bound to the given registry.
func NewLoader(r *Registry) *Loader {
	return &Loader{
		registry:  r,
		storage:   make(map[string]StorageFactory),
		parsers:   make(map[string]ParserFactory),
		notifiers: make(map[string]NotifierFactory),
		exporters: make(map[string]ExporterFactory),
	}
}

// RegisterStorageFactory associates a name with a storage factory.
func (l *Loader) RegisterStorageFactory(name string, f StorageFactory) {
	l.storage[name] = f
}

// RegisterParserFactory associates a name with a parser factory.
func (l *Loader) RegisterParserFactory(name string, f ParserFactory) {
	l.parsers[name] = f
}

// RegisterNotifierFactory associates a name with a notifier factory.
func (l *Loader) RegisterNotifierFactory(name string, f NotifierFactory) {
	l.notifiers[name] = f
}

// RegisterExporterFactory associates a name with an exporter factory.
func (l *Loader) RegisterExporterFactory(name string, f ExporterFactory) {
	l.exporters[name] = f
}

// Load instantiates all plugins declared in cfg, calls Init on each,
// and registers them in the loader's registry.
func (l *Loader) Load(cfg *Config) error {
	for _, entry := range cfg.Plugins.Storage {
		f, ok := l.storage[entry.Name]
		if !ok {
			return fmt.Errorf("plugin config: unknown storage plugin %q", entry.Name)
		}
		p, err := f(entry.Config)
		if err != nil {
			return fmt.Errorf("plugin config: create storage %q: %w", entry.Name, err)
		}
		if err := p.Init(entry.Config); err != nil {
			return fmt.Errorf("plugin config: init storage %q: %w", entry.Name, err)
		}
		if err := l.registry.RegisterStorage(entry.Name, p); err != nil {
			return fmt.Errorf("plugin config: register storage %q: %w", entry.Name, err)
		}
	}

	for _, entry := range cfg.Plugins.Parsers {
		f, ok := l.parsers[entry.Name]
		if !ok {
			return fmt.Errorf("plugin config: unknown parser plugin %q", entry.Name)
		}
		p, err := f(entry.Config)
		if err != nil {
			return fmt.Errorf("plugin config: create parser %q: %w", entry.Name, err)
		}
		if err := p.Init(entry.Config); err != nil {
			return fmt.Errorf("plugin config: init parser %q: %w", entry.Name, err)
		}
		if err := l.registry.RegisterParser(entry.Name, p); err != nil {
			return fmt.Errorf("plugin config: register parser %q: %w", entry.Name, err)
		}
	}

	for _, entry := range cfg.Plugins.Notifiers {
		f, ok := l.notifiers[entry.Name]
		if !ok {
			return fmt.Errorf("plugin config: unknown notifier plugin %q", entry.Name)
		}
		p, err := f(entry.Config)
		if err != nil {
			return fmt.Errorf("plugin config: create notifier %q: %w", entry.Name, err)
		}
		if err := p.Init(entry.Config); err != nil {
			return fmt.Errorf("plugin config: init notifier %q: %w", entry.Name, err)
		}
		if err := l.registry.RegisterNotifier(entry.Name, p); err != nil {
			return fmt.Errorf("plugin config: register notifier %q: %w", entry.Name, err)
		}
	}

	for _, entry := range cfg.Plugins.Exporters {
		f, ok := l.exporters[entry.Name]
		if !ok {
			return fmt.Errorf("plugin config: unknown exporter plugin %q", entry.Name)
		}
		p, err := f(entry.Config)
		if err != nil {
			return fmt.Errorf("plugin config: create exporter %q: %w", entry.Name, err)
		}
		if err := p.Init(entry.Config); err != nil {
			return fmt.Errorf("plugin config: init exporter %q: %w", entry.Name, err)
		}
		if err := l.registry.RegisterExporter(entry.Name, p); err != nil {
			return fmt.Errorf("plugin config: register exporter %q: %w", entry.Name, err)
		}
	}

	return nil
}
