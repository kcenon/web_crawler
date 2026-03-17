package storage

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// PostgresConfig holds connection settings for the PostgreSQL storage plugin.
type PostgresConfig struct {
	// DSN is the PostgreSQL connection string (e.g. "postgres://user:pass@host/db").
	// Required.
	DSN string

	// BatchSize is the minimum number of items to trigger a CopyFrom batch insert.
	// Defaults to 10.
	BatchSize int
}

// PostgresPlugin stores crawled items in a PostgreSQL database.
// It runs schema migrations on Init and uses pgx CopyFrom for bulk inserts.
type PostgresPlugin struct {
	cfg  PostgresConfig
	pool *pgxpool.Pool
}

// NewPostgresPlugin creates a new PostgresPlugin with the given configuration.
func NewPostgresPlugin(cfg PostgresConfig) *PostgresPlugin {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	return &PostgresPlugin{cfg: cfg}
}

// Name returns the plugin identifier for this storage backend.
func (p *PostgresPlugin) Name() string { return "postgres" }

// Init connects to PostgreSQL and runs pending migrations.
func (p *PostgresPlugin) Init(config map[string]any) error {
	if dsn, ok := config["dsn"].(string); ok && dsn != "" {
		p.cfg.DSN = dsn
	}
	if p.cfg.DSN == "" {
		return fmt.Errorf("postgres storage: dsn is required")
	}

	pool, err := pgxpool.New(context.Background(), p.cfg.DSN)
	if err != nil {
		return fmt.Errorf("postgres storage: connect: %w", err)
	}
	p.pool = pool

	if err := p.runMigrations(context.Background()); err != nil {
		pool.Close()
		p.pool = nil
		return fmt.Errorf("postgres storage: migrate: %w", err)
	}
	return nil
}

// Store persists items to the crawled_items table.
// When len(items) >= BatchSize, it uses pgx CopyFrom for high throughput.
// Smaller batches fall back to a single multi-row INSERT.
func (p *PostgresPlugin) Store(ctx context.Context, items []Item) error {
	if p.pool == nil {
		return fmt.Errorf("postgres storage: not initialized")
	}
	if len(items) == 0 {
		return nil
	}

	if len(items) >= p.cfg.BatchSize {
		return p.copyFrom(ctx, items)
	}
	return p.insertBatch(ctx, items)
}

// Close releases the connection pool.
func (p *PostgresPlugin) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

// runMigrations creates the schema_migrations tracking table and applies
// any up-migrations that have not yet been executed.
func (p *PostgresPlugin) runMigrations(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Collect and sort up-migrations by version.
	var upFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			upFiles = append(upFiles, e.Name())
		}
	}
	sort.Strings(upFiles)

	for _, name := range upFiles {
		version := strings.TrimSuffix(name, ".up.sql")

		var applied bool
		row := p.pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version)
		if err := row.Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		sql, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := p.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := p.pool.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("record migration %s: %w", name, err)
		}
	}
	return nil
}

// copyFrom uses the PostgreSQL COPY protocol for high-throughput bulk insert.
func (p *PostgresPlugin) copyFrom(ctx context.Context, items []Item) error {
	cols := []string{"url", "data", "metadata", "crawled_at"}

	rows := make([][]any, 0, len(items))
	for _, item := range items {
		dataJSON, err := toJSON(item.Data)
		if err != nil {
			return fmt.Errorf("postgres storage: marshal data: %w", err)
		}
		metaJSON, err := toJSON(item.Metadata)
		if err != nil {
			return fmt.Errorf("postgres storage: marshal metadata: %w", err)
		}
		rows = append(rows, []any{item.URL, dataJSON, metaJSON, item.CrawledAt})
	}

	_, err := p.pool.CopyFrom(
		ctx,
		pgx.Identifier{"crawled_items"},
		cols,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return fmt.Errorf("postgres storage: copy from: %w", err)
	}
	return nil
}

// insertBatch inserts a small batch using a parameterised multi-row INSERT.
func (p *PostgresPlugin) insertBatch(ctx context.Context, items []Item) error {
	const base = "INSERT INTO crawled_items (url, data, metadata, crawled_at) VALUES "
	var (
		placeholders []string
		args         []any
	)
	for i, item := range items {
		n := i * 4
		placeholders = append(placeholders,
			fmt.Sprintf("($%d, $%d, $%d, $%d)", n+1, n+2, n+3, n+4))

		dataJSON, err := toJSON(item.Data)
		if err != nil {
			return fmt.Errorf("postgres storage: marshal data: %w", err)
		}
		metaJSON, err := toJSON(item.Metadata)
		if err != nil {
			return fmt.Errorf("postgres storage: marshal metadata: %w", err)
		}
		args = append(args, item.URL, dataJSON, metaJSON, item.CrawledAt)
	}

	query := base + strings.Join(placeholders, ", ")
	_, err := p.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("postgres storage: insert: %w", err)
	}
	return nil
}

// toJSON marshals v to a JSON byte slice. Nil maps become nil (NULL in SQL).
func toJSON(v map[string]any) ([]byte, error) {
	if len(v) == 0 {
		return nil, nil //nolint:nilnil // NULL is intentional for empty JSONB
	}
	return json.Marshal(v)
}
