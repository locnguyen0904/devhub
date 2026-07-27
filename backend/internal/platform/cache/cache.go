// Package cache owns the Redis client.
package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client so callers depend on this package instead of
// go-redis directly.
type Client struct {
	rdb *redis.Client
}

// New builds a Redis client from a connection URL. Like database.New it does
// not connect eagerly, keeping startup usable without a live Redis.
func New(url string) (*Client, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return &Client{rdb: redis.NewClient(opt)}, nil
}

// Ping reports whether Redis answers.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}
	return nil
}

// Set stores a value with an expiry.
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

// Get reads a key, returning found=false when it is absent.
func (c *Client) Get(ctx context.Context, key string) (value string, found bool, err error) {
	v, err := c.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get %s: %w", key, err)
	}
	return v, true, nil
}

// GetDel reads a key and deletes it in one atomic step, returning found=false
// when the key is absent. Single-use tokens like OAuth state rely on the
// atomicity: two callbacks racing on the same state cannot both succeed.
func (c *Client) GetDel(ctx context.Context, key string) (value string, found bool, err error) {
	v, err := c.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("getdel %s: %w", key, err)
	}
	return v, true, nil
}

// HIncrBy adds n to a field of a hash, creating either as needed. Used to
// buffer high-frequency counters (like post views) in Redis before flushing.
func (c *Client) HIncrBy(ctx context.Context, key, field string, n int64) error {
	if err := c.rdb.HIncrBy(ctx, key, field, n).Err(); err != nil {
		return fmt.Errorf("hincrby %s: %w", key, err)
	}
	return nil
}

// DrainHash reads every field of a hash as an int64 and deletes the hash. A view
// arriving between the read and the delete is lost, which is acceptable for a
// view counter (docs/01-architecture.md §8 tolerates ~60s of drift).
func (c *Client) DrainHash(ctx context.Context, key string) (map[string]int64, error) {
	raw, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall %s: %w", key, err)
	}
	if len(raw) == 0 {
		return map[string]int64{}, nil
	}
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return nil, fmt.Errorf("del %s: %w", key, err)
	}

	out := make(map[string]int64, len(raw))
	for field, val := range raw {
		n, perr := strconv.ParseInt(val, 10, 64)
		if perr != nil {
			continue // skip a corrupt field rather than failing the whole flush
		}
		out[field] = n
	}
	return out, nil
}

// Close shuts the client down.
func (c *Client) Close() error {
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}
	return nil
}
