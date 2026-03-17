package storage

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"
)

// CSVConfig configures the CSV file storage.
type CSVConfig struct {
	// Path is the output file path. Required.
	Path string

	// Delimiter is the field separator. Default: comma.
	Delimiter rune

	// Columns defines the CSV header columns. If empty, columns are
	// auto-detected from the first batch of items.
	Columns []string
}

// CSVStorage writes crawled items to a CSV file.
// Nested data maps are flattened with dot notation (e.g., "data.title").
type CSVStorage struct {
	cfg     CSVConfig
	mu      sync.Mutex
	file    *os.File
	writer  *csv.Writer
	columns []string
	wrote   bool
}

// NewCSVStorage creates a new CSVStorage instance.
func NewCSVStorage(cfg CSVConfig) *CSVStorage {
	return &CSVStorage{cfg: cfg}
}

// Name returns the plugin identifier for this storage backend.
func (c *CSVStorage) Name() string { return "csv" }

// Init opens the output file and prepares the CSV writer.
func (c *CSVStorage) Init(config map[string]any) error {
	if p, ok := config["path"].(string); ok && p != "" {
		c.cfg.Path = p
	}
	if c.cfg.Path == "" {
		return fmt.Errorf("csv storage: path is required")
	}

	file, err := os.OpenFile(c.cfg.Path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644) //nolint:gosec // user-specified output file
	if err != nil {
		return fmt.Errorf("csv storage: open %s: %w", c.cfg.Path, err)
	}

	c.file = file
	c.writer = csv.NewWriter(file)
	if c.cfg.Delimiter != 0 {
		c.writer.Comma = c.cfg.Delimiter
	}
	c.columns = c.cfg.Columns

	return nil
}

// Store writes items as CSV rows. On the first call, if no columns are
// configured, they are auto-detected from the items.
func (c *CSVStorage) Store(_ context.Context, items []Item) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.file == nil {
		return fmt.Errorf("csv storage: not initialized")
	}

	if len(items) == 0 {
		return nil
	}

	// Auto-detect columns from first batch if not configured.
	if len(c.columns) == 0 {
		c.columns = detectColumns(items[0])
	}

	// Write header on first Store call.
	if !c.wrote {
		if err := c.writer.Write(c.columns); err != nil {
			return fmt.Errorf("csv storage: write header: %w", err)
		}
		c.wrote = true
	}

	for _, item := range items {
		flat := flattenItem(item)
		row := make([]string, len(c.columns))
		for i, col := range c.columns {
			row[i] = flat[col]
		}
		if err := c.writer.Write(row); err != nil {
			return fmt.Errorf("csv storage: write row: %w", err)
		}
	}

	c.writer.Flush()
	return c.writer.Error()
}

// Close flushes and closes the CSV file.
func (c *CSVStorage) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.file == nil {
		return nil
	}

	c.writer.Flush()
	err := c.file.Close()
	c.file = nil
	c.writer = nil
	return err
}

// detectColumns returns sorted column names from an item.
func detectColumns(item Item) []string {
	flat := flattenItem(item)
	cols := make([]string, 0, len(flat))
	for k := range flat {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

// flattenItem converts an Item into a flat key-value map suitable for CSV.
func flattenItem(item Item) map[string]string {
	flat := make(map[string]string)

	flat["url"] = item.URL
	flat["crawled_at"] = item.CrawledAt.Format(time.RFC3339)

	flattenMap(flat, "data", item.Data)
	flattenMap(flat, "metadata", item.Metadata)

	return flat
}

// flattenMap recursively flattens a nested map with dot-separated keys.
func flattenMap(out map[string]string, prefix string, m map[string]any) {
	for k, v := range m {
		key := prefix + "." + k
		switch val := v.(type) {
		case map[string]any:
			flattenMap(out, key, val)
		default:
			out[key] = fmt.Sprintf("%v", val)
		}
	}
}
