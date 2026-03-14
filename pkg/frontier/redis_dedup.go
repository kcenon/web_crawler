package frontier

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisDeduplicator tracks seen URLs using a Redis Set.
// It is safe for concurrent use across multiple processes.
//
// SADD returns 1 when the member is new and 0 when already present,
// providing an atomic check-and-set in a single round trip.
type RedisDeduplicator struct {
	client *redis.Client
	key    string
}

// NewRedisDeduplicator creates a RedisDeduplicator and verifies connectivity.
func NewRedisDeduplicator(cfg RedisConfig) (*RedisDeduplicator, error) {
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
		return nil, fmt.Errorf("redis deduplicator: ping %s: %w", cfg.Addr, err)
	}

	return &RedisDeduplicator{
		client: rdb,
		key:    cfg.KeyPrefix + ":seen",
	}, nil
}

// IsSeen returns true if the URL has already been recorded.
func (d *RedisDeduplicator) IsSeen(url string) bool {
	n, err := d.client.SIsMember(context.Background(), d.key, url).Result()
	return err == nil && n
}

// MarkSeen records the URL as seen. Returns true if the URL was new,
// false if it was already seen. The operation is atomic.
func (d *RedisDeduplicator) MarkSeen(url string) bool {
	n, err := d.client.SAdd(context.Background(), d.key, url).Result()
	return err == nil && n == 1
}

// Size returns the number of unique URLs recorded.
func (d *RedisDeduplicator) Size() int64 {
	n, err := d.client.SCard(context.Background(), d.key).Result()
	if err != nil {
		return 0
	}
	return n
}

// Reset removes all seen URLs from the Redis Set.
func (d *RedisDeduplicator) Reset() {
	d.client.Del(context.Background(), d.key)
}

// Close disconnects from Redis.
func (d *RedisDeduplicator) Close() error {
	return d.client.Close()
}
