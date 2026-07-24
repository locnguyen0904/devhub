// Package cache owns the Redis client.
package cache

import (
	"context"
	"errors"
	"fmt"
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

// Close shuts the client down.
func (c *Client) Close() error {
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}
	return nil
}
