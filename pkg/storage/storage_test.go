package storage

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var testTime = time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)

func testItems() []Item {
	return []Item{
		{
			URL:       "https://example.com/page1",
			Data:      map[string]any{"title": "Page 1", "score": 42},
			Metadata:  map[string]any{"depth": 1},
			CrawledAt: testTime,
		},
		{
			URL:       "https://example.com/page2",
			Data:      map[string]any{"title": "Page 2"},
			Metadata:  map[string]any{"depth": 2},
			CrawledAt: testTime.Add(time.Minute),
		},
	}
}

// --- FileStorage (JSON Lines) tests ---

func TestFileStorage_JSONLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")

	fs := NewFileStorage(FileConfig{Path: path})
	if err := fs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	items := testItems()
	if err := fs.Store(context.Background(), items); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON lines, got %d", len(lines))
	}

	for i, line := range lines {
		var item Item
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
		}
		if item.URL != items[i].URL {
			t.Errorf("line %d: expected URL %q, got %q", i, items[i].URL, item.URL)
		}
	}
}

func TestFileStorage_Pretty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	fs := NewFileStorage(FileConfig{Path: path, Pretty: true})
	if err := fs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := fs.Store(context.Background(), testItems()[:1]); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	// Pretty output should contain indentation
	if !strings.Contains(string(data), "  ") {
		t.Error("expected indented JSON output")
	}
}

func TestFileStorage_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.jsonl")

	// Write first batch
	fs := NewFileStorage(FileConfig{Path: path})
	if err := fs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := fs.Store(context.Background(), testItems()[:1]); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Write second batch (should append)
	fs2 := NewFileStorage(FileConfig{Path: path})
	if err := fs2.Init(nil); err != nil {
		t.Fatalf("init2: %v", err)
	}
	if err := fs2.Store(context.Background(), testItems()[1:]); err != nil {
		t.Fatalf("store2: %v", err)
	}
	if err := fs2.Close(); err != nil {
		t.Fatalf("close2: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines after append, got %d", len(lines))
	}
}

func TestFileStorage_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.jsonl")

	fs := NewFileStorage(FileConfig{Path: path})
	if err := fs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = fs.Store(context.Background(), testItems()[:1])
		}()
	}
	wg.Wait()

	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 10 {
		t.Errorf("expected 10 lines from concurrent writes, got %d", len(lines))
	}
}

func TestFileStorage_InitPathFromConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "from_config.jsonl")

	fs := NewFileStorage(FileConfig{})
	if err := fs.Init(map[string]any{"path": path}); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := fs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to be created: %v", err)
	}
}

func TestFileStorage_InitNoPath(t *testing.T) {
	fs := NewFileStorage(FileConfig{})
	if err := fs.Init(nil); err == nil {
		t.Error("expected error for missing path")
	}
}

func TestFileStorage_StoreNotInitialized(t *testing.T) {
	fs := NewFileStorage(FileConfig{Path: "unused"})
	err := fs.Store(context.Background(), testItems())
	if err == nil {
		t.Error("expected error for uninitialized storage")
	}
}

// --- CSVStorage tests ---

func TestCSVStorage_BasicOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.csv")

	cs := NewCSVStorage(CSVConfig{
		Path:    path,
		Columns: []string{"url", "data.title", "crawled_at"},
	})

	if err := cs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := cs.Store(context.Background(), testItems()); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := cs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// 1 header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}

	// Check header
	if records[0][0] != "url" || records[0][1] != "data.title" {
		t.Errorf("unexpected header: %v", records[0])
	}

	// Check first data row
	if records[1][0] != "https://example.com/page1" {
		t.Errorf("expected URL in first row, got %q", records[1][0])
	}
	if records[1][1] != "Page 1" {
		t.Errorf("expected title 'Page 1', got %q", records[1][1])
	}
}

func TestCSVStorage_AutoDetectColumns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auto.csv")

	cs := NewCSVStorage(CSVConfig{Path: path})
	if err := cs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := cs.Store(context.Background(), testItems()[:1]); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := cs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	if len(records) < 2 {
		t.Fatalf("expected at least header + 1 row, got %d", len(records))
	}

	// Columns should be sorted and include flattened keys
	header := records[0]
	foundURL := false
	for _, col := range header {
		if col == "url" {
			foundURL = true
		}
	}
	if !foundURL {
		t.Error("expected 'url' in auto-detected columns")
	}
}

func TestCSVStorage_CustomDelimiter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tabs.csv")

	cs := NewCSVStorage(CSVConfig{
		Path:      path,
		Delimiter: '\t',
		Columns:   []string{"url", "data.title"},
	})

	if err := cs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	if err := cs.Store(context.Background(), testItems()[:1]); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := cs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if !strings.Contains(string(data), "\t") {
		t.Error("expected tab-delimited output")
	}
}

func TestCSVStorage_NestedData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested.csv")

	items := []Item{
		{
			URL: "https://example.com",
			Data: map[string]any{
				"meta": map[string]any{
					"author": "John",
					"tags":   "go,web",
				},
			},
			CrawledAt: testTime,
		},
	}

	cs := NewCSVStorage(CSVConfig{
		Path:    path,
		Columns: []string{"url", "data.meta.author", "data.meta.tags"},
	})

	if err := cs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := cs.Store(context.Background(), items); err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := cs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	if records[1][1] != "John" {
		t.Errorf("expected author 'John', got %q", records[1][1])
	}
}

func TestCSVStorage_InitNoPath(t *testing.T) {
	cs := NewCSVStorage(CSVConfig{})
	if err := cs.Init(nil); err == nil {
		t.Error("expected error for missing path")
	}
}

func TestCSVStorage_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.csv")

	cs := NewCSVStorage(CSVConfig{
		Path:    path,
		Columns: []string{"url"},
	})

	if err := cs.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cs.Store(context.Background(), testItems()[:1])
		}()
	}
	wg.Wait()

	if err := cs.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read csv: %v", err)
	}

	// 1 header + 10 data rows
	if len(records) != 11 {
		t.Errorf("expected 11 rows (1 header + 10 data), got %d", len(records))
	}
}

// --- flattenItem tests ---

func TestFlattenItem(t *testing.T) {
	item := Item{
		URL: "https://example.com",
		Data: map[string]any{
			"title": "Hello",
			"nested": map[string]any{
				"key": "value",
			},
		},
		CrawledAt: testTime,
	}

	flat := flattenItem(item)

	if flat["url"] != "https://example.com" {
		t.Errorf("expected url, got %q", flat["url"])
	}
	if flat["data.title"] != "Hello" {
		t.Errorf("expected data.title='Hello', got %q", flat["data.title"])
	}
	if flat["data.nested.key"] != "value" {
		t.Errorf("expected data.nested.key='value', got %q", flat["data.nested.key"])
	}
}
