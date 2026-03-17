package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// FileConfig configures the JSON Lines file storage.
type FileConfig struct {
	// Path is the output file path. Required.
	Path string

	// Pretty enables indented JSON output instead of JSON Lines.
	Pretty bool
}

// FileStorage writes crawled items to a file in JSON Lines format.
// It is safe for concurrent use.
type FileStorage struct {
	cfg  FileConfig
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

// NewFileStorage creates a new FileStorage instance.
func NewFileStorage(cfg FileConfig) *FileStorage {
	return &FileStorage{cfg: cfg}
}

// Name returns the plugin identifier for this storage backend.
func (f *FileStorage) Name() string { return "file" }

// Init opens the output file for append-mode writing.
func (f *FileStorage) Init(config map[string]any) error {
	if p, ok := config["path"].(string); ok && p != "" {
		f.cfg.Path = p
	}
	if f.cfg.Path == "" {
		return fmt.Errorf("file storage: path is required")
	}

	file, err := os.OpenFile(f.cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644) //nolint:gosec // user-specified output file
	if err != nil {
		return fmt.Errorf("file storage: open %s: %w", f.cfg.Path, err)
	}

	f.file = file
	f.enc = json.NewEncoder(file)
	if f.cfg.Pretty {
		f.enc.SetIndent("", "  ")
	}

	return nil
}

// Store writes items to the file in JSON Lines format (one JSON object per line).
func (f *FileStorage) Store(_ context.Context, items []Item) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		return fmt.Errorf("file storage: not initialized")
	}

	for _, item := range items {
		if err := f.enc.Encode(item); err != nil {
			return fmt.Errorf("file storage: encode item: %w", err)
		}
	}

	return nil
}

// Close flushes and closes the file.
func (f *FileStorage) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		return nil
	}

	err := f.file.Close()
	f.file = nil
	f.enc = nil
	return err
}
