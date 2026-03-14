//go:build integration

package storage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// startPostgresContainer starts a PostgreSQL 15 container and returns its DSN.
func startPostgresContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("start Postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("terminate Postgres container: %v", err)
		}
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("get mapped port: %v", err)
	}

	return fmt.Sprintf("postgres://postgres:password@%s:%s/testdb?sslmode=disable", host, port.Port())
}

func TestPostgresPlugin_Integration_InitAndMigrate(t *testing.T) {
	dsn := startPostgresContainer(t)

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

func TestPostgresPlugin_Integration_Store100Items(t *testing.T) {
	dsn := startPostgresContainer(t)

	p := NewPostgresPlugin(PostgresConfig{DSN: dsn, BatchSize: 10})
	if err := p.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer p.Close()

	ctx := context.Background()

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

	if err := p.Store(ctx, items); err != nil {
		t.Fatalf("store: %v", err)
	}

	var count int
	row := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM crawled_items WHERE url LIKE 'https://test-item-%'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 100 {
		t.Errorf("expected 100 rows, got %d", count)
	}
}

func TestPostgresPlugin_Integration_SmallBatchInsert(t *testing.T) {
	dsn := startPostgresContainer(t)

	p := NewPostgresPlugin(PostgresConfig{DSN: dsn, BatchSize: 10})
	if err := p.Init(nil); err != nil {
		t.Fatalf("init: %v", err)
	}
	defer p.Close()

	ctx := context.Background()

	// Store 3 items (< BatchSize=10) → uses insertBatch.
	items := []Item{
		{URL: "https://small-1.example.com", Data: map[string]any{"k": "v1"}, CrawledAt: time.Now().UTC()},
		{URL: "https://small-2.example.com", Data: map[string]any{"k": "v2"}, CrawledAt: time.Now().UTC()},
		{URL: "https://small-3.example.com", Data: map[string]any{"k": "v3"}, CrawledAt: time.Now().UTC()},
	}
	if err := p.Store(ctx, items); err != nil {
		t.Fatalf("store small batch: %v", err)
	}

	var count int
	row := p.pool.QueryRow(ctx, "SELECT COUNT(*) FROM crawled_items WHERE url LIKE 'https://small-%.example.com'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 rows, got %d", count)
	}
}

func TestPostgresPlugin_Integration_InitFromConfig(t *testing.T) {
	dsn := startPostgresContainer(t)

	p := NewPostgresPlugin(PostgresConfig{})
	if err := p.Init(map[string]any{"dsn": dsn}); err != nil {
		t.Fatalf("init from config: %v", err)
	}
	defer p.Close()
}

func TestPostgresPlugin_Integration_StoreNotInitialized(t *testing.T) {
	p := NewPostgresPlugin(PostgresConfig{DSN: "unused"})
	err := p.Store(context.Background(), testItems())
	if err == nil {
		t.Error("expected error for uninitialized storage")
	}
}

func TestPostgresPlugin_Integration_InitMissingDSN(t *testing.T) {
	p := NewPostgresPlugin(PostgresConfig{})
	if err := p.Init(nil); err == nil {
		t.Error("expected error for missing DSN")
	}
}
