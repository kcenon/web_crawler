package plugin

import "sync"

// Registry is a thread-safe container for plugin instances.
// It organises plugins by category (storage, parser, notifier, exporter)
// and provides registration, retrieval, and listing operations.
type Registry struct {
	storage   map[string]StoragePlugin
	parsers   map[string]ParserPlugin
	notifiers map[string]NotifierPlugin
	exporters map[string]ExporterPlugin
	mu        sync.RWMutex
}

// NewRegistry creates an empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		storage:   make(map[string]StoragePlugin),
		parsers:   make(map[string]ParserPlugin),
		notifiers: make(map[string]NotifierPlugin),
		exporters: make(map[string]ExporterPlugin),
	}
}

// --- Storage ----------------------------------------------------------

// RegisterStorage registers a storage plugin under the given name.
func (r *Registry) RegisterStorage(name string, p StoragePlugin) error {
	if p == nil {
		return &ErrNilPlugin{Category: "storage"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.storage[name]; exists {
		return &ErrDuplicatePlugin{Category: "storage", Name: name}
	}
	r.storage[name] = p
	return nil
}

// GetStorage returns the storage plugin registered under name.
func (r *Registry) GetStorage(name string) (StoragePlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.storage[name]
	if !ok {
		return nil, &ErrPluginNotFound{Category: "storage", Name: name}
	}
	return p, nil
}

// ListStorage returns the names of all registered storage plugins.
func (r *Registry) ListStorage() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapKeys(r.storage)
}

// --- Parser -----------------------------------------------------------

// RegisterParser registers a parser plugin under the given name.
func (r *Registry) RegisterParser(name string, p ParserPlugin) error {
	if p == nil {
		return &ErrNilPlugin{Category: "parser"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.parsers[name]; exists {
		return &ErrDuplicatePlugin{Category: "parser", Name: name}
	}
	r.parsers[name] = p
	return nil
}

// GetParser returns the parser plugin registered under name.
func (r *Registry) GetParser(name string) (ParserPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.parsers[name]
	if !ok {
		return nil, &ErrPluginNotFound{Category: "parser", Name: name}
	}
	return p, nil
}

// ListParsers returns the names of all registered parser plugins.
func (r *Registry) ListParsers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapKeys(r.parsers)
}

// --- Notifier ---------------------------------------------------------

// RegisterNotifier registers a notifier plugin under the given name.
func (r *Registry) RegisterNotifier(name string, p NotifierPlugin) error {
	if p == nil {
		return &ErrNilPlugin{Category: "notifier"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.notifiers[name]; exists {
		return &ErrDuplicatePlugin{Category: "notifier", Name: name}
	}
	r.notifiers[name] = p
	return nil
}

// GetNotifier returns the notifier plugin registered under name.
func (r *Registry) GetNotifier(name string) (NotifierPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.notifiers[name]
	if !ok {
		return nil, &ErrPluginNotFound{Category: "notifier", Name: name}
	}
	return p, nil
}

// ListNotifiers returns the names of all registered notifier plugins.
func (r *Registry) ListNotifiers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapKeys(r.notifiers)
}

// --- Exporter ---------------------------------------------------------

// RegisterExporter registers an exporter plugin under the given name.
func (r *Registry) RegisterExporter(name string, p ExporterPlugin) error {
	if p == nil {
		return &ErrNilPlugin{Category: "exporter"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.exporters[name]; exists {
		return &ErrDuplicatePlugin{Category: "exporter", Name: name}
	}
	r.exporters[name] = p
	return nil
}

// GetExporter returns the exporter plugin registered under name.
func (r *Registry) GetExporter(name string) (ExporterPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.exporters[name]
	if !ok {
		return nil, &ErrPluginNotFound{Category: "exporter", Name: name}
	}
	return p, nil
}

// ListExporters returns the names of all registered exporter plugins.
func (r *Registry) ListExporters() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return mapKeys(r.exporters)
}

// --- CloseAll ---------------------------------------------------------

// CloseAll calls Close on every registered plugin, collecting any errors.
// It is safe to call CloseAll concurrently but the caller should ensure
// no new registrations happen while CloseAll is running.
func (r *Registry) CloseAll() []error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errs []error
	for _, p := range r.storage {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, p := range r.parsers {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, p := range r.notifiers {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	for _, p := range r.exporters {
		if err := p.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// --- helpers ----------------------------------------------------------

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
