package frontier

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds configuration for the Redis-backed frontier.
type RedisConfig struct {
	// Addr is the Redis server address. Default: "localhost:6379".
	Addr string

	// Password is the Redis AUTH password. Optional.
	Password string

	// DB is the Redis database number. Default: 0.
	DB int

	// KeyPrefix is prepended to all Redis keys. Default: "frontier".
	KeyPrefix string
}

func (c RedisConfig) withDefaults() RedisConfig {
	if c.Addr == "" {
		c.Addr = "localhost:6379"
	}
	if c.KeyPrefix == "" {
		c.KeyPrefix = "frontier"
	}
	return c
}

// RedisFrontier is a Redis-backed implementation of the Frontier interface.
// URLs are stored in a sorted set (ZSET); the score encodes priority and
// discovery time so that BZPOPMIN returns higher-priority, older URLs first.
//
// Score formula:
//
//	score = float64(priority * 1e13 + discoveredAt.UnixMilli())
//
// Priority bands are separated by 10 trillion; within each band earlier
// discoveries have lower scores and are therefore returned first.
type RedisFrontier struct {
	cfg    RedisConfig
	client *redis.Client
	key    string
	size   atomic.Int64
	closed atomic.Bool
}

// NewRedisFrontier creates a RedisFrontier and pings the server to verify
// connectivity. Returns an error if Redis is unreachable.
func NewRedisFrontier(cfg RedisConfig) (*RedisFrontier, error) {
	cfg = cfg.withDefaults()

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("redis frontier: ping %s: %w", cfg.Addr, err)
	}

	f := &RedisFrontier{
		cfg:    cfg,
		client: rdb,
		key:    cfg.KeyPrefix + ":urls",
	}

	// Sync the local size counter from existing queue entries.
	n, err := rdb.ZCard(context.Background(), f.key).Result()
	if err == nil {
		f.size.Store(n)
	}

	return f, nil
}

// Add enqueues a URL entry. The URL is canonicalized before storage.
// Returns ErrDuplicate if the URL is already in the queue (via ZADD NX).
func (f *RedisFrontier) Add(entry *URLEntry) error {
	if entry == nil {
		return ErrNilEntry
	}
	if entry.URL == "" {
		return ErrEmptyURL
	}
	if f.closed.Load() {
		return ErrClosed
	}

	entry.URL = Canonicalize(entry.URL)
	if entry.DiscoveredAt.IsZero() {
		entry.DiscoveredAt = time.Now()
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("redis frontier: marshal entry: %w", err)
	}

	score := scoreFor(entry.Priority, entry.DiscoveredAt)

	// NX: only add if the member does not already exist.
	added, err := f.client.ZAddNX(context.Background(), f.key, redis.Z{
		Score:  score,
		Member: string(payload),
	}).Result()
	if err != nil {
		return fmt.Errorf("redis frontier: ZADD: %w", err)
	}
	if added == 0 {
		return ErrDuplicate
	}

	f.size.Add(1)
	return nil
}

// Next blocks until a URL is available or the context is cancelled.
// It polls using BZPOPMIN with a 1-second timeout to respect context
// cancellation without busy-looping.
func (f *RedisFrontier) Next(ctx context.Context) (*URLEntry, error) {
	for {
		if f.closed.Load() {
			return nil, ErrClosed
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		results, err := f.client.BZPopMin(ctx, time.Second, f.key).Result()
		if err != nil {
			if err == redis.Nil {
				// Timeout with no items; loop and check context.
				continue
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("redis frontier: BZPOPMIN: %w", err)
		}

		member, ok := results.Member.(string)
		if !ok {
			continue
		}

		var entry URLEntry
		if err := json.Unmarshal([]byte(member), &entry); err != nil {
			// Skip malformed entries.
			continue
		}

		f.size.Add(-1)
		return &entry, nil
	}
}

// Size returns the number of URLs currently in the Redis queue.
func (f *RedisFrontier) Size() int64 {
	return f.size.Load()
}

// Close marks the frontier as closed and disconnects from Redis.
func (f *RedisFrontier) Close() error {
	f.closed.Store(true)
	return f.client.Close()
}

// scoreFor converts a priority and discovery time into a Redis ZSET score.
// Lower scores are dequeued first by BZPOPMIN.
func scoreFor(p Priority, t time.Time) float64 {
	return float64(int64(p)*int64(1e13) + t.UnixMilli())
}
