package plugin

import "github.com/kcenon/web_crawler/pkg/storage"

// Compile-time interface verification: every built-in storage backend
// must satisfy both the legacy storage.Plugin and the new plugin.StoragePlugin.
var (
	_ StoragePlugin = (*storage.FileStorage)(nil)
	_ StoragePlugin = (*storage.CSVStorage)(nil)
	_ StoragePlugin = (*storage.PostgresPlugin)(nil)
)

// RegisterBuiltinStorage registers the provided storage backends in the
// registry under their Name(). It is a convenience for wiring up all
// built-in storage implementations at application startup.
func RegisterBuiltinStorage(r *Registry, plugins ...StoragePlugin) error {
	for _, p := range plugins {
		if err := r.RegisterStorage(p.Name(), p); err != nil {
			return err
		}
	}
	return nil
}
