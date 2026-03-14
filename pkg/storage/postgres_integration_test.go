//go:build integration

package storage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// postgresTestDSN returns the DSN for integration tests.
// Set POSTGRES_TEST_DSN in the environment, e.g.:
//
//	POSTGRES_TEST_DSN="postgres://postgres:password@localhost:5432/test_db?sslmode=disable"
func postgresTestDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_TEST_DSN not set; skipping integration test")
	}
	return dsn
}

func TestPostgresPlugin_InitAndMigrate(t *testing.T) {
	dsn := postgresTestDSN(t)

	p := NewPostgresPlugin(PostgresConfig{DSN: dsn})
	if err := p.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer p.Close()

	// Running Init again should be idempotent (migrations already applied).
	p2 := NewPostgresPlugin(PostgresConfig{DSN: dsn})
	if err := p2.Init(nil); err != nil {
		t.Fatalf("second init (idempotent): %v", err)
	}
	defer p2.Close()
}

func TestPostgresPlugin_Store100Items(t *testing.T) {
	dsn := postgresTestDSN(t)

	p := NewPostgresPlugin(PostgresConfig{DSN: dsn, BatchSize: 10})
	if err := p.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer p.Close()

	ctx := context.Background()

	// Clean up before test.
	if _, err := p.pool.Exec(ctx, "DELETE FROM crawled_items WHERE url LIKE 'https://test-item-%'"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	items := make([]Item, 100)
	ts := time.Now().UTC().Truncate(time.Microsecond)
	for i := range items {
		items[i] = Item{
			URL:       fmt.Sprintf("https://test-item-%d.example.com", i),
			Data:      map[string]any{"index": i},
			Metadata:  map[string]any{"depth": 1},
			CrawledAt: ts,
		}
	}

	// Store via CopyFrom (len >= BatchSize).
	if err := p.Store(ctx, items); err != nil {
		t.Fatalf("store: %v", err)
	}

	var count int
	row := p.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM crawled_items WHERE url LIKE 'https://test-item-%'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 100 {
		t.Errorf("expected 100 rows, got %d", count)
	}

	// Clean up.
	if _, err := p.pool.Exec(ctx, "DELETE FROM crawled_items WHERE url LIKE 'https://test-item-%'"); err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func TestPostgresPlugin_SmallBatchInsert(t *testing.T) {
	dsn := postgresTestDSN(t)

	p := NewPostgresPlugin(PostgresConfig{DSN: dsn, BatchSize: 10})
	if err := p.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer p.Close()

	ctx := context.Background()

	const marker = "https://small-batch-test.example.com"
	if _, err := p.pool.Exec(ctx, "DELETE FROM crawled_items WHERE url = $1", marker); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	// Store 3 items (< BatchSize=10) → uses insertBatch.
	items := []Item{
		{URL: marker, Data: map[string]any{"k": "v"}, CrawledAt: time.Now().UTC()},
		{URL: marker, Data: map[string]any{"k": "v2"}, CrawledAt: time.Now().UTC()},
		{URL: marker, Data: map[string]any{"k": "v3"}, CrawledAt: time.Now().UTC()},
	}
	if err := p.Store(ctx, items); err != nil {
		t.Fatalf("store small batch: %v", err)
	}

	var count int
	row := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM crawled_items WHERE url = $1", marker)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}

	if _, err := p.pool.Exec(ctx, "DELETE FROM crawled_items WHERE url = $1", marker); err != nil {
		t.Logf("cleanup warning: %v", err)
	}
}

func TestPostgresPlugin_InitFromConfig(t *testing.T) {
	dsn := postgresTestDSN(t)

	p := NewPostgresPlugin(PostgresConfig{})
	if err := p.Init(map[string]any{"dsn": dsn}); err != nil {
		t.Fatalf("init from config: %v", err)
	}
	defer p.Close()
}

func TestPostgresPlugin_StoreNotInitialized(t *testing.T) {
	p := NewPostgresPlugin(PostgresConfig{DSN: "unused"})
	err := p.Store(context.Background(), testItems())
	if err == nil {
		t.Error("expected error for uninitialized storage")
	}
}

func TestPostgresPlugin_InitMissingDSN(t *testing.T) {
	p := NewPostgresPlugin(PostgresConfig{})
	if err := p.Init(nil); err == nil {
		t.Error("expected error for missing DSN")
	}
}
